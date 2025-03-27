package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

type Peer struct {
	ID           string
	IP           string
	Socket       *websocket.Conn
	RTCSupported bool
	Name         PeerName
	LastBeat     time.Time
	TimerID      *time.Timer
}

type PeerName struct {
	Model       string `json:"model"`
	OS          string `json:"os"`
	Browser     string `json:"browser"`
	Type        string `json:"type"`
	DeviceName  string `json:"deviceName"`
	DisplayName string `json:"displayName"`
}

type SnapdropServer struct {
	rooms    map[string]map[string]*Peer
	upgrader websocket.Upgrader
	mutex    sync.RWMutex
	done     chan struct{}    // For cleanup signaling
	wg       sync.WaitGroup   // For graceful shutdown
}

func main() {
	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("Shutdown signal received, exiting...")
		os.Exit(0)
	}()

	// Rate limiter setup
	rate := limiter.Rate{
		Period: 5 * time.Minute,
		Limit:  100,
	}
	store := memory.NewStore()
	rateLimiter := limiter.New(store, rate)

	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Server setup
	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(rateLimitMiddleware(rateLimiter))

	server := &SnapdropServer{
		rooms: make(map[string]map[string]*Peer),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		done: make(chan struct{}),
	}

	// Graceful shutdown handler
	go func() {
		<-sigChan
		fmt.Println("Shutdown signal received, cleaning up...")
		close(server.done)
		server.wg.Wait()
		os.Exit(0)
	}()

	// WebSocket endpoints
	router.GET("/ws", server.handleWebSocket)
	router.GET("/server/webrtc", server.handleWebSocket)

	// Serve static files
	router.StaticFile("/", "./public/index.html")
	router.StaticFile("/service-worker.js", "./public/service-worker.js")
	router.Static("/images", "./public/images")
	router.Static("/sounds", "./public/sounds")
	router.Static("/scripts", "./public/scripts")
	router.StaticFile("/styles.css", "./public/styles.css")
	router.StaticFile("/manifest.json", "./public/manifest.json")

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Start server
	var addr string
	if len(os.Args) > 1 && os.Args[1] == "public" {
		addr = ":" + port
	} else {
		addr = "0.0.0.0:" + port
	}

	log.Printf("Snapdrop is running on port %s", port)
	if err := router.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func rateLimitMiddleware(limiter *limiter.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ctx := c.Request.Context()
		result, err := limiter.Get(ctx, ip)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if result.Reached {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests from this IP Address, please try again after 5 minutes.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *SnapdropServer) handleWebSocket(c *gin.Context) {
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	if ip := c.ClientIP(); ip == "" {
		log.Println("Invalid client IP")
		conn.Close()
		return
	}

	s.wg.Add(1)
	peer := &Peer{
		ID:           generateUUID(),
		IP:           c.ClientIP(),
		Socket:       conn,
		RTCSupported: strings.Contains(c.Request.URL.Path, "webrtc"),
		LastBeat:     time.Now(),
	}

	peer.setName(c.Request.Header.Get("User-Agent"))
	s.onConnection(peer)

	// Handle incoming messages
	go s.handleMessages(peer)
}

func (s *SnapdropServer) onConnection(peer *Peer) {
	s.mutex.Lock()
	s.joinRoom(peer)
	s.mutex.Unlock()

	// Send display name
	s.send(peer, map[string]interface{}{
		"type": "display-name",
		"message": map[string]string{
			"displayName": peer.Name.DisplayName,
			"deviceName":  peer.Name.DeviceName,
		},
	})

	s.keepAlive(peer)
}

func (s *SnapdropServer) handleMessages(peer *Peer) {
	defer s.wg.Done()
	defer s.leaveRoom(peer)

	for {
		select {
		case <-s.done:
			return
		default:
			messageType, message, err := peer.Socket.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("Read error: %v", err)
				}
				return
			}

			if messageType != websocket.TextMessage {
				continue
			}

			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("JSON parse error: %v", err)
				continue
			}

			msgType, ok := msg["type"].(string)
			if !ok {
				continue
			}

			switch msgType {
			case "disconnect":
				return
			case "pong":
				peer.LastBeat = time.Now()
			default:
				s.handleRelayMessage(peer, msg)
			}
		}
	}
}

func (s *SnapdropServer) handleRelayMessage(peer *Peer, msg map[string]interface{}) {
	to, ok := msg["to"].(string)
	if !ok {
		return
	}

	s.mutex.RLock()
	recipient, exists := s.rooms[peer.IP][to]
	s.mutex.RUnlock()

	if exists {
		delete(msg, "to")
		msg["sender"] = peer.ID
		s.send(recipient, msg)
	}
}

func (s *SnapdropServer) joinRoom(peer *Peer) {
	// Create room if it doesn't exist
	if s.rooms[peer.IP] == nil {
		s.rooms[peer.IP] = make(map[string]*Peer)
	}

	// Notify other peers
	for _, otherPeer := range s.rooms[peer.IP] {
		s.send(otherPeer, map[string]interface{}{
			"type": "peer-joined",
			"peer": peer.getInfo(),
		})
	}

	// Send existing peers to new peer
	otherPeers := make([]map[string]interface{}, 0)
	for _, otherPeer := range s.rooms[peer.IP] {
		otherPeers = append(otherPeers, otherPeer.getInfo())
	}
	s.send(peer, map[string]interface{}{
		"type":  "peers",
		"peers": otherPeers,
	})

	// Add peer to room
	s.rooms[peer.IP][peer.ID] = peer
}

func (s *SnapdropServer) leaveRoom(peer *Peer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.rooms[peer.IP] == nil || s.rooms[peer.IP][peer.ID] == nil {
		return
	}

	s.cancelKeepAlive(peer)
	delete(s.rooms[peer.IP], peer.ID)
	peer.Socket.Close()

	if len(s.rooms[peer.IP]) == 0 {
		delete(s.rooms, peer.IP)
	} else {
		for _, otherPeer := range s.rooms[peer.IP] {
			s.send(otherPeer, map[string]interface{}{
				"type":   "peer-left",
				"peerId": peer.ID,
			})
		}
	}
}

func (s *SnapdropServer) send(peer *Peer, message interface{}) {
	if peer == nil || peer.Socket == nil {
		return
	}
	
	err := peer.Socket.WriteJSON(message)
	if err != nil {
		log.Printf("Send error: %v", err)
	}
}

func (s *SnapdropServer) keepAlive(peer *Peer) {
	s.cancelKeepAlive(peer)
	timeout := 30 * time.Second

	if time.Since(peer.LastBeat) > 2*timeout {
		s.leaveRoom(peer)
		return
	}

	s.send(peer, map[string]string{"type": "ping"})
	peer.TimerID = time.AfterFunc(timeout, func() {
		s.keepAlive(peer)
	})
}

func (s *SnapdropServer) cancelKeepAlive(peer *Peer) {
	if peer != nil && peer.TimerID != nil {
		peer.TimerID.Stop()
		peer.TimerID = nil
	}
}
