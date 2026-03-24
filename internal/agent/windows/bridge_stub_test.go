//go:build !windows

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

	if result.Status != adapter.StatusFailed {
		t.Errorf("Initialize should fail on non-Windows platform, got status: %v", result.Status)
	}

	if result.ReasonCode != adapter.ReasonCode("PLATFORM_NOT_SUPPORTED") {
		t.Errorf("Expected reason code PLATFORM_NOT_SUPPORTED, got %v", result.ReasonCode)
	}
}

func TestBridge_FindTopLevelWindows(t *testing.T) {
	bridge := NewBridge()
	handles, result := bridge.FindTopLevelWindows("", "微信")

	if result.Status != adapter.StatusFailed {
		t.Errorf("FindTopLevelWindows should fail on non-Windows platform, got status: %v", result.Status)
	}

	if handles != nil {
		t.Error("FindTopLevelWindows should return nil handles on non-Windows platform")
	}
}

func TestBridge_FindWindow(t *testing.T) {
	bridge := NewBridge()
	handle, result := bridge.FindWindow("", "微信")

	if result.Status != adapter.StatusFailed {
		t.Errorf("FindWindow should fail on non-Windows platform, got status: %v", result.Status)
	}

	if handle != 0 {
		t.Error("FindWindow should return 0 handle on non-Windows platform")
	}
}

func TestBridge_GetWindowInfo(t *testing.T) {
	bridge := NewBridge()
	info, result := bridge.GetWindowInfo(12345)

	if result.Status != adapter.StatusFailed {
		t.Errorf("GetWindowInfo should fail on non-Windows platform, got status: %v", result.Status)
	}

	if info.Handle != 0 {
		t.Error("GetWindowInfo should return empty WindowInfo on non-Windows platform")
	}
}

func TestBridge_FocusWindow(t *testing.T) {
	bridge := NewBridge()
	result := bridge.FocusWindow(12345)

	if result.Status != adapter.StatusFailed {
		t.Errorf("FocusWindow should fail on non-Windows platform, got status: %v", result.Status)
	}
}

func TestBridge_EnumerateAccessibleNodes(t *testing.T) {
	bridge := NewBridge()
	nodes, result := bridge.EnumerateAccessibleNodes(12345)

	if result.Status != adapter.StatusFailed {
		t.Errorf("EnumerateAccessibleNodes should fail on non-Windows platform, got status: %v", result.Status)
	}

	if nodes != nil {
		t.Error("EnumerateAccessibleNodes should return nil on non-Windows platform")
	}
}

func TestBridge_CaptureWindow(t *testing.T) {
	bridge := NewBridge()
	data, result := bridge.CaptureWindow(12345)

	if result.Status != adapter.StatusFailed {
		t.Errorf("CaptureWindow should fail on non-Windows platform, got status: %v", result.Status)
	}

	if data != nil {
		t.Error("CaptureWindow should return nil on non-Windows platform")
	}
}

func TestBridge_SendKeys(t *testing.T) {
	bridge := NewBridge()
	result := bridge.SendKeys(12345, "test")

	if result.Status != adapter.StatusFailed {
		t.Errorf("SendKeys should fail on non-Windows platform, got status: %v", result.Status)
	}
}

func TestBridge_Click(t *testing.T) {
	bridge := NewBridge()
	result := bridge.Click(12345, 100, 200)

	if result.Status != adapter.StatusFailed {
		t.Errorf("Click should fail on non-Windows platform, got status: %v", result.Status)
	}
}

func TestBridge_SetClipboardText(t *testing.T) {
	bridge := NewBridge()
	result := bridge.SetClipboardText("test")

	if result.Status != adapter.StatusFailed {
		t.Errorf("SetClipboardText should fail on non-Windows platform, got status: %v", result.Status)
	}
}

func TestBridge_GetClipboardText(t *testing.T) {
	bridge := NewBridge()
	text, result := bridge.GetClipboardText()

	if result.Status != adapter.StatusFailed {
		t.Errorf("GetClipboardText should fail on non-Windows platform, got status: %v", result.Status)
	}

	if text != "" {
		t.Error("GetClipboardText should return empty string on non-Windows platform")
	}
}

func TestBridge_Release(t *testing.T) {
	bridge := NewBridge()
	// Release should not panic on non-Windows platform
	bridge.Release()
}
