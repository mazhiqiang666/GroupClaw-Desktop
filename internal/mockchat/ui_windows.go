//go:build windows

package mockchat

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows constants
const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_BORDER           = 0x00800000
	WS_VSCROLL          = 0x00200000
	ES_MULTILINE        = 0x0004
	ES_AUTOVSCROLL      = 0x0040
	ES_WANTRETURN       = 0x1000
	BS_PUSHBUTTON       = 0x00000000

	CW_USEDEFAULT = 0x80000000

	WM_CREATE  = 0x0001
	WM_DESTROY = 0x0002
	WM_PAINT   = 0x000F
	WM_CLOSE   = 0x0010
	WM_SIZE    = 0x0005
	WM_COMMAND = 0x0111
	WM_LBUTTONDOWN = 0x0201
	WM_KEYDOWN = 0x0100

	WM_USER = 0x0400

	// Custom messages for UI updates
	WM_UPDATE_CONVERSATIONS = WM_USER + 1
	WM_UPDATE_MESSAGES      = WM_USER + 2

	IDC_ARROW = 32512

	// Edit control ID
	ID_EDIT_INPUT = 1001
	ID_BTN_SEND   = 1002
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
	procFillRect            = user32.NewProc("FillRect")
	procCreateSolidBrush    = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject        = gdi32.NewProc("DeleteObject")
	procGetDC               = user32.NewProc("GetDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procTextOutW            = gdi32.NewProc("TextOutW")
	procSetBkMode           = gdi32.NewProc("SetBkMode")
	procSetTextColor        = gdi32.NewProc("SetTextColor")
	procCreateWindowExWEdit = user32.NewProc("CreateWindowExW") // Edit control
	procCreateWindowExWBtn  = user32.NewProc("CreateWindowExW") // Button control
	procSendMessageW        = user32.NewProc("SendMessageW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
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

	// GUI component handles
	hwndEdit    windows.Handle // Input edit control
	hwndBtn     windows.Handle // Send button

	// Layout dimensions
	convListWidth int
	inputAreaHeight int
}

// NewMockChatUI creates a new UI instance
func NewMockChatUI(app *MockChatApp) *MockChatUI {
	return &MockChatUI{
		app:             app,
		className:       "MockChatWindowClass",
		windowTitle:     "Mock Chat App - 微信桌面版模拟器",
		running:         false,
		convListWidth:   200,
		inputAreaHeight: 100,
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

	// Create input edit control
	editStyle := WS_CHILD | WS_VISIBLE | ES_MULTILINE | ES_AUTOVSCROLL | ES_WANTRETURN | WS_BORDER
	editTitle, _ := windows.UTF16PtrFromString("")
	editClass, _ := windows.UTF16PtrFromString("EDIT")
	hwndEdit, _, err := procCreateWindowExW.Call(
		0,                              // dwExStyle
		uintptr(unsafe.Pointer(editClass)), // lpClassName
		uintptr(unsafe.Pointer(editTitle)), // lpWindowName
		uintptr(editStyle),            // dwStyle
		uintptr(ui.convListWidth),     // x
		uintptr(600-ui.inputAreaHeight), // y
		uintptr(600-ui.convListWidth-100), // nWidth
		uintptr(ui.inputAreaHeight),   // nHeight
		uintptr(ui.hwnd),              // hWndParent
		uintptr(ID_EDIT_INPUT),        // hMenu
		uintptr(ui.hInstance),         // hInstance
		0,                             // lpParam
	)
	if hwndEdit == 0 {
		return fmt.Errorf("failed to create edit control: %v", err)
	}
	ui.hwndEdit = windows.Handle(hwndEdit)

	// Create send button
	btnStyle := WS_CHILD | WS_VISIBLE | BS_PUSHBUTTON
	btnTitle, _ := windows.UTF16PtrFromString("发送")
	btnClass, _ := windows.UTF16PtrFromString("BUTTON")
	hwndBtn, _, err := procCreateWindowExW.Call(
		0,                              // dwExStyle
		uintptr(unsafe.Pointer(btnClass)), // lpClassName
		uintptr(unsafe.Pointer(btnTitle)), // lpWindowName
		uintptr(btnStyle),            // dwStyle
		uintptr(600-100),             // x
		uintptr(600-ui.inputAreaHeight), // y
		uintptr(100),                 // nWidth
		uintptr(ui.inputAreaHeight),  // nHeight
		uintptr(ui.hwnd),             // hWndParent
		uintptr(ID_BTN_SEND),         // hMenu
		uintptr(ui.hInstance),        // hInstance
		0,                            // lpParam
	)
	if hwndBtn == 0 {
		return fmt.Errorf("failed to create button: %v", err)
	}
	ui.hwndBtn = windows.Handle(hwndBtn)

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
		// Handle window resize - trigger repaint
		procInvalidateRect.Call(uintptr(hwnd), 0, 1)
		return 0

	case WM_COMMAND:
		// Handle button clicks
		if wParam == ID_BTN_SEND {
			ui.handleSendButtonClick()
		}
		return 0

	case WM_LBUTTONDOWN:
		// Handle conversation selection
		x := int32(lParam & 0xFFFF)
		y := int32((lParam >> 16) & 0xFFFF)
		ui.handleConversationClick(x, y)
		return 0

	default:
		ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
		return ret
	}
}

// handleSendButtonClick handles the send button click event
func (ui *MockChatUI) handleSendButtonClick() {
	// Get text from edit control
	textLen, _, _ := procGetWindowTextLengthW.Call(uintptr(ui.hwndEdit))
	if textLen > 0 {
		buf := make([]uint16, textLen+1)
		procGetWindowTextW.Call(uintptr(ui.hwndEdit), uintptr(unsafe.Pointer(&buf[0])), textLen+1)
		text := windows.UTF16ToString(buf)

		// Add message to active conversation
		ui.app.mu.Lock()
		if activeConv, exists := ui.app.conversations[ui.app.activeConvID]; exists {
			msg := MockMessage{
				ID:         generateMessageID(),
				ConvID:     ui.app.activeConvID,
				SenderSide: "agent",
				Content:    text,
				Timestamp:  time.Now(),
			}
			activeConv.Messages = append(activeConv.Messages, msg)
		}
		ui.app.mu.Unlock()

		// Clear edit control
		procSendMessageW.Call(uintptr(ui.hwndEdit), 0x000C, 0, 0) // WM_SETTEXT

		// Trigger repaint
		procInvalidateRect.Call(uintptr(ui.hwnd), 0, 1)
	}
}

// handleConversationClick handles clicking on a conversation in the list
func (ui *MockChatUI) handleConversationClick(x int32, y int32) {
	// Check if click is in conversation list area
	if x < int32(ui.convListWidth) {
		// Calculate which conversation was clicked
		itemHeight := int32(40)
		startY := int32(50) // Title + padding

		ui.app.mu.Lock()
		defer ui.app.mu.Unlock()

		for _, conv := range ui.app.conversations {
			if y >= startY && y < startY+itemHeight {
				// Set this conversation as active
				ui.app.activeConvID = conv.ID
				for _, c := range ui.app.conversations {
					c.IsActive = (c.ID == conv.ID)
				}
				// Trigger repaint
				procInvalidateRect.Call(uintptr(ui.hwnd), 0, 1)
				return
			}
			startY += itemHeight
		}
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

	// Draw conversation list background (left panel)
	convListRect := struct {
		Left   int32
		Top    int32
		Right  int32
		Bottom int32
	}{
		Left:   0,
		Top:    0,
		Right:  int32(ui.convListWidth),
		Bottom: rect.Bottom,
	}
	brush, _, _ := procCreateSolidBrush.Call(0xE8E8E8) // Light gray for list
	procFillRect.Call(uintptr(ps.Hdc), uintptr(unsafe.Pointer(&convListRect)), brush)
	procDeleteObject.Call(brush)

	// Draw message area background (right panel)
	msgAreaRect := struct {
		Left   int32
		Top    int32
		Right  int32
		Bottom int32
	}{
		Left:   int32(ui.convListWidth),
		Top:    0,
		Right:  rect.Right,
		Bottom: rect.Bottom - int32(ui.inputAreaHeight),
	}
	brush2, _, _ := procCreateSolidBrush.Call(0xFFFFFF) // White for message area
	procFillRect.Call(uintptr(ps.Hdc), uintptr(unsafe.Pointer(&msgAreaRect)), brush2)
	procDeleteObject.Call(brush2)

	// Draw input area background
	inputAreaRect := struct {
		Left   int32
		Top    int32
		Right  int32
		Bottom int32
	}{
		Left:   int32(ui.convListWidth),
		Top:    rect.Bottom - int32(ui.inputAreaHeight),
		Right:  rect.Right - 100,
		Bottom: rect.Bottom,
	}
	brush3, _, _ := procCreateSolidBrush.Call(0xF5F5F5) // Light gray for input
	procFillRect.Call(uintptr(ps.Hdc), uintptr(unsafe.Pointer(&inputAreaRect)), brush3)
	procDeleteObject.Call(brush3)

	// Draw conversation list items
	ui.drawConversationList(ps.Hdc)

	// Draw messages for active conversation
	ui.drawMessages(ps.Hdc)

	// Draw separator line
	procSetTextColor.Call(uintptr(ps.Hdc), 0xCCCCCC)
	procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps))) // Already in paint

	procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
}

