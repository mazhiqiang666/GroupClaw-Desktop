//go:build !windows

package mockchat

import (
	"fmt"
	"log"
)

// MockChatUI manages the GUI for the mock chat application (stub for non-Windows platforms)
type MockChatUI struct {
	app         *MockChatApp
	running     bool
}

// NewMockChatUI creates a new UI instance
func NewMockChatUI(app *MockChatApp) *MockChatUI {
	return &MockChatUI{
		app:     app,
		running: false,
	}
}

// Initialize initializes the UI (stub for non-Windows platforms)
func (ui *MockChatUI) Initialize() error {
	// Stub implementation for non-Windows platforms
	log.Println("MockChatUI.Initialize() called on non-Windows platform - using stub implementation")
	return nil
}

// Run starts the GUI (stub for non-Windows platforms)
func (ui *MockChatUI) Run() {
	// Stub implementation for non-Windows platforms
	log.Println("MockChatUI.Run() called on non-Windows platform - using stub implementation")

	// Start HTTP server in background
	go func() {
		if err := ui.app.StartHTTPServer(":8081"); err != nil {
			log.Printf("HTTP server failed: %v", err)
		}
	}()

	// Keep running
	select {}
}

// GetWindowHandle returns the window handle (stub for non-Windows platforms)
func (ui *MockChatUI) GetWindowHandle() uintptr {
	// Stub implementation - return 0 for non-Windows platforms
	return 0
}

// GetUIAMode returns the current UIA mode
func (ui *MockChatUI) GetUIAMode() UIAMode {
	return ui.app.uiaMode
}

// Close closes the UI (stub for non-Windows platforms)
func (ui *MockChatUI) Close() {
	ui.running = false
}
