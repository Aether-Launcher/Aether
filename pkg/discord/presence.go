package discord

import (
	"fmt"
	"sync"
	"time"

	"github.com/hugolgst/rich-go/client"
)

var (
	mu        sync.Mutex
	connected bool
)

// ensureConnected logs in if not already connected.
// No-op if Discord is not running.
func ensureConnected() error {
	mu.Lock()
	defer mu.Unlock()
	if connected {
		return nil
	}
	if err := client.Login(ClientID); err != nil {
		return fmt.Errorf("discord login failed: %w", err)
	}
	connected = true
	return nil
}

// SetActivity updates the Rich Presence.
// details: instance name, state: "1.21.1 • fabric", start: when launch began.
func SetActivity(details, state, largeImage, largeText, smallImage, smallText string, start *time.Time) error {
	// placeholder ID means build hasn't been configured yet — no-op
	if ClientID == "000000000000000000" {
		return nil
	}
	if err := ensureConnected(); err != nil {
		return err
	}

	activity := client.Activity{
		Details:    details,
		State:      state,
		LargeImage: largeImage,
		LargeText:  largeText,
		SmallImage: smallImage,
		SmallText:  smallText,
		Buttons: []*client.Button{
			{Label: "Join Aether Discord", Url: "https://discord.gg/pQc9NnGhpG"},
		},
	}
	if start != nil {
		activity.Timestamps = &client.Timestamps{Start: start}
	}

	mu.Lock()
	defer mu.Unlock()
	return client.SetActivity(activity)
}

// Clear removes the presence (back to idle).
func Clear() error {
	mu.Lock()
	wasConnected := connected
	mu.Unlock()
	if !wasConnected {
		return nil
	}
	client.Logout()
	mu.Lock()
	connected = false
	mu.Unlock()
	return nil
}

// Shutdown logs out and marks disconnected.
func Shutdown() {
	mu.Lock()
	wasConnected := connected
	mu.Unlock()
	if wasConnected {
		client.Logout()
		mu.Lock()
		connected = false
		mu.Unlock()
	}
}