// drawConversationList draws the conversation list on the left panel
func (ui *MockChatUI) drawConversationList(hdc windows.Handle) {
	ui.app.mu.RLock()
	defer ui.app.mu.RUnlock()

	y := int32(10)
	itemHeight := int32(40)

	// Draw title
	titleColor := 0x000000
	procSetTextColor.Call(uintptr(hdc), uintptr(titleColor))
	procSetBkMode.Call(uintptr(hdc), 2) // TRANSPARENT

	titlePtr, _ := windows.UTF16PtrFromString("会话列表")
	procTextOutW.Call(uintptr(hdc), 10, uintptr(y), uintptr(unsafe.Pointer(titlePtr)), uintptr(len("会话列表")))
	y += itemHeight + 5

	// Draw each conversation
	for _, conv := range ui.app.conversations {
		// Highlight active conversation
		if conv.IsActive {
			brush, _, _ := procCreateSolidBrush.Call(0x0078D7) // Blue highlight
			itemRect := struct {
				Left   int32
				Top    int32
				Right  int32
				Bottom int32
			}{
				Left:   0,
				Top:    y - 5,
				Right:  int32(ui.convListWidth),
				Bottom: y + itemHeight,
			}
			procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&itemRect)), brush)
			procDeleteObject.Call(brush)
			procSetTextColor.Call(uintptr(hdc), 0xFFFFFF) // White text
		} else {
			procSetTextColor.Call(uintptr(hdc), 0x000000) // Black text
		}

		// Draw conversation name
		name := conv.DisplayName
		if conv.UnreadCount > 0 {
			name = fmt.Sprintf("%s (%d)", conv.DisplayName, conv.UnreadCount)
		}
		namePtr, _ := windows.UTF16PtrFromString(name)
		procTextOutW.Call(uintptr(hdc), 10, uintptr(y), uintptr(unsafe.Pointer(namePtr)), uintptr(len(name)))

		y += itemHeight
	}
}

