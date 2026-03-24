//go:build windows

package windows

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
)

// VisionDebugResult 视觉检测调试结果
type VisionDebugResult struct {
	WindowHandle     uintptr                      `json:"window_handle"`
	WindowWidth      int                          `json:"window_width"`
	WindowHeight     int                          `json:"window_height"`
	ImageSize        int                          `json:"image_size"`
	LeftSidebarRect  [4]int                       `json:"left_sidebar_rect"` // x, y, width, height
	ConversationRects []ConversationRect          `json:"conversation_rects"`
	DetectedFeatures map[string]int               `json:"detected_features"` // 检测到的特征计数
	ProcessingTime   time.Duration                `json:"processing_time"`
	DebugImagePath   string                       `json:"debug_image_path,omitempty"`
	Error            string                       `json:"error,omitempty"`
}

// ConversationRect 会话项矩形和特征
type ConversationRect struct {
	Index         int    `json:"index"`
	X             int    `json:"x"`
	Y             int    `json:"y"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	HasAvatar     bool   `json:"has_avatar"`
	HasText       bool   `json:"has_text"`
	IsSelected    bool   `json:"is_selected"`
	HasUnreadDot  bool   `json:"has_unread_dot"`
	AvatarRect    [4]int `json:"avatar_rect,omitempty"`   // x, y, width, height
	TextRect      [4]int `json:"text_rect,omitempty"`     // x, y, width, height
	UnreadDotRect [4]int `json:"unread_dot_rect,omitempty"` // x, y, width, height
}

// bgrToRGBA 将BGR像素数据转换为RGBA图像
func bgrToRGBA(bgrData []byte, width, height, rowSize int) (*image.RGBA, error) {
	if len(bgrData) < rowSize*height {
		return nil, fmt.Errorf("bgr data too small: expected %d bytes, got %d", rowSize*height, len(bgrData))
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		rowStart := y * rowSize
		for x := 0; x < width; x++ {
			pixelStart := rowStart + x*3
			if pixelStart+2 >= len(bgrData) {
				return nil, fmt.Errorf("pixel data out of bounds at (%d, %d)", x, y)
			}
			// BGR -> RGBA
			b := bgrData[pixelStart]
			g := bgrData[pixelStart+1]
			r := bgrData[pixelStart+2]
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	return img, nil
}

// saveDebugImage 保存调试图像
func saveDebugImage(img *image.RGBA, leftSidebarRect [4]int, convRects []ConversationRect) (string, error) {
	// 创建调试目录
	debugDir := filepath.Join(os.TempDir(), "wechat_vision_debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create debug directory: %v", err)
	}

	// 创建带标注的图像
	annotated := image.NewRGBA(img.Bounds())
	draw.Draw(annotated, img.Bounds(), img, image.Point{}, draw.Src)

	// 绘制左侧会话列表区域（蓝色半透明）
	if leftSidebarRect[2] > 0 && leftSidebarRect[3] > 0 {
		sidebarRect := image.Rect(
			leftSidebarRect[0],
			leftSidebarRect[1],
			leftSidebarRect[0]+leftSidebarRect[2],
			leftSidebarRect[1]+leftSidebarRect[3],
		)
		drawSidebarRect(annotated, sidebarRect)
	}

	// 绘制每个会话项矩形
	for _, rect := range convRects {
		convRect := image.Rect(rect.X, rect.Y, rect.X+rect.Width, rect.Y+rect.Height)
		drawConversationRect(annotated, convRect, rect)

		// 绘制头像区域（绿色）
		if rect.HasAvatar && rect.AvatarRect[2] > 0 && rect.AvatarRect[3] > 0 {
			avatarRect := image.Rect(
				rect.AvatarRect[0],
				rect.AvatarRect[1],
				rect.AvatarRect[0]+rect.AvatarRect[2],
				rect.AvatarRect[1]+rect.AvatarRect[3],
			)
			drawAvatarRect(annotated, avatarRect)
		}

		// 绘制文本区域（黄色）
		if rect.HasText && rect.TextRect[2] > 0 && rect.TextRect[3] > 0 {
			textRect := image.Rect(
				rect.TextRect[0],
				rect.TextRect[1],
				rect.TextRect[0]+rect.TextRect[2],
				rect.TextRect[1]+rect.TextRect[3],
			)
			drawTextRect(annotated, textRect)
		}

		// 绘制未读红点区域（红色）
		if rect.HasUnreadDot && rect.UnreadDotRect[2] > 0 && rect.UnreadDotRect[3] > 0 {
			dotRect := image.Rect(
				rect.UnreadDotRect[0],
				rect.UnreadDotRect[1],
				rect.UnreadDotRect[0]+rect.UnreadDotRect[2],
				rect.UnreadDotRect[1]+rect.UnreadDotRect[3],
			)
			drawUnreadDotRect(annotated, dotRect)
		}
	}

	// 保存PNG文件
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("vision_debug_%d.png", timestamp)
	filepath := filepath.Join(debugDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create debug image file: %v", err)
	}
	defer file.Close()

	if err := png.Encode(file, annotated); err != nil {
		return "", fmt.Errorf("failed to encode PNG: %v", err)
	}

	return filepath, nil
}

// 绘制左侧会话列表区域
func drawSidebarRect(img *image.RGBA, rect image.Rectangle) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			c := img.RGBAAt(x, y)
			// 添加蓝色半透明叠加
			c.B = uint8(min(255, int(c.B)+100))
			c.A = 200
			img.SetRGBA(x, y, c)
		}
	}
}

// 绘制会话项矩形
func drawConversationRect(img *image.RGBA, rect image.Rectangle, convRect ConversationRect) {
	// 根据特征选择颜色
	var rectColor color.RGBA
	if convRect.IsSelected {
		rectColor = color.RGBA{R: 255, G: 255, B: 0, A: 200} // 黄色表示选中
	} else {
		rectColor = color.RGBA{R: 0, G: 255, B: 0, A: 150} // 绿色表示普通
	}

	// 绘制矩形边框
	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Min.Y+2; y++ { // 上边框
			if y < img.Bounds().Max.Y {
				img.SetRGBA(x, y, rectColor)
			}
		}
		for y := rect.Max.Y - 2; y < rect.Max.Y; y++ { // 下边框
			if y < img.Bounds().Max.Y {
				img.SetRGBA(x, y, rectColor)
			}
		}
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Min.X+2; x++ { // 左边框
			if x < img.Bounds().Max.X {
				img.SetRGBA(x, y, rectColor)
			}
		}
		for x := rect.Max.X - 2; x < rect.Max.X; x++ { // 右边框
			if x < img.Bounds().Max.X {
				img.SetRGBA(x, y, rectColor)
			}
		}
	}
}

// 绘制头像区域
func drawAvatarRect(img *image.RGBA, rect image.Rectangle) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			c := img.RGBAAt(x, y)
			// 添加绿色半透明叠加
			c.G = uint8(min(255, int(c.G)+100))
			c.A = 180
			img.SetRGBA(x, y, c)
		}
	}
}

// 绘制文本区域
func drawTextRect(img *image.RGBA, rect image.Rectangle) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			c := img.RGBAAt(x, y)
			// 添加黄色半透明叠加
			c.R = uint8(min(255, int(c.R)+100))
			c.G = uint8(min(255, int(c.G)+100))
			c.A = 180
			img.SetRGBA(x, y, c)
		}
	}
}

// 绘制未读红点区域
func drawUnreadDotRect(img *image.RGBA, rect image.Rectangle) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			c := img.RGBAAt(x, y)
			// 添加红色半透明叠加
			c.R = uint8(min(255, int(c.R)+150))
			c.A = 220
			img.SetRGBA(x, y, c)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// detectHorizontalLines 检测水平线（会话项之间的分隔）
func detectHorizontalLines(img *image.RGBA, rect image.Rectangle, threshold int) []int {
	var lines []int
	prevLineY := -10 // 避免检测到相邻的线

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		// 计算该行的平均亮度变化
		edgeCount := 0
		for x := rect.Min.X; x < rect.Max.X-1; x++ {
			c1 := img.RGBAAt(x, y)
			c2 := img.RGBAAt(x+1, y)
			// 简单的边缘检测：颜色差异
			if colorDiff(c1, c2) > 30 {
				edgeCount++
			}
		}

		// 如果检测到足够的边缘，可能是水平分隔线
		if edgeCount > threshold && y-prevLineY > 5 {
			lines = append(lines, y)
			prevLineY = y
		}
	}

	return lines
}

// colorDiff 计算两个颜色的差异
func colorDiff(c1, c2 color.RGBA) int {
	dr := abs(int(c1.R) - int(c2.R))
	dg := abs(int(c1.G) - int(c2.G))
	db := abs(int(c1.B) - int(c2.B))
	return (dr + dg + db) / 3
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// detectAvatarRegion 检测头像区域
func detectAvatarRegion(img *image.RGBA, convRect image.Rectangle) (bool, [4]int) {
	// 假设头像在会话项左侧，大小约为40x40
	avatarWidth := 40
	avatarHeight := 40
	avatarMargin := 10

	avatarRect := image.Rect(
		convRect.Min.X+avatarMargin,
		convRect.Min.Y+avatarMargin,
		convRect.Min.X+avatarMargin+avatarWidth,
		convRect.Min.Y+avatarMargin+avatarHeight,
	)

	// 检查区域是否在图像范围内
	if avatarRect.Max.X > img.Bounds().Max.X || avatarRect.Max.Y > img.Bounds().Max.Y {
		return false, [4]int{}
	}

	// 简单检查：区域内是否有足够的颜色变化（头像通常有细节）
	colorVariance := computeColorVariance(img, avatarRect)
	if colorVariance > 20 {
		return true, [4]int{avatarRect.Min.X, avatarRect.Min.Y, avatarRect.Dx(), avatarRect.Dy()}
	}

	return false, [4]int{}
}

// computeColorVariance 计算区域内的颜色方差
func computeColorVariance(img *image.RGBA, rect image.Rectangle) int {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return 0
	}

	var sumR, sumG, sumB int
	pixelCount := rect.Dx() * rect.Dy()

	// 计算平均值
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			c := img.RGBAAt(x, y)
			sumR += int(c.R)
			sumG += int(c.G)
			sumB += int(c.B)
		}
	}

	if pixelCount == 0 {
		return 0
	}

	avgR := sumR / pixelCount
	avgG := sumG / pixelCount
	avgB := sumB / pixelCount

	// 计算方差
	var variance int
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			c := img.RGBAAt(x, y)
			variance += abs(int(c.R)-avgR) + abs(int(c.G)-avgG) + abs(int(c.B)-avgB)
		}
	}

	return variance / pixelCount
}

// detectTextRegion 检测文本区域
func detectTextRegion(img *image.RGBA, convRect image.Rectangle, avatarRect [4]int) (bool, [4]int) {
	// 文本区域通常在头像右侧
	textStartX := convRect.Min.X + 60 // 头像宽度 + 边距
	if avatarRect[2] > 0 {
		textStartX = avatarRect[0] + avatarRect[2] + 10
	}

	textRect := image.Rect(
		textStartX,
		convRect.Min.Y+5,
		convRect.Max.X-10,
		convRect.Min.Y+convRect.Dy()-5,
	)

	if textRect.Dx() <= 20 || textRect.Dy() <= 10 {
		return false, [4]int{}
	}

	// 检查文本区域：高边缘密度
	edgeDensity := computeEdgeDensity(img, textRect)
	if edgeDensity > 5 {
		return true, [4]int{textRect.Min.X, textRect.Min.Y, textRect.Dx(), textRect.Dy()}
	}

	return false, [4]int{}
}

// computeEdgeDensity 计算边缘密度
func computeEdgeDensity(img *image.RGBA, rect image.Rectangle) int {
	if rect.Dx() <= 1 || rect.Dy() <= 1 {
		return 0
	}

	edgeCount := 0
	totalPixels := (rect.Dx() - 1) * (rect.Dy() - 1)

	for y := rect.Min.Y; y < rect.Max.Y-1; y++ {
		for x := rect.Min.X; x < rect.Max.X-1; x++ {
			c1 := img.RGBAAt(x, y)
			c2 := img.RGBAAt(x+1, y)
			c3 := img.RGBAAt(x, y+1)

			if colorDiff(c1, c2) > 20 || colorDiff(c1, c3) > 20 {
				edgeCount++
			}
		}
	}

	if totalPixels == 0 {
		return 0
	}

	// 返回每100像素的边缘密度
	return edgeCount * 100 / totalPixels
}

// detectUnreadDot 检测未读红点
func detectUnreadDot(img *image.RGBA, convRect image.Rectangle, avatarRect [4]int) (bool, [4]int) {
	// 未读红点通常在头像右上角
	dotSize := 8
	dotMargin := 3

	var dotRect image.Rectangle
	if avatarRect[2] > 0 {
		// 相对于头像的位置
		dotRect = image.Rect(
			avatarRect[0]+avatarRect[2]-dotSize-dotMargin,
			avatarRect[1]+dotMargin,
			avatarRect[0]+avatarRect[2]-dotMargin,
			avatarRect[1]+dotSize+dotMargin,
		)
	} else {
		// 如果没有检测到头像，假设在会话项左侧
		dotRect = image.Rect(
			convRect.Min.X+5,
			convRect.Min.Y+5,
			convRect.Min.X+5+dotSize,
			convRect.Min.Y+5+dotSize,
		)
	}

	// 检查区域是否在图像范围内
	if dotRect.Max.X > img.Bounds().Max.X || dotRect.Max.Y > img.Bounds().Max.Y {
		return false, [4]int{}
	}

	// 检查红色像素比例
	redPixelCount := 0
	totalPixels := dotRect.Dx() * dotRect.Dy()

	for y := dotRect.Min.Y; y < dotRect.Max.Y; y++ {
		for x := dotRect.Min.X; x < dotRect.Max.X; x++ {
			c := img.RGBAAt(x, y)
			// 检查是否是红色（R值高，G和B值低）
			if int(c.R) > 150 && int(c.G) < 100 && int(c.B) < 100 {
				redPixelCount++
			}
		}
	}

	if totalPixels == 0 {
		return false, [4]int{}
	}

	redRatio := redPixelCount * 100 / totalPixels
	if redRatio > 50 {
		return true, [4]int{dotRect.Min.X, dotRect.Min.Y, dotRect.Dx(), dotRect.Dy()}
	}

	return false, [4]int{}
}

// detectSelectedState 检测选中状态
func detectSelectedState(img *image.RGBA, convRect image.Rectangle) bool {
	// 检查整个会话项的背景色
	backgroundColor := computeAverageColor(img, convRect)

	// 选中项通常有较亮的背景
	brightness := (int(backgroundColor.R) + int(backgroundColor.G) + int(backgroundColor.B)) / 3
	return brightness > 200 // 假设选中项背景较亮
}

// computeAverageColor 计算区域的平均颜色
func computeAverageColor(img *image.RGBA, rect image.Rectangle) color.RGBA {
	var sumR, sumG, sumB int
	pixelCount := rect.Dx() * rect.Dy()

	if pixelCount == 0 {
		return color.RGBA{}
	}

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			c := img.RGBAAt(x, y)
			sumR += int(c.R)
			sumG += int(c.G)
			sumB += int(c.B)
		}
	}

	return color.RGBA{
		R: uint8(sumR / pixelCount),
		G: uint8(sumG / pixelCount),
		B: uint8(sumB / pixelCount),
		A: 255,
	}
}

// DetectConversations 检测会话列表项
func (b *Bridge) DetectConversations(windowHandle uintptr) (VisionDebugResult, adapter.Result) {
	startTime := time.Now()
	result := VisionDebugResult{
		WindowHandle:     windowHandle,
		DetectedFeatures: make(map[string]int),
		ConversationRects: []ConversationRect{},
	}

	// 截图
	pixels, captureResult := b.CaptureWindow(windowHandle)
	if captureResult.Status != adapter.StatusSuccess {
		result.Error = captureResult.Error
		return result, captureResult
	}
	result.ImageSize = len(pixels)

	// 获取窗口尺寸
	rect, rectResult := b.getWindowRectInternal(windowHandle)
	if rectResult.Status != adapter.StatusSuccess {
		result.Error = "Failed to get window dimensions"
		return result, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("WINDOW_DIMENSIONS_FAILED"),
			Error:      result.Error,
		}
	}

	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)
	result.WindowWidth = width
	result.WindowHeight = height

	if width <= 0 || height <= 0 {
		result.Error = fmt.Sprintf("Invalid window dimensions: %dx%d", width, height)
		return result, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("INVALID_WINDOW_DIMENSIONS"),
			Error:      result.Error,
		}
	}

	// 计算行大小
	rowSize := ((width*24 + 31) / 32) * 4

	// 转换为RGBA图像
	img, err := bgrToRGBA(pixels, width, height, rowSize)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to convert BGR to RGBA: %v", err)
		return result, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("IMAGE_CONVERSION_FAILED"),
			Error:      result.Error,
		}
	}

	// 定义左侧会话列表区域（左侧30%）
	leftSidebarWidth := width * 30 / 100
	if leftSidebarWidth < 100 {
		leftSidebarWidth = 100 // 最小宽度
	}

	leftSidebarRect := image.Rect(0, 0, leftSidebarWidth, height)
	result.LeftSidebarRect = [4]int{0, 0, leftSidebarWidth, height}

	// 检测水平线（会话项分隔）
	lines := detectHorizontalLines(img, leftSidebarRect, width/10)
	result.DetectedFeatures["horizontal_lines"] = len(lines)

	// 根据水平线划分会话项
	if len(lines) >= 2 {
		for i := 0; i < len(lines)-1; i++ {
			top := lines[i]
			bottom := lines[i+1]
			itemHeight := bottom - top

			// 过滤掉太小的区域（可能是分隔线本身）
			if itemHeight < 20 || itemHeight > 150 {
				continue
			}

			convRect := image.Rect(
				leftSidebarRect.Min.X,
				top,
				leftSidebarRect.Max.X,
				bottom,
			)

			// 检测特征
			hasAvatar, avatarRect := detectAvatarRegion(img, convRect)
			hasText, textRect := detectTextRegion(img, convRect, avatarRect)
			hasUnreadDot, unreadDotRect := detectUnreadDot(img, convRect, avatarRect)
			isSelected := detectSelectedState(img, convRect)

			// 创建会话项记录
			convItem := ConversationRect{
				Index:        len(result.ConversationRects),
				X:            convRect.Min.X,
				Y:            convRect.Min.Y,
				Width:        convRect.Dx(),
				Height:       convRect.Dy(),
				HasAvatar:    hasAvatar,
				HasText:      hasText,
				HasUnreadDot: hasUnreadDot,
				IsSelected:   isSelected,
			}

			if hasAvatar {
				convItem.AvatarRect = avatarRect
				result.DetectedFeatures["avatars"]++
			}
			if hasText {
				convItem.TextRect = textRect
				result.DetectedFeatures["text_regions"]++
			}
			if hasUnreadDot {
				convItem.UnreadDotRect = unreadDotRect
				result.DetectedFeatures["unread_dots"]++
			}
			if isSelected {
				result.DetectedFeatures["selected_items"]++
			}

			result.ConversationRects = append(result.ConversationRects, convItem)
		}
	}

	// 如果未检测到水平线，尝试基于固定高度划分
	if len(result.ConversationRects) == 0 && height > 0 {
		// 假设每个会话项高度约为60像素
		itemHeight := 60
		itemCount := height / itemHeight

		for i := 0; i < itemCount; i++ {
			top := i * itemHeight
			bottom := top + itemHeight
			if bottom > height {
				bottom = height
			}

			convRect := image.Rect(
				leftSidebarRect.Min.X,
				top,
				leftSidebarRect.Max.X,
				bottom,
			)

			// 检测特征
			hasAvatar, avatarRect := detectAvatarRegion(img, convRect)
			hasText, textRect := detectTextRegion(img, convRect, avatarRect)
			hasUnreadDot, unreadDotRect := detectUnreadDot(img, convRect, avatarRect)
			isSelected := detectSelectedState(img, convRect)

			convItem := ConversationRect{
				Index:        i,
				X:            convRect.Min.X,
				Y:            convRect.Min.Y,
				Width:        convRect.Dx(),
				Height:       convRect.Dy(),
				HasAvatar:    hasAvatar,
				HasText:      hasText,
				HasUnreadDot: hasUnreadDot,
				IsSelected:   isSelected,
			}

			if hasAvatar {
				convItem.AvatarRect = avatarRect
				result.DetectedFeatures["avatars"]++
			}
			if hasText {
				convItem.TextRect = textRect
				result.DetectedFeatures["text_regions"]++
			}
			if hasUnreadDot {
				convItem.UnreadDotRect = unreadDotRect
				result.DetectedFeatures["unread_dots"]++
			}
			if isSelected {
				result.DetectedFeatures["selected_items"]++
			}

			result.ConversationRects = append(result.ConversationRects, convItem)
		}
	}

	// 生成调试图像
	if len(result.ConversationRects) > 0 {
		debugImagePath, err := saveDebugImage(img, result.LeftSidebarRect, result.ConversationRects)
		if err == nil {
			result.DebugImagePath = debugImagePath
			result.DetectedFeatures["debug_image_saved"] = 1
		}
	}

	result.ProcessingTime = time.Since(startTime)

	// 构建诊断结果
	diagnostics := []adapter.Diagnostic{
		{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "Vision detection completed",
			Context: map[string]string{
				"window_handle":        strconv.FormatUint(uint64(windowHandle), 10),
				"window_width":         strconv.Itoa(width),
				"window_height":        strconv.Itoa(height),
				"left_sidebar_width":   strconv.Itoa(leftSidebarWidth),
				"conversations_found":  strconv.Itoa(len(result.ConversationRects)),
				"horizontal_lines":     strconv.Itoa(len(lines)),
				"avatars_detected":     strconv.Itoa(result.DetectedFeatures["avatars"]),
				"text_regions":         strconv.Itoa(result.DetectedFeatures["text_regions"]),
				"unread_dots":          strconv.Itoa(result.DetectedFeatures["unread_dots"]),
				"selected_items":       strconv.Itoa(result.DetectedFeatures["selected_items"]),
				"processing_time":      result.ProcessingTime.String(),
			},
		},
	}

	return result, adapter.Result{
		Status:      adapter.StatusSuccess,
		ReasonCode:  adapter.ReasonOK,
		Diagnostics: diagnostics,
	}
}

// GetConversationClickPoint 获取会话项的点击坐标
// strategy: "avatar_center", "text_center", "rect_center", "left_quarter_center", 或空字符串（使用默认优先级）
func (b *Bridge) GetConversationClickPoint(result VisionDebugResult, index int, strategy string) (x, y int, clickSource string, diag adapter.Diagnostic) {
	if index < 0 || index >= len(result.ConversationRects) {
		return 0, 0, "invalid_index", adapter.Diagnostic{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "Invalid conversation index",
			Context: map[string]string{
				"requested_index": strconv.Itoa(index),
				"total_conversations": strconv.Itoa(len(result.ConversationRects)),
			},
		}
	}

	conv := result.ConversationRects[index]

	// 如果指定了策略，直接使用该策略
	if strategy != "" {
		switch strategy {
		case "avatar_center":
			if conv.HasAvatar && conv.AvatarRect[2] > 0 && conv.AvatarRect[3] > 0 {
				x = conv.AvatarRect[0] + conv.AvatarRect[2]/2
				y = conv.AvatarRect[1] + conv.AvatarRect[3]/2
				return x, y, "avatar_center", adapter.Diagnostic{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   "Click point calculated from avatar center (explicit strategy)",
					Context: map[string]string{
						"index": strconv.Itoa(index),
						"avatar_x": strconv.Itoa(conv.AvatarRect[0]),
						"avatar_y": strconv.Itoa(conv.AvatarRect[1]),
						"avatar_width": strconv.Itoa(conv.AvatarRect[2]),
						"avatar_height": strconv.Itoa(conv.AvatarRect[3]),
						"click_x": strconv.Itoa(x),
						"click_y": strconv.Itoa(y),
						"strategy": strategy,
					},
				}
			} else {
				return 0, 0, "strategy_unavailable", adapter.Diagnostic{
					Timestamp: time.Now(),
					Level:     "warn",
					Message:   "Avatar center strategy requested but avatar not available",
					Context: map[string]string{
						"index": strconv.Itoa(index),
						"has_avatar": strconv.FormatBool(conv.HasAvatar),
						"strategy": strategy,
					},
				}
			}

		case "text_center":
			if conv.HasText && conv.TextRect[2] > 0 && conv.TextRect[3] > 0 {
				x = conv.TextRect[0] + conv.TextRect[2]/2
				y = conv.TextRect[1] + conv.TextRect[3]/2
				return x, y, "text_center", adapter.Diagnostic{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   "Click point calculated from text center (explicit strategy)",
					Context: map[string]string{
						"index": strconv.Itoa(index),
						"text_x": strconv.Itoa(conv.TextRect[0]),
						"text_y": strconv.Itoa(conv.TextRect[1]),
						"text_width": strconv.Itoa(conv.TextRect[2]),
						"text_height": strconv.Itoa(conv.TextRect[3]),
						"click_x": strconv.Itoa(x),
						"click_y": strconv.Itoa(y),
						"strategy": strategy,
					},
				}
			} else {
				return 0, 0, "strategy_unavailable", adapter.Diagnostic{
					Timestamp: time.Now(),
					Level:     "warn",
					Message:   "Text center strategy requested but text not available",
					Context: map[string]string{
						"index": strconv.Itoa(index),
						"has_text": strconv.FormatBool(conv.HasText),
						"strategy": strategy,
					},
				}
			}

		case "rect_center":
			x = conv.X + conv.Width/2
			y = conv.Y + conv.Height/2
			return x, y, "rect_center", adapter.Diagnostic{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Click point calculated from conversation rectangle center (explicit strategy)",
				Context: map[string]string{
					"index": strconv.Itoa(index),
					"rect_x": strconv.Itoa(conv.X),
					"rect_y": strconv.Itoa(conv.Y),
					"rect_width": strconv.Itoa(conv.Width),
					"rect_height": strconv.Itoa(conv.Height),
					"click_x": strconv.Itoa(x),
					"click_y": strconv.Itoa(y),
					"strategy": strategy,
				},
			}

		case "left_quarter_center":
			// 点击矩形左侧四分之一处的中心点
			x = conv.X + conv.Width/4
			y = conv.Y + conv.Height/2
			return x, y, "left_quarter_center", adapter.Diagnostic{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Click point calculated from left quarter center",
				Context: map[string]string{
					"index": strconv.Itoa(index),
					"rect_x": strconv.Itoa(conv.X),
					"rect_y": strconv.Itoa(conv.Y),
					"rect_width": strconv.Itoa(conv.Width),
					"rect_height": strconv.Itoa(conv.Height),
					"click_x": strconv.Itoa(x),
					"click_y": strconv.Itoa(y),
					"strategy": strategy,
				},
			}

		default:
			return 0, 0, "invalid_strategy", adapter.Diagnostic{
				Timestamp: time.Now(),
				Level:     "error",
				Message:   "Invalid click strategy",
				Context: map[string]string{
					"index": strconv.Itoa(index),
					"strategy": strategy,
					"valid_strategies": "avatar_center, text_center, rect_center, left_quarter_center",
				},
			}
		}
	}

	// 未指定策略，使用默认优先级
	// 1. 优先点击头像区域中心
	if conv.HasAvatar && conv.AvatarRect[2] > 0 && conv.AvatarRect[3] > 0 {
		x = conv.AvatarRect[0] + conv.AvatarRect[2]/2
		y = conv.AvatarRect[1] + conv.AvatarRect[3]/2
		return x, y, "avatar_center", adapter.Diagnostic{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "Click point calculated from avatar center (default priority)",
			Context: map[string]string{
				"index": strconv.Itoa(index),
				"avatar_x": strconv.Itoa(conv.AvatarRect[0]),
				"avatar_y": strconv.Itoa(conv.AvatarRect[1]),
				"avatar_width": strconv.Itoa(conv.AvatarRect[2]),
				"avatar_height": strconv.Itoa(conv.AvatarRect[3]),
				"click_x": strconv.Itoa(x),
				"click_y": strconv.Itoa(y),
				"strategy": "default_priority",
			},
		}
	}

	// 2. 其次点击文本区域中心
	if conv.HasText && conv.TextRect[2] > 0 && conv.TextRect[3] > 0 {
		x = conv.TextRect[0] + conv.TextRect[2]/2
		y = conv.TextRect[1] + conv.TextRect[3]/2
		return x, y, "text_center", adapter.Diagnostic{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "Click point calculated from text center (default priority)",
			Context: map[string]string{
				"index": strconv.Itoa(index),
				"text_x": strconv.Itoa(conv.TextRect[0]),
				"text_y": strconv.Itoa(conv.TextRect[1]),
				"text_width": strconv.Itoa(conv.TextRect[2]),
				"text_height": strconv.Itoa(conv.TextRect[3]),
				"click_x": strconv.Itoa(x),
				"click_y": strconv.Itoa(y),
				"strategy": "default_priority",
			},
		}
	}

	// 3. 最后点击会话项矩形中心
	x = conv.X + conv.Width/2
	y = conv.Y + conv.Height/2
	return x, y, "rect_center", adapter.Diagnostic{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "Click point calculated from conversation rectangle center (default priority)",
		Context: map[string]string{
			"index": strconv.Itoa(index),
			"rect_x": strconv.Itoa(conv.X),
			"rect_y": strconv.Itoa(conv.Y),
			"rect_width": strconv.Itoa(conv.Width),
			"rect_height": strconv.Itoa(conv.Height),
			"click_x": strconv.Itoa(x),
			"click_y": strconv.Itoa(y),
			"strategy": "default_priority",
		},
	}
}

// ImageDifferenceResult 图像差异分析结果
type ImageDifferenceResult struct {
	TotalPixels       int     `json:"total_pixels"`
	DifferentPixels   int     `json:"different_pixels"`
	DifferencePercent float64 `json:"difference_percent"`
	DiffBoundingBox   [4]int  `json:"diff_bounding_box"` // x, y, width, height
	DiffCentroidX     int     `json:"diff_centroid_x"`
	DiffCentroidY     int     `json:"diff_centroid_y"`
	LeftSideDiffPixels int    `json:"left_side_diff_pixels"`   // 左侧区域差异像素数
	RightSideDiffPixels int   `json:"right_side_diff_pixels"`  // 右侧区域差异像素数
	LeftSidePercent   float64 `json:"left_side_percent"`       // 左侧差异百分比
	RightSidePercent  float64 `json:"right_side_percent"`      // 右侧差异百分比
	DiffImagePath     string  `json:"diff_image_path,omitempty"`
}

// ComputeImageDifference 计算两幅RGBA图像的差异
func ComputeImageDifference(img1, img2 *image.RGBA, leftSidebarRect [4]int, windowWidth int) (ImageDifferenceResult, error) {
	result := ImageDifferenceResult{}

	if img1 == nil || img2 == nil {
		return result, fmt.Errorf("one or both images are nil")
	}

	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	if bounds1 != bounds2 {
		return result, fmt.Errorf("image dimensions differ: %v vs %v", bounds1, bounds2)
	}

	width := bounds1.Dx()
	height := bounds1.Dy()
	result.TotalPixels = width * height

	// 计算差异
	diffCount := 0
	minX, minY := width, height
	maxX, maxY := 0, 0
	sumX, sumY := 0, 0

	// 定义左侧和右侧区域
	leftSideEnd := leftSidebarRect[0] + leftSidebarRect[2]
	leftDiffCount := 0
	rightDiffCount := 0

	// 创建差异图像（可选）
	diffImg := image.NewRGBA(bounds1)

	for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
		for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
			idx1 := img1.PixOffset(x, y)
			idx2 := img2.PixOffset(x, y)

			// 简单比较RGBA值
			diff := false
			for i := 0; i < 4; i++ {
				if img1.Pix[idx1+i] != img2.Pix[idx2+i] {
					diff = true
					break
				}
			}

			if diff {
				diffCount++
				// 更新边界框
				if x < minX { minX = x }
				if x > maxX { maxX = x }
				if y < minY { minY = y }
				if y > maxY { maxY = y }

				sumX += x
				sumY += y

				// 标记差异图像为红色
				diffImg.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})

				// 统计左右侧差异
				if x < leftSideEnd {
					leftDiffCount++
				} else {
					rightDiffCount++
				}
			} else {
				// 无差异处设置为透明
				diffImg.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}

	result.DifferentPixels = diffCount
	if result.TotalPixels > 0 {
		result.DifferencePercent = float64(diffCount) / float64(result.TotalPixels) * 100.0
	}

	// 计算边界框
	if diffCount > 0 {
		result.DiffBoundingBox = [4]int{minX, minY, maxX - minX + 1, maxY - minY + 1}
		result.DiffCentroidX = sumX / diffCount
		result.DiffCentroidY = sumY / diffCount
	}

	// 左右侧差异统计
	result.LeftSideDiffPixels = leftDiffCount
	result.RightSideDiffPixels = rightDiffCount

	// 计算左右侧差异百分比
	leftSidePixels := leftSideEnd * height
	rightSidePixels := (width - leftSideEnd) * height

	if leftSidePixels > 0 {
		result.LeftSidePercent = float64(leftDiffCount) / float64(leftSidePixels) * 100.0
	}
	if rightSidePixels > 0 {
		result.RightSidePercent = float64(rightDiffCount) / float64(rightSidePixels) * 100.0
	}

	// 保存差异图像（如果差异像素数大于0）
	if diffCount > 0 {
		tempDir := os.TempDir()
		timestamp := time.Now().UnixNano()
		diffPath := filepath.Join(tempDir, fmt.Sprintf("diff_%d.png", timestamp))

		f, err := os.Create(diffPath)
		if err == nil {
			defer f.Close()
			png.Encode(f, diffImg)
			result.DiffImagePath = diffPath
		}
	}

	return result, nil
}

// ComputeRegionDifference 计算特定区域的图像差异
func ComputeRegionDifference(img1, img2 *image.RGBA, regionX, regionY, regionWidth, regionHeight int) (int, float64, error) {
	if img1 == nil || img2 == nil {
		return 0, 0.0, fmt.Errorf("one or both images are nil")
	}

	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	if bounds1 != bounds2 {
		return 0, 0.0, fmt.Errorf("image dimensions differ")
	}

	// 确保区域在图像范围内
	if regionX < bounds1.Min.X { regionX = bounds1.Min.X }
	if regionY < bounds1.Min.Y { regionY = bounds1.Min.Y }
	if regionX+regionWidth > bounds1.Max.X { regionWidth = bounds1.Max.X - regionX }
	if regionY+regionHeight > bounds1.Max.Y { regionHeight = bounds1.Max.Y - regionY }

	if regionWidth <= 0 || regionHeight <= 0 {
		return 0, 0.0, fmt.Errorf("invalid region dimensions")
	}

	totalPixels := regionWidth * regionHeight
	diffCount := 0

	for y := regionY; y < regionY+regionHeight; y++ {
		for x := regionX; x < regionX+regionWidth; x++ {
			idx1 := img1.PixOffset(x, y)
			idx2 := img2.PixOffset(x, y)

			diff := false
			for i := 0; i < 4; i++ {
				if img1.Pix[idx1+i] != img2.Pix[idx2+i] {
					diff = true
					break
				}
			}

			if diff {
				diffCount++
			}
		}
	}

	diffPercent := 0.0
	if totalPixels > 0 {
		diffPercent = float64(diffCount) / float64(totalPixels) * 100.0
	}

	return diffCount, diffPercent, nil
}

// CaptureWindowScreenshot 捕获窗口截图并返回RGBA图像
func (b *Bridge) CaptureWindowScreenshot(windowHandle uintptr) (*image.RGBA, error) {
	// 使用现有的CaptureWindow方法获取像素数据
	pixels, result := b.CaptureWindow(windowHandle)
	if result.Status != adapter.StatusSuccess {
		return nil, fmt.Errorf("failed to capture window: %s", result.Error)
	}

	// 获取窗口尺寸
	rect, rectResult := b.getWindowRectInternal(windowHandle)
	if rectResult.Status != adapter.StatusSuccess {
		return nil, fmt.Errorf("failed to get window rect: %s", rectResult.Error)
	}

	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid window dimensions: %dx%d", width, height)
	}

	// 计算行大小
	rowSize := ((width*24 + 31) / 32) * 4

	// 将BGR转换为RGBA
	return bgrToRGBA(pixels, width, height, rowSize)
}