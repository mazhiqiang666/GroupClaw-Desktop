//go:build windows

package mockchat

import (
	"fmt"
	"log"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows constants
const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	CW_USEDEFAULT       = 0x80000000

	WM_CREATE  = 0x0001
	WM_DESTROY = 0x0002
	WM_PAINT   = 0x000F
	WM_CLOSE   = 0x0010
	WM_SIZE    = 0x0005

	WM_USER = 0x0400

	// Custom messages for UI updates
	WM_UPDATE_CONVERSATIONS = WM_USER + 1
	WM_UPDATE_MESSAGES      = WM_USER + 2

	IDC_ARROW = 32512
)

var (
	user32                  = windows.NewLazyDLL("user32.dll")
	gdi32                   = windows.NewLazyDLL("gdi32.dll")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procUpdateWindow        = user32.NewProc("UpdateWindow")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procLoadCursorW         = user32.NewProc("LoadCursorW")
	procGetClientRect       = user32.NewProc("GetClientRect")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procSetWindowTextW      = user32.NewProc("SetWindowTextW")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procBringWindowToTop    = user32.NewProc("BringWindowToTop")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procInvalidateRect      = user32.NewProc("InvalidateRect")
	procBeginPaint          = user32.NewProc("BeginPaint")
	procEndPaint            = user32.NewProc("EndPaint")
	procCreateSolidBrush    = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject        = gdi32.NewProc("DeleteObject")
	procGetDC               = user32.NewProc("GetDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procTextOutW            = gdi32.NewProc("TextOutW")
	procSetBkMode           = gdi32.NewProc("SetBkMode")
	procSetTextColor        = gdi32.NewProc("SetTextColor")
)

// WNDCLASSEXW structure
type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

// MSG structure
type MSG struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct {
		X int32
		Y int32
	}
}

// MockChatUI manages the GUI for the mock chat application
type MockChatUI struct {
	app         *MockChatApp
	hwnd        windows.Handle
	hInstance   windows.Handle
	className   string
	windowTitle string
	mu          sync.RWMutex
	running     bool
}

// NewMockChatUI creates a new UI instance
func NewMockChatUI(app *MockChatApp) *MockChatUI {
	return &MockChatUI{
		app:         app,
		className:   "MockChatWindowClass",
		windowTitle: "Mock Chat App - 微信桌面版模拟器",
		running:     false,
	}
}

// Initialize initializes the Windows window
func (ui *MockChatUI) Initialize() error {
	// Get module handle
	kernel32 := windows.NewLazyDLL("kernel32.dll")
	getModuleHandle := kernel32.NewProc("GetModuleHandleW")
	hInstance, _, _ := getModuleHandle.Call(0)
	ui.hInstance = windows.Handle(hInstance)

	// Register window class
	classNamePtr, _ := windows.UTF16PtrFromString(ui.className)
	menuNamePtr := (*uint16)(nil)

	wndClass := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         0,
		LpfnWndProc:   syscall.NewCallback(ui.wndProc),
		CbClsExtra:    0,
		CbWndExtra:    0,
		HInstance:     windows.Handle(hInstance),
		HIcon:         0,
		HCursor:       0,
		HbrBackground: windows.Handle(5), // COLOR_WINDOW+1
		LpszMenuName:  menuNamePtr,
		LpszClassName: classNamePtr,
		HIconSm:       0,
	}

	ret, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))
	if ret == 0 {
		return fmt.Errorf("failed to register window class: %v", err)
	}

	// Create window
	titlePtr, _ := windows.UTF16PtrFromString(ui.windowTitle)
	hwnd, _, err := procCreateWindowExW.Call(
		0,                              // dwExStyle
		uintptr(unsafe.Pointer(classNamePtr)), // lpClassName
		uintptr(unsafe.Pointer(titlePtr)),     // lpWindowName
		WS_OVERLAPPEDWINDOW|WS_VISIBLE, // dwStyle
		CW_USEDEFAULT,                  // x
		CW_USEDEFAULT,                  // y
		800,                            // nWidth
		600,                            // nHeight
		0,                              // hWndParent
		0,                              // hMenu
		uintptr(hInstance),             // hInstance
		0,                              // lpParam
	)

	if hwnd == 0 {
		return fmt.Errorf("failed to create window: %v", err)
	}

	ui.hwnd = windows.Handle(hwnd)
	ui.app.windowHandle = uintptr(hwnd)

	// Show window
	procShowWindow.Call(uintptr(ui.hwnd), 1) // SW_SHOW
	procUpdateWindow.Call(uintptr(ui.hwnd))

	return nil
}

// wndProc is the window procedure
func (ui *MockChatUI) wndProc(hwnd windows.Handle, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		return 0

	case WM_PAINT:
		ui.paintWindow(hwnd)
		return 0

	case WM_CLOSE:
		procPostQuitMessage.Call(0)
		return 0

	case WM_DESTROY:
		ui.running = false
		return 0

	case WM_SIZE:
		// Handle window resize
		return 0

	default:
		ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
		return ret
	}
}

// paintWindow handles window painting
func (ui *MockChatUI) paintWindow(hwnd windows.Handle) {
	var rect struct {
		Left   int32
		Top    int32
		Right  int32
		Bottom int32
	}
	procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))

	var ps struct {
		Hdc         windows.Handle
		RcPaint     struct{ Left, Top, Right, Bottom int32 }
		FErase      int32
		FIncUpdate  uint32
		RgbReserved [32]byte
	}
	procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

	// Draw background
	brush, _, _ := procCreateSolidBrush.Call(0xF0F0F0) // Light gray
	procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
	procDeleteObject.Call(brush)
}

// Run starts the GUI event loop
func (ui *MockChatUI) Run() {
	ui.running = true

	// Start HTTP server in background
	go func() {
		if err := ui.app.StartHTTPServer(":8081"); err != nil {
			log.Printf("HTTP server failed: %v", err)
		}
	}()

	// Message loop
	var msg MSG
	for ui.running {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break // WM_QUIT
		}

		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// GetWindowHandle returns the window handle
func (ui *MockChatUI) GetWindowHandle() uintptr {
	ui.mu.RLock()
	defer ui.mu.RUnlock()
	return uintptr(ui.hwnd)
}

// GetUIAMode returns the current UIA mode
func (ui *MockChatUI) GetUIAMode() UIAMode {
	ui.app.mu.RLock()
	defer ui.app.mu.RUnlock()
	return ui.app.uiaMode
}

// Close closes the window
func (ui *MockChatUI) Close() {
	ui.running = false
	if ui.hwnd != 0 {
		procPostQuitMessage.Call(0)
	}
}
