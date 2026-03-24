//go:build windows

package windows

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
)

// RECT 窗口矩形
type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// OCRDebugResult 用于调试的OCR结果
type OCRDebugResult struct {
	WindowHandle uintptr `json:"window_handle"`
	WindowWidth  int     `json:"window_width"`
	WindowHeight int     `json:"window_height"`
	ImageSize    int     `json:"image_size"`
	Text         string  `json:"text"`
	RegionTexts  map[string]string `json:"region_texts,omitempty"`
	Error        string  `json:"error,omitempty"`
	TesseractPath string `json:"tesseract_path"`
	Language     string  `json:"language"`
	ProcessingTime time.Duration `json:"processing_time"`
}

// bmpHeader BMP文件头
type bmpHeader struct {
	Signature  [2]byte // "BM"
	FileSize   uint32
	Reserved1  uint16
	Reserved2  uint16
	DataOffset uint32
}

// bmpInfoHeader BMP信息头
type bmpInfoHeader struct {
	Size           uint32
	Width          int32
	Height         int32
	Planes         uint16
	BitCount       uint16
	Compression    uint32
	ImageSize      uint32
	XPelsPerMeter  int32
	YPelsPerMeter  int32
	ColorsUsed     uint32
	ColorsImportant uint32
}

// findTesseract 查找tesseract可执行文件路径
func findTesseract() (string, error) {
	// 检查PATH
	if path, err := exec.LookPath("tesseract"); err == nil {
		return path, nil
	}

	// 常见安装位置
	possiblePaths := []string{
		`C:\Program Files\Tesseract-OCR\tesseract.exe`,
		`C:\Program Files (x86)\Tesseract-OCR\tesseract.exe`,
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("tesseract not found in PATH or common locations")
}

// checkTesseractAvailable 检查tesseract是否可用
func checkTesseractAvailable() (string, adapter.Result) {
	path, err := findTesseract()
	if err != nil {
		return "", adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("TESSERACT_NOT_FOUND"),
			Error:      err.Error(),
		}
	}

	// 检查版本
	cmd := exec.Command(path, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("TESSERACT_VERSION_CHECK_FAILED"),
			Error:      fmt.Sprintf("failed to check tesseract version: %v", err),
		}
	}

	return path, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Tesseract OCR engine available",
				Context: map[string]string{
					"tesseract_path": path,
					"version":        strings.TrimSpace(string(output)),
				},
			},
		},
	}
}

// bgrToBMP 将BGR像素数据转换为BMP文件
func bgrToBMP(bgrData []byte, width, height int32, rowSize int32) ([]byte, error) {
	// BMP文件头
	header := bmpHeader{
		Signature:  [2]byte{'B', 'M'},
		FileSize:   0, // 稍后计算
		Reserved1:  0,
		Reserved2:  0,
		DataOffset: 54, // 文件头+信息头大小
	}

	// BMP信息头
	infoHeader := bmpInfoHeader{
		Size:           40,
		Width:          width,
		Height:         height,
		Planes:         1,
		BitCount:       24,
		Compression:    0, // BI_RGB
		ImageSize:      uint32(rowSize * height),
		XPelsPerMeter:  0,
		YPelsPerMeter:  0,
		ColorsUsed:     0,
		ColorsImportant: 0,
	}

	// 计算文件大小
	header.FileSize = uint32(binary.Size(header)) + uint32(binary.Size(infoHeader)) + uint32(infoHeader.ImageSize)

	// 写入缓冲区
	var buf bytes.Buffer
	buf.Grow(int(header.FileSize))

	// 写入文件头
	if err := binary.Write(&buf, binary.LittleEndian, header); err != nil {
		return nil, err
	}

	// 写入信息头
	if err := binary.Write(&buf, binary.LittleEndian, infoHeader); err != nil {
		return nil, err
	}

	// 写入像素数据（BGR格式，行已对齐）
	// 注意：BMP存储顺序是从下到上，而我们的数据是从上到下
	// 需要翻转行顺序
	for y := height - 1; y >= 0; y-- {
		rowStart := int32(y) * rowSize
		rowEnd := rowStart + rowSize
		if int(rowEnd) > len(bgrData) {
			return nil, fmt.Errorf("bgr data too small for row %d", y)
		}
		buf.Write(bgrData[rowStart:rowEnd])
	}

	return buf.Bytes(), nil
}

