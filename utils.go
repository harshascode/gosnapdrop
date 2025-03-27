package main

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/mssola/user_agent"
)

var colors = []string{"Red", "Blue", "Green", "Yellow", "Purple", "Orange", "Pink"}
var animals = []string{"Dog", "Cat", "Elephant", "Lion", "Tiger", "Bear", "Penguin"}

func generateUUID() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

func generateDisplayName(seed string) string {
	// Simple deterministic name generator
	seedInt := int64(hashString(seed))
	colorIndex := abs(seedInt) % int64(len(colors))
	animalIndex := abs(seedInt) % int64(len(animals))

	return fmt.Sprintf("%s %s", colors[colorIndex], animals[animalIndex])
}

func (p *Peer) setName(userAgentString string) {
	ua := user_agent.New(userAgentString)
	osInfo := ua.OS()
	browserName, _ := ua.Browser() // Ignore browser version

	// Sanitize device name
	deviceName := strings.TrimSpace(osInfo)
	deviceName = strings.ReplaceAll(deviceName, "Mac OS", "Mac")
	
	if ua.Mobile() {
		deviceName += " Mobile"
	} else if browserName != "" {
		deviceName += " " + browserName
	}

	if deviceName == "" {
		deviceName = "Unknown Device"
	}

	// Check for WebRTC support in modern browsers
	p.RTCSupported = isWebRTCSupported(browserName)

	p.Name = PeerName{
		OS:          osInfo,
		Browser:     browserName,
		DeviceName:  deviceName,
		DisplayName: generateDisplayName(p.ID),
	}
}

func isWebRTCSupported(browserName string) bool {
	supportedBrowsers := map[string]bool{
		"Chrome":  true,
		"Firefox": true,
		"Safari":  true,
		"Edge":    true,
	}
	
	return supportedBrowsers[browserName] || strings.Contains(browserName, "Edge")
}

func hashString(s string) int64 {
	var hash int64
	for _, c := range s {
		hash = ((hash << 5) - hash) + int64(c)
		hash = hash & 0x7FFFFFFFFFFFFFFF // Convert to positive 64bit integer
	}
	return hash
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func (p *Peer) getInfo() map[string]interface{} {
	return map[string]interface{}{
		"id": p.ID,
		"name": map[string]interface{}{
			"model":       p.Name.Model,
			"os":         p.Name.OS,
			"browser":    p.Name.Browser,
			"type":       p.Name.Type,
			"deviceName": p.Name.DeviceName,
			"displayName": p.Name.DisplayName,
		},
		"rtcSupported": p.RTCSupported,
	}
}