// drawMessages draws messages for the active conversation
func (ui *MockChatUI) drawMessages(hdc windows.Handle) {
	ui.app.mu.RLock()
	defer ui.app.mu.RUnlock()

	activeConv, exists := ui.app.conversations[ui.app.activeConvID]
	if !exists {
		return
	}

	procSetTextColor.Call(uintptr(hdc), 0x000000)
	procSetBkMode.Call(uintptr(hdc), 2) // TRANSPARENT

	y := int32(10)
	itemHeight := int32(30)

	// Draw active conversation name at top
	titlePtr, _ := windows.UTF16PtrFromString(fmt.Sprintf("当前会话: %s", activeConv.DisplayName))
	procTextOutW.Call(uintptr(hdc), uintptr(ui.convListWidth+10), uintptr(y), uintptr(unsafe.Pointer(titlePtr)), uintptr(len(fmt.Sprintf("当前会话: %s", activeConv.DisplayName))))
	y += itemHeight + 10

	// Draw messages
	for _, msg := range activeConv.Messages {
		prefix := "客户: "
		if msg.SenderSide == "agent" {
			prefix = "客服: "
		}
		text := prefix + msg.Content
		if len(text) > 50 {
			text = text[:47] + "..."
		}

		textPtr, _ := windows.UTF16PtrFromString(text)
		procTextOutW.Call(uintptr(hdc), uintptr(ui.convListWidth+10), uintptr(y), uintptr(unsafe.Pointer(textPtr)), uintptr(len(text)))

		y += itemHeight
		if y > 450 {
			break // Don't draw beyond message area
		}
	}

	// Draw placeholder if no messages
	if len(activeConv.Messages) == 0 {
		placeholderPtr, _ := windows.UTF16PtrFromString("暂无消息")
		procTextOutW.Call(uintptr(hdc), uintptr(ui.convListWidth+10), uintptr(y), uintptr(unsafe.Pointer(placeholderPtr)), uintptr(len("暂无消息")))
	}
}

// Run starts the GUI event loop
func (ui *MockChatUI) Run() {
	ui.running = true

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