// extractTextFromBMP 从BMP数据中提取文本
func extractTextFromBMP(bmpData []byte, tesseractPath, lang string) (string, error) {
	// 创建临时文件
	tempDir := os.TempDir()
	inputFile := filepath.Join(tempDir, fmt.Sprintf("ocr_%d.bmp", time.Now().UnixNano()))
	outputBase := filepath.Join(tempDir, fmt.Sprintf("ocr_%d", time.Now().UnixNano()))
	outputFile := outputBase + ".txt"

	// 写入BMP文件
	if err := os.WriteFile(inputFile, bmpData, 0644); err != nil {
		return "", fmt.Errorf("failed to write BMP file: %v", err)
	}
	defer os.Remove(inputFile)
	defer os.Remove(outputFile)

	// 构建tesseract命令
	args := []string{inputFile, outputBase}
	if lang != "" {
		args = append(args, "-l", lang)
	}

	cmd := exec.Command(tesseractPath, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract execution failed: %v", err)
	}

	// 读取输出文件
	textBytes, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read OCR output: %v", err)
	}

	return string(textBytes), nil
}

// ExtractTextFromWindow 从窗口截图提取文本
func (b *Bridge) ExtractTextFromWindow(windowHandle uintptr, lang string) (OCRDebugResult, adapter.Result) {
	startTime := time.Now()
	result := OCRDebugResult{
		WindowHandle: windowHandle,
		Language:     lang,
	}

	// 检查tesseract
	tesseractPath, tessResult := checkTesseractAvailable()
	if tessResult.Status != adapter.StatusSuccess {
		result.Error = tessResult.Error
		return result, tessResult
	}
	result.TesseractPath = tesseractPath

	// 截图
	pixels, captureResult := b.CaptureWindow(windowHandle)
	if captureResult.Status != adapter.StatusSuccess {
		result.Error = captureResult.Error
		return result, captureResult
	}
	result.ImageSize = len(pixels)

	// 获取窗口尺寸用于诊断
	rect, rectResult := b.getWindowRectInternal(windowHandle)
	if rectResult.Status != adapter.StatusSuccess {
		// 非致命错误，继续处理
		result.WindowWidth = 0
		result.WindowHeight = 0
	} else {
		width := rect.Right - rect.Left
		height := rect.Bottom - rect.Top
		result.WindowWidth = int(width)
		result.WindowHeight = int(height)
	}

	// 计算行大小（与CaptureWindow中相同）
	width := result.WindowWidth
	height := result.WindowHeight
	if width <= 0 || height <= 0 {
		// 如果无法获取窗口尺寸，使用估算值
		// 24位BGR，行对齐到4字节
		rowSize := ((width*24 + 31) / 32) * 4
		// 根据像素数据大小估算高度
		if rowSize > 0 && len(pixels) > 0 {
			height = len(pixels) / rowSize
		}
	}

	// 将BGR转换为BMP
	rowSize := ((width*24 + 31) / 32) * 4
	bmpData, err := bgrToBMP(pixels, int32(width), int32(height), int32(rowSize))
	if err != nil {
		result.Error = fmt.Sprintf("BGR to BMP conversion failed: %v", err)
		return result, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("BMP_CONVERSION_FAILED"),
			Error:      result.Error,
		}
	}

	// 调用OCR
	text, err := extractTextFromBMP(bmpData, tesseractPath, lang)
	if err != nil {
		result.Error = fmt.Sprintf("OCR failed: %v", err)
		return result, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("OCR_EXTRACTION_FAILED"),
			Error:      result.Error,
		}
	}

	result.Text = strings.TrimSpace(text)
	result.ProcessingTime = time.Since(startTime)

	return result, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "OCR extraction completed",
				Context: map[string]string{
					"window_handle":   strconv.FormatUint(uint64(windowHandle), 10),
					"window_width":    strconv.Itoa(result.WindowWidth),
					"window_height":   strconv.Itoa(result.WindowHeight),
					"image_size":      strconv.Itoa(result.ImageSize),
					"text_length":     strconv.Itoa(len(result.Text)),
					"tesseract_path":  tesseractPath,
					"language":        lang,
					"processing_time": result.ProcessingTime.String(),
				},
			},
		},
	}
}

// ExtractTextFromWindowRegions 从窗口的不同区域提取文本
func (b *Bridge) ExtractTextFromWindowRegions(windowHandle uintptr, lang string) (OCRDebugResult, adapter.Result) {
	// 暂时直接使用全图OCR，区域分割待实现
	return b.ExtractTextFromWindow(windowHandle, lang)
}

// getWindowRectInternal 获取窗口矩形（内部方法）
func (b *Bridge) getWindowRectInternal(handle uintptr) (RECT, adapter.Result) {
	moduser32 := syscall.NewLazyDLL("user32.dll")
	procGetWindowRect := moduser32.NewProc("GetWindowRect")

	var rect RECT
	ret, _, _ := procGetWindowRect.Call(handle, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return rect, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("GET_WINDOW_RECT_FAILED"),
			Error:      "Failed to get window rectangle",
		}
	}

	return rect, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}