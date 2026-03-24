//go:build windows

package windows

import (
	"testing"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
)

func TestNewBridge(t *testing.T) {
	bridge := NewBridge()
	if bridge == nil {
		t.Error("NewBridge should return a non-nil bridge")
	}
}

func TestBridge_Initialize(t *testing.T) {
	bridge := NewBridge()
	result := bridge.Initialize()

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Initialize should succeed on Windows platform, got status: %v, error: %v", result.Status, result.Error)
	}

	if result.ReasonCode != adapter.ReasonOK {
		t.Errorf("Expected reason code OK, got %v", result.ReasonCode)
	}
}

func TestBridge_Initialize_AlreadyInitialized(t *testing.T) {
	bridge := NewBridge()
	result1 := bridge.Initialize()
	if result1.Status != adapter.StatusSuccess {
		t.Errorf("First Initialize should succeed, got status: %v", result1.Status)
	}

	result2 := bridge.Initialize()
	if result2.Status != adapter.StatusSuccess {
		t.Errorf("Second Initialize should succeed (idempotent), got status: %v", result2.Status)
	}
}

func TestBridge_FindTopLevelWindows_Initialized(t *testing.T) {
	bridge := NewBridge()
	bridge.Initialize()

	handles, result := bridge.FindTopLevelWindows("", "微信")

	// Should succeed even if window not found (returns empty list)
	if result.Status != adapter.StatusSuccess {
		t.Errorf("FindTopLevelWindows should succeed, got status: %v, error: %v", result.Status, result.Error)
	}

	// handles can be empty if no window found
	if handles == nil {
		t.Error("FindTopLevelWindows should return a slice (possibly empty), not nil")
	}
}

func TestBridge_FindTopLevelWindows_NotInitialized(t *testing.T) {
	bridge := NewBridge()

	handles, result := bridge.FindTopLevelWindows("", "微信")

	if result.Status != adapter.StatusFailed {
		t.Errorf("FindTopLevelWindows should fail when not initialized, got status: %v", result.Status)
	}

	if handles != nil {
		t.Error("FindTopLevelWindows should return nil handles when not initialized")
	}

	if result.ReasonCode != adapter.ReasonCode("NOT_INITIALIZED") {
		t.Errorf("Expected reason code NOT_INITIALIZED, got %v", result.ReasonCode)
	}
}

func TestBridge_GetWindowInfo_InvalidHandle(t *testing.T) {
	bridge := NewBridge()

	info, result := bridge.GetWindowInfo(0)

	if result.Status != adapter.StatusFailed {
		t.Errorf("GetWindowInfo should fail with invalid handle, got status: %v", result.Status)
	}

	if info.Handle != 0 {
		t.Error("GetWindowInfo should return empty WindowInfo with invalid handle")
	}

	if result.ReasonCode != adapter.ReasonCode("INVALID_HANDLE") {
		t.Errorf("Expected reason code INVALID_HANDLE, got %v", result.ReasonCode)
	}
}

func TestBridge_FocusWindow_InvalidHandle(t *testing.T) {
	bridge := NewBridge()

	result := bridge.FocusWindow(0)

	if result.Status != adapter.StatusFailed {
		t.Errorf("FocusWindow should fail with invalid handle, got status: %v", result.Status)
	}

	if result.ReasonCode != adapter.ReasonCode("INVALID_HANDLE") {
		t.Errorf("Expected reason code INVALID_HANDLE, got %v", result.ReasonCode)
	}
}

func TestBridge_Release(t *testing.T) {
	bridge := NewBridge()
	bridge.Initialize()

	// Release should not panic
	bridge.Release()
}

func TestBridge_GetWindowText_InvalidHandle(t *testing.T) {
	bridge := NewBridge()

	text, result := bridge.GetWindowText(0)

	if result.Status != adapter.StatusFailed {
		t.Errorf("GetWindowText should fail with invalid handle, got status: %v", result.Status)
	}

	if text != "" {
		t.Error("GetWindowText should return empty string with invalid handle")
	}
}

func TestBridge_GetWindowClass_InvalidHandle(t *testing.T) {
	bridge := NewBridge()

	class, result := bridge.GetWindowClass(0)

	if result.Status != adapter.StatusFailed {
		t.Errorf("GetWindowClass should fail with invalid handle, got status: %v", result.Status)
	}

	if class != "" {
		t.Error("GetWindowClass should return empty string with invalid handle")
	}
}

func TestBridge_SendKeys_InvalidHandle(t *testing.T) {
	bridge := NewBridge()

	result := bridge.SendKeys(0, "test")

	if result.Status != adapter.StatusFailed {
		t.Errorf("SendKeys should fail with invalid handle, got status: %v", result.Status)
	}

	if result.ReasonCode != adapter.ReasonCode("INVALID_HANDLE") {
		t.Errorf("Expected reason code INVALID_HANDLE, got %v", result.ReasonCode)
	}
}

func TestBridge_SendKeys_NotInitialized(t *testing.T) {
	bridge := NewBridge()

	result := bridge.SendKeys(12345, "test")

	if result.Status != adapter.StatusFailed {
		t.Errorf("SendKeys should fail when not initialized, got status: %v", result.Status)
	}

	if result.ReasonCode != adapter.ReasonCode("NOT_INITIALIZED") {
		t.Errorf("Expected reason code NOT_INITIALIZED, got %v", result.ReasonCode)
	}
}

func TestBridge_Click_InvalidHandle(t *testing.T) {
	bridge := NewBridge()

	result := bridge.Click(0, 100, 200)

	if result.Status != adapter.StatusFailed {
		t.Errorf("Click should fail with invalid handle, got status: %v", result.Status)
	}

	if result.ReasonCode != adapter.ReasonCode("INVALID_HANDLE") {
		t.Errorf("Expected reason code INVALID_HANDLE, got %v", result.ReasonCode)
	}
}

func TestBridge_Click_NotInitialized(t *testing.T) {
	bridge := NewBridge()

	result := bridge.Click(12345, 100, 200)

	if result.Status != adapter.StatusFailed {
		t.Errorf("Click should fail when not initialized, got status: %v", result.Status)
	}

	if result.ReasonCode != adapter.ReasonCode("NOT_INITIALIZED") {
		t.Errorf("Expected reason code NOT_INITIALIZED, got %v", result.ReasonCode)
	}
}

func TestBridge_SetClipboardText(t *testing.T) {
	bridge := NewBridge()
	bridge.Initialize()

	result := bridge.SetClipboardText("test clipboard text")

	if result.Status != adapter.StatusSuccess {
		t.Errorf("SetClipboardText should succeed, got status: %v, error: %v", result.Status, result.Error)
	}
}

func TestBridge_GetClipboardText(t *testing.T) {
	bridge := NewBridge()
	bridge.Initialize()

	// Set text first
	setResult := bridge.SetClipboardText("test clipboard text")
	if setResult.Status != adapter.StatusSuccess {
		t.Errorf("SetClipboardText should succeed, got status: %v", setResult.Status)
	}

	// Get text
	text, result := bridge.GetClipboardText()

	if result.Status != adapter.StatusSuccess {
		t.Errorf("GetClipboardText should succeed, got status: %v, error: %v", result.Status, result.Error)
	}

	if text != "test clipboard text" {
		t.Errorf("GetClipboardText should return 'test clipboard text', got: %v", text)
	}
}

func TestBridge_EnumerateAccessibleNodes_InvalidHandle(t *testing.T) {
	bridge := NewBridge()
	bridge.Initialize()

	nodes, result := bridge.EnumerateAccessibleNodes(0)

	// Should succeed even with invalid handle (returns empty list)
	if result.Status != adapter.StatusSuccess {
		t.Errorf("EnumerateAccessibleNodes should succeed, got status: %v, error: %v", result.Status, result.Error)
	}

	if nodes == nil {
		t.Error("EnumerateAccessibleNodes should return a slice (possibly empty), not nil")
	}
}

func TestBridge_CaptureWindow_InvalidHandle(t *testing.T) {
	bridge := NewBridge()

	data, result := bridge.CaptureWindow(0)

	if result.Status != adapter.StatusFailed {
		t.Errorf("CaptureWindow should fail with invalid handle, got status: %v", result.Status)
	}

	if data != nil {
		t.Error("CaptureWindow should return nil data with invalid handle")
	}

	if result.ReasonCode != adapter.ReasonCode("INVALID_HANDLE") {
		t.Errorf("Expected reason code INVALID_HANDLE, got %v", result.ReasonCode)
	}
}
