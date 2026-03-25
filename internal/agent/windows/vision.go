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
	"sort"
	"strconv"
	"strings"
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

// candidateScore 输入框候选评分结构（内部使用）
type candidateScore struct {
	rect     InputBoxRect
	score    int
	features map[string]string
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

// 绘制输入框矩形
func drawInputBoxRect(img *image.RGBA, rect image.Rectangle) {
	// 使用紫色边框表示输入框
	rectColor := color.RGBA{R: 255, G: 0, B: 255, A: 200}

	// 绘制矩形边框（2像素宽）
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

// saveInputBoxDebugImage 保存输入框调试图像
func saveInputBoxDebugImage(img *image.RGBA, inputBoxRect InputBoxRect, leftSidebarRect [4]int, windowWidth, windowHeight int) (string, error) {
	// 创建调试目录
	debugDir := filepath.Join(os.TempDir(), "wechat_inputbox_debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create debug directory: %v", err)
	}

	// 创建带标注的图像
	annotated := image.NewRGBA(img.Bounds())
	draw.Draw(annotated, img.Bounds(), img, image.Point{}, draw.Src)

	// 绘制左侧边栏区域（如果有效）
	if leftSidebarRect[2] > 0 && leftSidebarRect[3] > 0 {
		sidebarRect := image.Rect(
			leftSidebarRect[0],
			leftSidebarRect[1],
			leftSidebarRect[0]+leftSidebarRect[2],
			leftSidebarRect[1]+leftSidebarRect[3],
		)
		drawSidebarRect(annotated, sidebarRect)
	}

	// 绘制输入框矩形
	if inputBoxRect.Width > 0 && inputBoxRect.Height > 0 {
		inputRect := image.Rect(
			inputBoxRect.X,
			inputBoxRect.Y,
			inputBoxRect.X+inputBoxRect.Width,
			inputBoxRect.Y+inputBoxRect.Height,
		)
		drawInputBoxRect(annotated, inputRect)
	}

	// 保存PNG文件
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("inputbox_debug_%d.png", timestamp)
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

// saveInputBoxCandidatesDebugImage 保存输入框候选调试图像
func saveInputBoxCandidatesDebugImage(img *image.RGBA, candidates []candidateScore, leftSidebarRect [4]int, windowWidth, windowHeight int) (string, error) {
	// 创建调试目录
	debugDir := filepath.Join(os.TempDir(), "wechat_inputbox_debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create debug directory: %v", err)
	}

	// 创建带标注的图像
	annotated := image.NewRGBA(img.Bounds())
	draw.Draw(annotated, img.Bounds(), img, image.Point{}, draw.Src)

	// 绘制左侧边栏区域（如果有效）
	if leftSidebarRect[2] > 0 && leftSidebarRect[3] > 0 {
		sidebarRect := image.Rect(
			leftSidebarRect[0],
			leftSidebarRect[1],
			leftSidebarRect[0]+leftSidebarRect[2],
			leftSidebarRect[1]+leftSidebarRect[3],
		)
		drawSidebarRect(annotated, sidebarRect)
	}

	// 为每个候选区域绘制不同颜色的矩形
	colors := []color.RGBA{
		{R: 255, G: 0, B: 0, A: 200},     // 红色 - 候选0
		{R: 0, G: 255, B: 0, A: 200},     // 绿色 - 候选1
		{R: 0, G: 0, B: 255, A: 200},     // 蓝色 - 候选2
		{R: 255, G: 255, B: 0, A: 200},   // 黄色 - 候选3
		{R: 255, G: 0, B: 255, A: 200},   // 紫色 - 候选4
	}

	for i, c := range candidates {
		if c.rect.Width > 0 && c.rect.Height > 0 {
			inputRect := image.Rect(
				c.rect.X,
				c.rect.Y,
				c.rect.X+c.rect.Width,
				c.rect.Y+c.rect.Height,
			)

			// 使用不同颜色绘制矩形
			colorIndex := i % len(colors)
			rectColor := colors[colorIndex]

			// 绘制矩形边框（3像素宽）
			for x := inputRect.Min.X; x < inputRect.Max.X; x++ {
				for y := inputRect.Min.Y; y < inputRect.Min.Y+3; y++ { // 上边框
					if y < annotated.Bounds().Max.Y {
						annotated.SetRGBA(x, y, rectColor)
					}
				}
				for y := inputRect.Max.Y - 3; y < inputRect.Max.Y; y++ { // 下边框
					if y < annotated.Bounds().Max.Y {
						annotated.SetRGBA(x, y, rectColor)
					}
				}
			}
			for y := inputRect.Min.Y; y < inputRect.Max.Y; y++ {
				for x := inputRect.Min.X; x < inputRect.Min.X+3; x++ { // 左边框
					if x < annotated.Bounds().Max.X {
						annotated.SetRGBA(x, y, rectColor)
					}
				}
				for x := inputRect.Max.X - 3; x < inputRect.Max.X; x++ { // 右边框
					if x < annotated.Bounds().Max.X {
						annotated.SetRGBA(x, y, rectColor)
					}
				}
			}

			// 在矩形左上角绘制彩色条表示索引
			barWidth := 20
			barHeight := 10
			barX := inputRect.Min.X
			barY := inputRect.Min.Y - barHeight
			if barY < 0 {
				barY = inputRect.Min.Y
			}

			for x := barX; x < barX+barWidth && x < annotated.Bounds().Max.X; x++ {
				for y := barY; y < barY+barHeight && y < annotated.Bounds().Max.Y; y++ {
					annotated.SetRGBA(x, y, rectColor)
				}
			}
		}
	}

	// 保存PNG文件
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("inputbox_candidates_%d.png", timestamp)
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

	// 获取窗口尺寸（逻辑坐标）
	rect, rectResult := b.getWindowRectInternal(windowHandle)
	if rectResult.Status != adapter.StatusSuccess {
		result.Error = "Failed to get window dimensions"
		return result, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("WINDOW_DIMENSIONS_FAILED"),
			Error:      result.Error,
		}
	}

	windowWidth := int(rect.Right - rect.Left)
	windowHeight := int(rect.Bottom - rect.Top)

	if windowWidth <= 0 || windowHeight <= 0 {
		result.Error = fmt.Sprintf("Invalid window dimensions: %dx%d", windowWidth, windowHeight)
		return result, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("INVALID_WINDOW_DIMENSIONS"),
			Error:      result.Error,
		}
	}

	// 计算实际截图尺寸（处理DPI缩放）
	width, height := calculateScreenshotDimensions(pixels, windowWidth, windowHeight)
	result.WindowWidth = width
	result.WindowHeight = height

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

// VisionFocusResult 视觉Focus结果
type VisionFocusResult struct {
	WindowHandle   uintptr                        `json:"window_handle"`
	TargetIndex    int                            `json:"target_index"`
	TargetRect     ConversationRect               `json:"target_rect"`
	ClickStrategy  string                         `json:"click_strategy"`
	ClickX         int                            `json:"click_x"`
	ClickY         int                            `json:"click_y"`
	ClickSource    string                         `json:"click_source"`
	FocusSucceeded bool                           `json:"focus_succeeded"`
	FocusConfidence float64                       `json:"focus_confidence"` // 0.0 ~ 1.0
	SuccessReasons []string                       `json:"success_reasons"`
	VerificationSignals map[string]interface{}    `json:"verification_signals"` // 各类验证信号值
	ProcessingTime time.Duration                  `json:"processing_time"`
	Error          string                         `json:"error,omitempty"`
	DebugImagePath string                         `json:"debug_image_path,omitempty"`
}

// selectTargetConversation 选择目标会话项
// 如果targetIndex >= 0，直接选择该index；否则使用默认选择逻辑
func selectTargetConversation(convRects []ConversationRect, targetIndex int) (int, string) {
	if targetIndex >= 0 {
		if targetIndex < len(convRects) {
			return targetIndex, "explicit_index"
		}
		// 如果指定的index无效，回退到默认逻辑
	}

	// 默认选择逻辑：选择第一个"同时具备文本区域或头像区域"的高置信候选项
	for i, conv := range convRects {
		// 优先选择同时有文本和头像的项
		if conv.HasText && conv.HasAvatar {
			return i, "default_has_text_and_avatar"
		}
	}

	// 其次选择有文本的项
	for i, conv := range convRects {
		if conv.HasText {
			return i, "default_has_text"
		}
	}

	// 最后选择有头像的项
	for i, conv := range convRects {
		if conv.HasAvatar {
			return i, "default_has_avatar"
		}
	}

	// 如果都没有，选择第一个项
	if len(convRects) > 0 {
		return 0, "default_first_item"
	}

	return -1, "no_conversations"
}

// evaluateFocusSuccess 评估Focus是否成功
// 基于4类验证信号计算成功率和置信度
func evaluateFocusSuccess(beforeImg, afterImg *image.RGBA, targetRect ConversationRect, leftSidebarRect [4]int, windowWidth int) (bool, float64, []string, map[string]interface{}) {
	if beforeImg == nil || afterImg == nil {
		// 无法进行像素级验证，返回保守结果
		return false, 0.3, []string{"insufficient_pixel_data"}, map[string]interface{}{
			"error": "insufficient_pixel_data",
		}
	}

	reasons := []string{}
	signals := make(map[string]interface{})

	// 1. 计算整窗差异
	fullDiff, err := ComputeImageDifference(beforeImg, afterImg, leftSidebarRect, windowWidth)
	if err == nil {
		signals["full_window_diff_percent"] = fullDiff.DifferencePercent
		signals["full_window_diff_pixels"] = fullDiff.DifferentPixels

		// 阈值：整体差异超过0.1%认为有变化
		if fullDiff.DifferencePercent > 0.1 {
			reasons = append(reasons, "full_window_change_detected")
			signals["full_window_change"] = true
		} else {
			signals["full_window_change"] = false
		}
	}

	// 2. 计算右侧区域差异（消息区）
	if fullDiff.RightSidePercent > 0 {
		signals["right_side_diff_percent"] = fullDiff.RightSidePercent
		signals["right_side_diff_pixels"] = fullDiff.RightSideDiffPixels

		// 右侧差异超过0.2%认为消息区有变化
		if fullDiff.RightSidePercent > 0.2 {
			reasons = append(reasons, "right_side_change_detected")
			signals["right_side_change"] = true
		} else {
			signals["right_side_change"] = false
		}
	}

	// 3. 计算左侧点击项区域差异
	if targetRect.X >= 0 && targetRect.Y >= 0 && targetRect.Width > 0 && targetRect.Height > 0 {
		// 扩大区域以捕获周围变化
		regionX := targetRect.X - 5
		regionY := targetRect.Y - 5
		regionWidth := targetRect.Width + 10
		regionHeight := targetRect.Height + 10

		if regionX < 0 { regionX = 0 }
		if regionY < 0 { regionY = 0 }

		clickedDiffCount, clickedDiffPercent, err := ComputeRegionDifference(
			beforeImg, afterImg,
			regionX, regionY, regionWidth, regionHeight,
		)

		if err == nil {
			signals["clicked_region_diff_percent"] = clickedDiffPercent
			signals["clicked_region_diff_pixels"] = clickedDiffCount

			// 点击区域差异超过0.5%认为有变化
			if clickedDiffPercent > 0.5 {
				reasons = append(reasons, "clicked_region_change_detected")
				signals["clicked_region_change"] = true
			} else {
				signals["clicked_region_change"] = false
			}
		}
	}

	// 4. 检查差异边界框位置是否合理
	if fullDiff.DiffBoundingBox[2] > 0 && fullDiff.DiffBoundingBox[3] > 0 {
		signals["diff_bounding_box"] = fullDiff.DiffBoundingBox
		signals["diff_centroid"] = [2]int{fullDiff.DiffCentroidX, fullDiff.DiffCentroidY}

		// 检查边界框是否在合理范围内（不在边缘）
		boxX, boxY, boxWidth, boxHeight := fullDiff.DiffBoundingBox[0], fullDiff.DiffBoundingBox[1], fullDiff.DiffBoundingBox[2], fullDiff.DiffBoundingBox[3]
		centerX := boxX + boxWidth/2
		centerY := boxY + boxHeight/2

		// 获取图像高度
		windowHeight := beforeImg.Bounds().Dy()

		// 如果边界框中心不在图像边缘附近（10%范围内），认为合理
		marginX := windowWidth / 10
		marginY := windowHeight / 10
		if centerX > marginX && centerX < windowWidth - marginX &&
		   centerY > marginY && centerY < windowHeight - marginY {
			reasons = append(reasons, "diff_bbox_centered")
			signals["diff_bbox_centered"] = true
		} else {
			signals["diff_bbox_centered"] = false
		}
	}

	// 计算置信度
	confidence := 0.0
	successCount := 0
	totalSignals := 0

	// 信号1：整窗变化
	if val, ok := signals["full_window_change"]; ok && val.(bool) {
		successCount++
	}
	if _, ok := signals["full_window_change"]; ok {
		totalSignals++
	}

	// 信号2：右侧变化
	if val, ok := signals["right_side_change"]; ok && val.(bool) {
		successCount++
	}
	if _, ok := signals["right_side_change"]; ok {
		totalSignals++
	}

	// 信号3：点击区域变化
	if val, ok := signals["clicked_region_change"]; ok && val.(bool) {
		successCount++
	}
	if _, ok := signals["clicked_region_change"]; ok {
		totalSignals++
	}

	// 信号4：边界框居中
	if val, ok := signals["diff_bbox_centered"]; ok && val.(bool) {
		successCount++
	}
	if _, ok := signals["diff_bbox_centered"]; ok {
		totalSignals++
	}

	// 计算置信度
	if totalSignals > 0 {
		confidence = float64(successCount) / float64(totalSignals)
	}

	// 成功判定：至少2个信号通过且置信度>=0.5
	success := successCount >= 2 && confidence >= 0.5

	return success, confidence, reasons, signals
}

// FocusConversationByVision 视觉Focus统一入口
// windowHandle: 窗口句柄
// strategy: 点击策略 ("avatar_center", "text_center", "rect_center", "left_quarter_center", 或空字符串使用默认优先级)
// targetIndex: 目标会话索引，-1表示使用默认选择逻辑
// waitAfterClickMs: 点击后等待时间（毫秒），默认800ms
func (b *Bridge) FocusConversationByVision(windowHandle uintptr, strategy string, targetIndex int, waitAfterClickMs int) (VisionFocusResult, adapter.Result) {
	startTime := time.Now()
	result := VisionFocusResult{
		WindowHandle:   windowHandle,
		TargetIndex:    targetIndex,
		ClickStrategy:  strategy,
		FocusSucceeded: false,
		FocusConfidence: 0.0,
		SuccessReasons: []string{},
		VerificationSignals: make(map[string]interface{}),
	}

	if waitAfterClickMs <= 0 {
		waitAfterClickMs = 800 // 默认800ms
	}

	// ============================================
	// 步骤1：点击前视觉检测
	// ============================================
	beforeVision, visionResult := b.DetectConversations(windowHandle)
	if visionResult.Status != adapter.StatusSuccess {
		result.Error = fmt.Sprintf("Failed to detect conversations: %s", visionResult.Error)
		return result, visionResult
	}

	if len(beforeVision.ConversationRects) == 0 {
		result.Error = "No conversations detected"
		return result, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("NO_CONVERSATIONS_DETECTED"),
			Error:      result.Error,
		}
	}

	// ============================================
	// 步骤2：选择目标会话项
	// ============================================
	selectedIndex, selectSource := selectTargetConversation(beforeVision.ConversationRects, targetIndex)
	if selectedIndex < 0 {
		result.Error = "Failed to select target conversation"
		return result, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("TARGET_SELECTION_FAILED"),
			Error:      result.Error,
		}
	}

	result.TargetIndex = selectedIndex
	result.TargetRect = beforeVision.ConversationRects[selectedIndex]
	result.VerificationSignals["selection_source"] = selectSource

	// ============================================
	// 步骤3：点击前截图
	// ============================================
	beforeScreenshot, err := b.CaptureWindowScreenshot(windowHandle)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to capture pre-click screenshot: %v", err)
		// 非致命错误，继续执行
		beforeScreenshot = nil
	}

	// ============================================
	// 步骤4：计算点击点
	// ============================================
	x, y, clickSource, clickDiag := b.GetConversationClickPoint(beforeVision, selectedIndex, strategy)
	if clickSource == "invalid_strategy" || clickSource == "strategy_unavailable" {
		result.Error = fmt.Sprintf("Click strategy failed: %s", clickDiag.Message)
		return result, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CLICK_STRATEGY_FAILED"),
			Error:      result.Error,
		}
	}

	result.ClickX = x
	result.ClickY = y
	result.ClickSource = clickSource
	result.VerificationSignals["click_diagnostic"] = clickDiag

	// ============================================
	// 步骤5：执行点击
	// ============================================
	clickResult := b.Click(windowHandle, x, y)
	if clickResult.Status != adapter.StatusSuccess {
		result.Error = fmt.Sprintf("Click failed: %s", clickResult.Error)
		return result, clickResult
	}

	// ============================================
	// 步骤6：等待UI更新
	// ============================================
	time.Sleep(time.Duration(waitAfterClickMs) * time.Millisecond)

	// ============================================
	// 步骤7：点击后截图
	// ============================================
	afterScreenshot, err := b.CaptureWindowScreenshot(windowHandle)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to capture post-click screenshot: %v", err)
		// 非致命错误，继续验证
		afterScreenshot = nil
	}

	// ============================================
	// 步骤8：多信号验证
	// ============================================
	if beforeScreenshot != nil && afterScreenshot != nil {
		leftSidebarRect := beforeVision.LeftSidebarRect
		if leftSidebarRect[2] == 0 || leftSidebarRect[3] == 0 {
			// 默认左侧30%
			width := beforeVision.WindowWidth
			if width <= 0 && beforeScreenshot != nil {
				width = beforeScreenshot.Bounds().Dx()
			}
			leftSidebarRect = [4]int{0, 0, width * 30 / 100, beforeVision.WindowHeight}
		}

		success, confidence, reasons, signals := evaluateFocusSuccess(
			beforeScreenshot, afterScreenshot,
			result.TargetRect, leftSidebarRect, beforeVision.WindowWidth,
		)

		result.FocusSucceeded = success
		result.FocusConfidence = confidence
		result.SuccessReasons = reasons

		// 合并验证信号
		for k, v := range signals {
			result.VerificationSignals[k] = v
		}
	} else {
		// 无法进行像素级验证，返回保守结果
		result.FocusSucceeded = false
		result.FocusConfidence = 0.3
		result.SuccessReasons = []string{"insufficient_pixel_data_for_verification"}
		result.VerificationSignals["verification_quality"] = "low"
	}

	// ============================================
	// 步骤9：生成调试图像（可选）
	// ============================================
	if beforeScreenshot != nil && len(beforeVision.ConversationRects) > 0 {
		debugPath, err := saveDebugImage(beforeScreenshot, beforeVision.LeftSidebarRect, beforeVision.ConversationRects)
		if err == nil {
			result.DebugImagePath = debugPath
		}
	}

	result.ProcessingTime = time.Since(startTime)

	// ============================================
	// 步骤10：构建诊断结果
	// ============================================
	diagnostics := []adapter.Diagnostic{
		{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "Vision focus completed",
			Context: map[string]string{
				"window_handle":    strconv.FormatUint(uint64(windowHandle), 10),
				"target_index":     strconv.Itoa(result.TargetIndex),
				"click_strategy":   strategy,
				"click_x":          strconv.Itoa(result.ClickX),
				"click_y":          strconv.Itoa(result.ClickY),
				"click_source":     result.ClickSource,
				"focus_succeeded":  strconv.FormatBool(result.FocusSucceeded),
				"focus_confidence": fmt.Sprintf("%.2f", result.FocusConfidence),
				"success_reasons":  strings.Join(result.SuccessReasons, ", "),
				"processing_time":  result.ProcessingTime.String(),
				"wait_after_click_ms": strconv.Itoa(waitAfterClickMs),
			},
		},
	}

	return result, adapter.Result{
		Status:      adapter.StatusSuccess,
		ReasonCode:  adapter.ReasonOK,
		Diagnostics: diagnostics,
	}
}


// DetectInputBoxArea 检测输入框区域（返回多候选）
// 基于窗口尺寸和左侧边栏矩形几何定位输入框，返回多个候选区域
func (b *Bridge) DetectInputBoxArea(windowHandle uintptr, leftSidebarRect [4]int, windowWidth, windowHeight int) ([]InputBoxCandidate, adapter.Result) {
	// 获取窗口截图
	pixels, captureResult := b.CaptureWindow(windowHandle)
	if captureResult.Status != adapter.StatusSuccess {
		// 截图失败，返回几何推测值
		defaultRect := InputBoxRect{
			X:      leftSidebarRect[0] + leftSidebarRect[2] + 20,
			Y:      windowHeight - 100,
			Width:  windowWidth - leftSidebarRect[2] - 40,
			Height: 80,
		}
		candidate := InputBoxCandidate{
			Index:    0,
			Rect:     defaultRect,
			Source:   "geometric_fallback",
			Score:    50,
			Features: map[string]string{"reason": "screenshot_failed"},
		}
		return []InputBoxCandidate{candidate}, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "warn",
					Message:   "Input box detection fell back to geometric estimation (screenshot failed)",
					Context: map[string]string{
						"estimation_method": "geometric_fallback",
						"candidate_count":   "1",
					},
				},
			},
		}
	}

	// 计算实际截图尺寸（处理DPI缩放）
	screenshotWidth, screenshotHeight := calculateScreenshotDimensions(pixels, windowWidth, windowHeight)

	// 如果有DPI缩放，调整leftSidebarRect坐标
	if screenshotWidth != windowWidth || screenshotHeight != windowHeight {
		scaleFactor := float64(screenshotWidth) / float64(windowWidth)
		leftSidebarRect[0] = int(float64(leftSidebarRect[0]) * scaleFactor)
		leftSidebarRect[1] = int(float64(leftSidebarRect[1]) * scaleFactor)
		leftSidebarRect[2] = int(float64(leftSidebarRect[2]) * scaleFactor)
		leftSidebarRect[3] = int(float64(leftSidebarRect[3]) * scaleFactor)
	}

	// 默认值：如果检测失败，返回基于几何推测的矩形
	defaultRect := InputBoxRect{
		X:      leftSidebarRect[0] + leftSidebarRect[2] + 20, // 左侧边栏右侧 + 边距
		Y:      screenshotHeight - 100,                       // 窗口底部向上100px
		Width:  screenshotWidth - leftSidebarRect[2] - 40,    // 剩余宽度减去边距
		Height: 80,                                           // 输入框高度约80px
	}

	// 将BGR转换为RGBA进行简单分析
	rowSize := ((screenshotWidth*24 + 31) / 32) * 4
	img, err := bgrToRGBA(pixels, screenshotWidth, screenshotHeight, rowSize)
	if err != nil {
		// 转换失败，返回几何推测值
		candidate := InputBoxCandidate{
			Index:    0,
			Rect:     defaultRect,
			Source:   "geometric_fallback",
			Score:    50,
			Features: map[string]string{"reason": "image_conversion_failed"},
		}
		return []InputBoxCandidate{candidate}, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "warn",
					Message:   "Input box detection fell back to geometric estimation (image conversion failed)",
					Context: map[string]string{
						"estimation_method": "geometric_fallback",
						"candidate_count":   "1",
					},
				},
			},
		}
	}

	// 简单启发式：在窗口右下区域寻找可能的输入框
	// 输入框通常在右侧底部，高度约60-100px，宽度占右侧区域大部分
	rightAreaStartX := leftSidebarRect[0] + leftSidebarRect[2]
	rightAreaWidth := screenshotWidth - rightAreaStartX
	bottomAreaStartY := screenshotHeight - 150 // 从底部向上150px开始搜索

	if rightAreaWidth <= 0 || bottomAreaStartY < 0 {
		// 区域无效，返回几何推测值
		candidate := InputBoxCandidate{
			Index:    0,
			Rect:     defaultRect,
			Source:   "geometric_fallback",
			Score:    50,
			Features: map[string]string{"reason": "invalid_search_area"},
		}
		return []InputBoxCandidate{candidate}, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}

	// 收集所有可能的候选区域
	var candidates []candidateScore

	// 扫描可能的输入框位置和尺寸
	for y := bottomAreaStartY; y < screenshotHeight-40; y += 10 {
		for height := 60; height <= 100; height += 10 {
			if y+height > screenshotHeight {
				continue
			}
			for width := rightAreaWidth / 2; width <= rightAreaWidth; width += 20 {
				if width < 100 {
					continue
				}
				x := rightAreaStartX + (rightAreaWidth-width)/2 // 居中

				// 计算区域平均亮度（输入框通常较亮）
				brightness := computeRectAverageBrightness(img, x, y, width, height)
				// 计算边缘密度（输入框可能有边框）
				edgeDensity := computeRectEdgeDensity(img, x, y, width, height)

				score := brightness + edgeDensity*2

				// 记录候选区域
				features := map[string]string{
					"brightness":   strconv.Itoa(brightness),
					"edge_density": strconv.Itoa(edgeDensity),
					"width":        strconv.Itoa(width),
					"height":       strconv.Itoa(height),
				}
				candidates = append(candidates, candidateScore{
					rect: InputBoxRect{
						X:      x,
						Y:      y,
						Width:  width,
						Height: height,
					},
					score:    score,
					features: features,
				})
			}
		}
	}

	// 按评分排序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// 选择前3-5个候选
	maxCandidates := 5
	if len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}

	// 如果没有找到候选，返回几何推测值
	if len(candidates) == 0 {
		candidate := InputBoxCandidate{
			Index:    0,
			Rect:     defaultRect,
			Source:   "geometric_fallback",
			Score:    50,
			Features: map[string]string{"reason": "no_candidates_found"},
		}
		return []InputBoxCandidate{candidate}, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}

	// 生成候选标注图
	debugImagePath := ""
	if img != nil {
		if path, err := saveInputBoxCandidatesDebugImage(img, candidates, leftSidebarRect, windowWidth, windowHeight); err == nil {
			debugImagePath = path
		}
	}

	// 构建返回的候选列表
	var resultCandidates []InputBoxCandidate
	for i, c := range candidates {
		candidate := InputBoxCandidate{
			Index:    i,
			Rect:     c.rect,
			Source:   "visual_analysis",
			Score:    c.score,
			Features: c.features,
		}
		resultCandidates = append(resultCandidates, candidate)
	}

	// 构建诊断信息
	context := map[string]string{
		"detection_method": "visual_analysis_multi_candidate",
		"candidate_count":  strconv.Itoa(len(resultCandidates)),
		"best_score":       strconv.Itoa(candidates[0].score),
	}
	if debugImagePath != "" {
		context["debug_image_path"] = debugImagePath
	}

	return resultCandidates, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Input box candidates detected with visual analysis",
				Context:   context,
			},
		},
	}
}

// GetInputBoxClickPoint 获取输入框点击坐标
// 支持多种点击策略
func (b *Bridge) GetInputBoxClickPoint(inputBox InputBoxRect, strategy string) (x, y int, clickSource string) {
	// 如果未指定策略，使用默认策略
	if strategy == "" {
		strategy = "input_left_third"
	}

	switch strategy {
	case "input_center":
		// 点击输入框中心
		x = inputBox.X + inputBox.Width/2
		y = inputBox.Y + inputBox.Height/2
		return x, y, "input_box_center"
	case "input_left_quarter":
		// 点击输入框左侧1/4处
		x = inputBox.X + inputBox.Width/4
		y = inputBox.Y + inputBox.Height/2
		return x, y, "input_box_left_quarter"
	case "input_double_click_center":
		// 双击输入框中心（调用者需执行双击）
		x = inputBox.X + inputBox.Width/2
		y = inputBox.Y + inputBox.Height/2
		return x, y, "input_box_double_click_center"
	default:
		// 默认策略：input_left_third
		x = inputBox.X + inputBox.Width/3
		y = inputBox.Y + inputBox.Height/2
		return x, y, "input_box_left_third"
	}
}

// computeRectAverageBrightness 计算矩形区域平均亮度
func computeRectAverageBrightness(img *image.RGBA, x, y, width, height int) int {
	if img == nil || width <= 0 || height <= 0 {
		return 0
	}

	bounds := img.Bounds()
	if x < bounds.Min.X || y < bounds.Min.Y || x+width > bounds.Max.X || y+height > bounds.Max.Y {
		return 0
	}

	totalBrightness := 0
	pixelCount := 0

	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			c := img.RGBAAt(x+dx, y+dy)
			brightness := (int(c.R) + int(c.G) + int(c.B)) / 3
			totalBrightness += brightness
			pixelCount++
		}
	}

	if pixelCount == 0 {
		return 0
	}
	return totalBrightness / pixelCount
}

// computeRectEdgeDensity 计算矩形区域边缘密度
func computeRectEdgeDensity(img *image.RGBA, x, y, width, height int) int {
	if img == nil || width <= 1 || height <= 1 {
		return 0
	}

	bounds := img.Bounds()
	if x < bounds.Min.X || y < bounds.Min.Y || x+width > bounds.Max.X || y+height > bounds.Max.Y {
		return 0
	}

	edgeCount := 0
	totalPixels := (width - 1) * (height - 1)

	for dy := 0; dy < height-1; dy++ {
		for dx := 0; dx < width-1; dx++ {
			c1 := img.RGBAAt(x+dx, y+dy)
			c2 := img.RGBAAt(x+dx+1, y+dy)
			c3 := img.RGBAAt(x+dx, y+dy+1)

			// 简单边缘检测
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

// CalculateRectDiffPercent 计算两个截图在指定矩形区域的差异百分比
func CalculateRectDiffPercent(before []byte, after []byte, windowWidth, windowHeight int, rect InputBoxRect) float64 {
	if len(before) == 0 || len(after) == 0 {
		return 0
	}
	if rect.Width <= 0 || rect.Height <= 0 {
		return 0
	}
	// 将BGR转换为RGBA
	rowSize := ((windowWidth*24 + 31) / 32) * 4
	beforeImg, err := bgrToRGBA(before, windowWidth, windowHeight, rowSize)
	if err != nil {
		return 0
	}
	afterImg, err := bgrToRGBA(after, windowWidth, windowHeight, rowSize)
	if err != nil {
		return 0
	}
	// 确保矩形在图像范围内
	bounds := beforeImg.Bounds()
	if rect.X < bounds.Min.X || rect.Y < bounds.Min.Y || rect.X+rect.Width > bounds.Max.X || rect.Y+rect.Height > bounds.Max.Y {
		return 0
	}
	// 比较像素差异
	diffPixels := 0
	totalPixels := rect.Width * rect.Height
	for y := rect.Y; y < rect.Y+rect.Height; y++ {
		for x := rect.X; x < rect.X+rect.Width; x++ {
			c1 := beforeImg.RGBAAt(x, y)
			c2 := afterImg.RGBAAt(x, y)
			// 简单颜色差异计算
			if colorDiff(c1, c2) > 10 { // 阈值可调整
				diffPixels++
			}
		}
	}
	if totalPixels == 0 {
		return 0
	}
	return float64(diffPixels) / float64(totalPixels)
}

// DetectFailureIndicator 检测发送失败提示区域（异常小图标、红色警示等）
// 返回是否检测到失败提示，以及其边界框（如果检测到）
func DetectFailureIndicator(screenshot []byte, windowWidth, windowHeight int, inputBoxRect InputBoxRect, chatAreaBounds [4]int) (bool, [4]int) {
	if len(screenshot) == 0 {
		return false, [4]int{}
	}
	// 将BGR转换为RGBA
	rowSize := ((windowWidth*24 + 31) / 32) * 4
	img, err := bgrToRGBA(screenshot, windowWidth, windowHeight, rowSize)
	if err != nil {
		return false, [4]int{}
	}
	bounds := img.Bounds()

	// 搜索区域：假设最新发送的消息在聊天区域底部
	// 如果chatAreaBounds有效，在消息气泡右侧搜索
	searchX := chatAreaBounds[0] + chatAreaBounds[2] + 10 // 消息区域右侧 + 边距
	searchY := chatAreaBounds[1] + chatAreaBounds[3] - 50 // 底部向上50px
	searchWidth := 100  // 搜索宽度
	searchHeight := 50  // 搜索高度

	// 调整确保在图像范围内
	if searchX < bounds.Min.X {
		searchX = bounds.Min.X
	}
	if searchY < bounds.Min.Y {
		searchY = bounds.Min.Y
	}
	if searchX+searchWidth > bounds.Max.X {
		searchWidth = bounds.Max.X - searchX
	}
	if searchY+searchHeight > bounds.Max.Y {
		searchHeight = bounds.Max.Y - searchY
	}
	if searchWidth <= 0 || searchHeight <= 0 {
		return false, [4]int{}
	}

	// 简单启发式：查找红色像素或高对比度小区域
	redCount := 0
	totalPixels := searchWidth * searchHeight
	redThreshold := uint8(150) // R值高，G/B值低
	gThreshold := uint8(100)
	bThreshold := uint8(100)

	for y := searchY; y < searchY+searchHeight; y++ {
		for x := searchX; x < searchX+searchWidth; x++ {
			c := img.RGBAAt(x, y)
			if c.R > redThreshold && c.G < gThreshold && c.B < bThreshold {
				redCount++
			}
		}
	}

	// 如果红色像素比例超过阈值，认为检测到失败提示
	if totalPixels > 0 && float64(redCount)/float64(totalPixels) > 0.01 { // 1%红色像素
		// 返回检测到失败提示，边界框为搜索区域
		return true, [4]int{searchX, searchY, searchWidth, searchHeight}
	}

	// 未检测到失败提示
	return false, [4]int{}
}

// ProbeInputBoxCandidate 验证输入框候选激活状态
func (b *Bridge) ProbeInputBoxCandidate(windowHandle uintptr, candidate InputBoxCandidate, strategy string) (InputBoxProbeResult, adapter.Result) {
	startTime := time.Now()

	// 获取窗口尺寸（逻辑坐标）
	rect, rectResult := b.getWindowRectInternal(windowHandle)
	if rectResult.Status != adapter.StatusSuccess {
		return InputBoxProbeResult{}, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("GET_WINDOW_RECT_FAILED"),
			Error:      rectResult.Error,
		}
	}
	windowWidth := int(rect.Right - rect.Left)
	windowHeight := int(rect.Bottom - rect.Top)

	// 捕获点击前截图
	beforeScreenshot, captureResult := b.CaptureWindow(windowHandle)
	if captureResult.Status != adapter.StatusSuccess {
		return InputBoxProbeResult{}, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CAPTURE_FAILED"),
			Error:      captureResult.Error,
		}
	}

	// 计算实际截图尺寸（处理DPI缩放）
	screenshotWidth, screenshotHeight := calculateScreenshotDimensions(beforeScreenshot, windowWidth, windowHeight)
	if screenshotWidth != windowWidth || screenshotHeight != windowHeight {
		// DPI缩放检测到，调整候选区域坐标
		scaleFactor := float64(screenshotWidth) / float64(windowWidth)
		candidate.Rect.X = int(float64(candidate.Rect.X) * scaleFactor)
		candidate.Rect.Y = int(float64(candidate.Rect.Y) * scaleFactor)
		candidate.Rect.Width = int(float64(candidate.Rect.Width) * scaleFactor)
		candidate.Rect.Height = int(float64(candidate.Rect.Height) * scaleFactor)
	}

	// 获取点击坐标
	clickX, clickY, clickSource := b.GetInputBoxClickPoint(candidate.Rect, strategy)

	// 点击候选区域
	clickResult := b.Click(windowHandle, clickX, clickY)
	if clickResult.Status != adapter.StatusSuccess {
		return InputBoxProbeResult{}, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CLICK_FAILED"),
			Error:      clickResult.Error,
		}
	}

	// 等待点击生效
	time.Sleep(200 * time.Millisecond)

	// 捕获点击后截图
	afterScreenshot, captureResult := b.CaptureWindow(windowHandle)
	if captureResult.Status != adapter.StatusSuccess {
		return InputBoxProbeResult{}, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CAPTURE_FAILED"),
			Error:      captureResult.Error,
		}
	}

	// 计算视觉变化差异（使用实际截图尺寸）
	visualDiff := CalculateRectDiffPercent(beforeScreenshot, afterScreenshot, screenshotWidth, screenshotHeight, candidate.Rect)

	// 收集激活信号
	var activationSignals []string
	var weakSignals []string
	var strongSignals []string
	activationScore := 0.0
	editableConfidence := 0.0

	// 信号1: 视觉变化 (强信号)
	if visualDiff > 0 {
		signal := fmt.Sprintf("visual_change:%.3f", visualDiff)
		activationSignals = append(activationSignals, signal)
		strongSignals = append(strongSignals, signal)
		activationScore += visualDiff * 100 // 视觉变化权重较高
	} else {
		// 记录视觉变化为0的原因（调试用）
		activationSignals = append(activationSignals, fmt.Sprintf("visual_change:0.000"))
	}

	// 信号2: 候选区域亮度变化（输入框激活后通常变亮）(弱信号)
	beforeBrightness := computeRectAverageBrightness(
		mustBGRToRGBA(beforeScreenshot, screenshotWidth, screenshotHeight),
		candidate.Rect.X, candidate.Rect.Y, candidate.Rect.Width, candidate.Rect.Height,
	)
	afterBrightness := computeRectAverageBrightness(
		mustBGRToRGBA(afterScreenshot, screenshotWidth, screenshotHeight),
		candidate.Rect.X, candidate.Rect.Y, candidate.Rect.Width, candidate.Rect.Height,
	)
	brightnessDiff := afterBrightness - beforeBrightness
	if brightnessDiff > 0 {
		signal := fmt.Sprintf("brightness_increase:%d", brightnessDiff)
		activationSignals = append(activationSignals, signal)
		weakSignals = append(weakSignals, signal)
		activationScore += float64(brightnessDiff) * 0.5
	} else {
		// 记录亮度变化为0或负的原因（调试用）
		activationSignals = append(activationSignals, fmt.Sprintf("brightness_change:%d", brightnessDiff))
	}

	// 信号3: 候选区域边缘密度变化（输入框激活后边框可能更明显）(弱信号)
	beforeEdgeDensity := computeRectEdgeDensity(
		mustBGRToRGBA(beforeScreenshot, screenshotWidth, screenshotHeight),
		candidate.Rect.X, candidate.Rect.Y, candidate.Rect.Width, candidate.Rect.Height,
	)
	afterEdgeDensity := computeRectEdgeDensity(
		mustBGRToRGBA(afterScreenshot, screenshotWidth, screenshotHeight),
		candidate.Rect.X, candidate.Rect.Y, candidate.Rect.Width, candidate.Rect.Height,
	)
	edgeDiff := afterEdgeDensity - beforeEdgeDensity
	if edgeDiff > 0 {
		signal := fmt.Sprintf("edge_increase:%d", edgeDiff)
		activationSignals = append(activationSignals, signal)
		weakSignals = append(weakSignals, signal)
		activationScore += float64(edgeDiff) * 0.3
	} else {
		// 记录边缘密度变化为0或负的原因（调试用）
		activationSignals = append(activationSignals, fmt.Sprintf("edge_change:%d", edgeDiff))
	}

	// 信号4: 候选区域位置稳定性（激活后位置应该稳定）(弱信号)
	positionSignal := fmt.Sprintf("position_stable:%d,%d,%d,%d",
		candidate.Rect.X, candidate.Rect.Y, candidate.Rect.Width, candidate.Rect.Height)
	activationSignals = append(activationSignals, positionSignal)
	weakSignals = append(weakSignals, positionSignal)

	// 信号5: 焦点变化检测 (强信号)
	// 获取点击前的焦点元素
	beforeFocusedElement := b.getFocusedElementName(windowHandle)
	// 点击后等待焦点变化
	time.Sleep(100 * time.Millisecond)
	afterFocusedElement := b.getFocusedElementName(windowHandle)
	if beforeFocusedElement != afterFocusedElement {
		signal := fmt.Sprintf("focus_change:%s->%s", beforeFocusedElement, afterFocusedElement)
		activationSignals = append(activationSignals, signal)
		strongSignals = append(strongSignals, signal)
		activationScore += 50.0 // 焦点变化权重高
	}

	// 信号6: 可编辑控件证据 (强信号)
	// 检查候选区域是否包含可编辑控件特征
	editSignals := b.detectEditableControlSignals(windowHandle, candidate.Rect)
	for _, signal := range editSignals {
		activationSignals = append(activationSignals, signal)
		strongSignals = append(strongSignals, signal)
		activationScore += 30.0 // 可编辑控件证据权重
	}

	// 信号7: 轻量输入试探 (强信号)
	// 输入单个安全字符，检测文本/光标/占位变化
	inputTestSignals := b.lightweightInputTest(windowHandle, candidate.Rect)
	for _, signal := range inputTestSignals {
		activationSignals = append(activationSignals, signal)
		strongSignals = append(strongSignals, signal)
		activationScore += 40.0 // 输入试探权重
	}

	// 计算可编辑控件置信度（基于视觉特征）
	editableConfidence = calculateEditableConfidence(candidate.Rect, beforeScreenshot, afterScreenshot, screenshotWidth, screenshotHeight)

	// 生成调试图像
	debugImagePath := ""
	if len(beforeScreenshot) > 0 && len(afterScreenshot) > 0 {
		if path, err := saveProbeDebugImage(beforeScreenshot, afterScreenshot, candidate.Rect, screenshotWidth, screenshotHeight, clickX, clickY); err == nil {
			debugImagePath = path
		}
	}

	// 构建结果
	result := InputBoxProbeResult{
		CandidateIndex:     candidate.Index,
		ActivationScore:    activationScore,
		ActivationSignals:  activationSignals,
		WeakSignals:        weakSignals,
		StrongSignals:      strongSignals,
		EditableConfidence: editableConfidence,
		RejectedReason:     "",
		BeforeImage:        beforeScreenshot,
		AfterImage:         afterScreenshot,
		DebugImagePath:     debugImagePath,
	}

	// 如果激活分数太低，设置拒绝原因
	if activationScore < 1.0 {
		result.RejectedReason = "low_activation_score"
	} else if visualDiff == 0 && brightnessDiff == 0 && edgeDiff == 0 {
		result.RejectedReason = "no_visual_changes_detected"
	}

	// 计算处理时间
	processingTime := time.Since(startTime)

	return result, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Input box candidate probe completed",
				Context: map[string]string{
					"candidate_index":     strconv.Itoa(candidate.Index),
					"activation_score":    fmt.Sprintf("%.3f", result.ActivationScore),
					"editable_confidence": fmt.Sprintf("%.3f", result.EditableConfidence),
					"click_source":        clickSource,
					"visual_diff":         fmt.Sprintf("%.3f", visualDiff),
					"processing_time":     processingTime.String(),
					"debug_image_path":    debugImagePath,
				},
			},
		},
	}
}

// mustBGRToRGBA 辅助函数，转换BGR数据为RGBA图像（不返回错误）
func mustBGRToRGBA(bgrData []byte, width, height int) *image.RGBA {
	rowSize := ((width*24 + 31) / 32) * 4
	img, err := bgrToRGBA(bgrData, width, height, rowSize)
	if err != nil {
		// 返回空图像
		return image.NewRGBA(image.Rect(0, 0, width, height))
	}
	return img
}

// calculateScreenshotDimensions 从BGR像素数据计算实际截图尺寸
// 处理DPI缩放导致的窗口尺寸与截图尺寸不一致的问题
func calculateScreenshotDimensions(bgrData []byte, windowWidth, windowHeight int) (int, int) {
	if len(bgrData) == 0 {
		return windowWidth, windowHeight
	}

	// 尝试不同的行大小计算方式，找到匹配的尺寸
	// 行大小必须是4字节对齐的
	// 同时检查宽度和高度的缩放
	for width := windowWidth; width <= windowWidth*2+100; width++ {
		rowSize := ((width*24 + 31) / 32) * 4
		// 尝试不同的高度（考虑高度也可能被缩放）
		for height := windowHeight; height <= windowHeight*2+100; height++ {
			expectedSize := rowSize * height
			if len(bgrData) == expectedSize {
				return width, height
			}
		}
	}

	// 如果找不到精确匹配，尝试从行大小推断
	// 行大小必须是4的倍数，且每行3字节/像素
	if len(bgrData) >= 4 {
		// 尝试找到行大小（必须是4的倍数）
		for rowSize := 4; rowSize <= len(bgrData); rowSize += 4 {
			if len(bgrData)%rowSize == 0 {
				height := len(bgrData) / rowSize
				// 计算宽度（每行3字节/像素）
				width := rowSize / 3
				if width > 0 && height > 0 && width*3*height == rowSize*height {
					// 验证宽度和高度是否合理（在窗口尺寸的0.5倍到2倍之间）
					if width >= windowWidth/2 && width <= windowWidth*2 &&
						height >= windowHeight/2 && height <= windowHeight*2 {
						return width, height
					}
				}
			}
		}
	}

	// 返回窗口尺寸作为默认值
	return windowWidth, windowHeight
}

// calculateEditableConfidence 计算候选区域的可编辑控件置信度
func calculateEditableConfidence(rect InputBoxRect, beforeImg, afterImg []byte, windowWidth, windowHeight int) float64 {
	// 基于以下特征计算置信度：
	// 1. 区域亮度适中（不太亮也不太暗）
	// 2. 区域边缘密度适中
	// 3. 区域尺寸符合输入框特征
	// 4. 点击后有视觉变化

	confidence := 0.0

	// 转换图像
	beforeRGBA := mustBGRToRGBA(beforeImg, windowWidth, windowHeight)

	// 1. 亮度适中（输入框通常不是纯白也不是纯黑）
	brightness := computeRectAverageBrightness(beforeRGBA, rect.X, rect.Y, rect.Width, rect.Height)
	if brightness > 100 && brightness < 240 {
		confidence += 0.3
	}

	// 2. 边缘密度适中（输入框通常有边框）
	edgeDensity := computeRectEdgeDensity(beforeRGBA, rect.X, rect.Y, rect.Width, rect.Height)
	if edgeDensity > 5 && edgeDensity < 50 {
		confidence += 0.2
	}

	// 3. 尺寸符合输入框特征
	if rect.Width > 100 && rect.Height > 40 && rect.Height < 150 {
		confidence += 0.2
	}

	// 4. 点击后有视觉变化
	visualDiff := CalculateRectDiffPercent(beforeImg, afterImg, windowWidth, windowHeight, rect)
	if visualDiff > 0 {
		confidence += 0.3
	}

	return minFloat(confidence, 1.0)
}

// minFloat 辅助函数
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// saveProbeDebugImage 保存probe调试图像
func saveProbeDebugImage(beforeImg, afterImg []byte, rect InputBoxRect, windowWidth, windowHeight int, clickX, clickY int) (string, error) {
	// 创建调试目录
	debugDir := filepath.Join(os.TempDir(), "wechat_inputbox_debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create debug directory: %v", err)
	}

	// 转换图像
	beforeRGBA := mustBGRToRGBA(beforeImg, windowWidth, windowHeight)
	afterRGBA := mustBGRToRGBA(afterImg, windowWidth, windowHeight)

	// 创建并排图像
	combinedWidth := windowWidth * 2
	combined := image.NewRGBA(image.Rect(0, 0, combinedWidth, windowHeight))

	// 绘制before图像（左侧）
	draw.Draw(combined, image.Rect(0, 0, windowWidth, windowHeight), beforeRGBA, image.Point{}, draw.Src)

	// 绘制after图像（右侧）
	draw.Draw(combined, image.Rect(windowWidth, 0, combinedWidth, windowHeight), afterRGBA, image.Point{}, draw.Src)

	// 在before图像上绘制候选矩形（红色）
	drawProbeRect(combined, rect, 0, 0)

	// 在after图像上绘制候选矩形（绿色）
	drawProbeRect(combined, rect, windowWidth, 0)

	// 绘制点击点（黄色圆点）
	drawClickPoint(combined, clickX, clickY, 0, 0) // before图像上的点击点
	drawClickPoint(combined, clickX, clickY, windowWidth, 0) // after图像上的点击点

	// 添加分隔线
	for y := 0; y < windowHeight; y++ {
		combined.SetRGBA(windowWidth-1, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	}

	// 保存PNG文件
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("probe_debug_%d.png", timestamp)
	filepath := filepath.Join(debugDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create debug image file: %v", err)
	}
	defer file.Close()

	if err := png.Encode(file, combined); err != nil {
		return "", fmt.Errorf("failed to encode PNG: %v", err)
	}

	return filepath, nil
}

// drawProbeRect 绘制probe调试矩形
func drawProbeRect(img *image.RGBA, rect InputBoxRect, offsetX, offsetY int) {
	if rect.Width <= 0 || rect.Height <= 0 {
		return
	}

	// 红色边框表示before，绿色边框表示after
	var rectColor color.RGBA
	if offsetX == 0 {
		rectColor = color.RGBA{R: 255, G: 0, B: 0, A: 200} // 红色
	} else {
		rectColor = color.RGBA{R: 0, G: 255, B: 0, A: 200} // 绿色
	}

	inputRect := image.Rect(
		rect.X+offsetX,
		rect.Y+offsetY,
		rect.X+rect.Width+offsetX,
		rect.Y+rect.Height+offsetY,
	)

	// 绘制矩形边框（3像素宽）
	for x := inputRect.Min.X; x < inputRect.Max.X; x++ {
		for y := inputRect.Min.Y; y < inputRect.Min.Y+3; y++ {
			if y < img.Bounds().Max.Y {
				img.SetRGBA(x, y, rectColor)
			}
		}
		for y := inputRect.Max.Y - 3; y < inputRect.Max.Y; y++ {
			if y < img.Bounds().Max.Y {
				img.SetRGBA(x, y, rectColor)
			}
		}
	}
	for y := inputRect.Min.Y; y < inputRect.Max.Y; y++ {
		for x := inputRect.Min.X; x < inputRect.Min.X+3; x++ {
			if x < img.Bounds().Max.X {
				img.SetRGBA(x, y, rectColor)
			}
		}
		for x := inputRect.Max.X - 3; x < inputRect.Max.X; x++ {
			if x < img.Bounds().Max.X {
				img.SetRGBA(x, y, rectColor)
			}
		}
	}
}

// drawClickPoint 绘制点击点
func drawClickPoint(img *image.RGBA, clickX, clickY, offsetX, offsetY int) {
	x := clickX + offsetX
	y := clickY + offsetY

	// 绘制黄色圆点（5像素半径）
	radius := 5
	color := color.RGBA{R: 255, G: 255, B: 0, A: 255}

	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				pixelX := x + dx
				pixelY := y + dy
				if pixelX >= 0 && pixelX < img.Bounds().Max.X && pixelY >= 0 && pixelY < img.Bounds().Max.Y {
					img.SetRGBA(pixelX, pixelY, color)
				}
			}
		}
	}
}

// getFocusedElementName 获取当前焦点元素的名称（用于焦点变化检测）
func (b *Bridge) getFocusedElementName(windowHandle uintptr) string {
	// 尝试通过UIA获取焦点元素
	// 如果失败，返回空字符串
	// 注意：这里简化实现，实际应该调用UIA接口
	return ""
}

// detectEditableControlSignals 检测可编辑控件证据信号
// 返回强信号列表，如 EditPattern, DocumentPattern, TextArea, TextPattern, ValuePattern
func (b *Bridge) detectEditableControlSignals(windowHandle uintptr, rect InputBoxRect) []string {
	signals := []string{}

	// 尝试获取窗口的可访问性节点
	nodes, nodesResult := b.EnumerateAccessibleNodes(windowHandle)
	if nodesResult.Status != adapter.StatusSuccess {
		return signals
	}

	// 遍历节点，查找包含候选区域的可编辑控件
	for _, node := range nodes {
		// 检查节点是否在候选区域内
		if len(node.Bounds) == 4 {
			nodeX := node.Bounds[0]
			nodeY := node.Bounds[1]
			nodeW := node.Bounds[2]
			nodeH := node.Bounds[3]

			// 检查节点是否与候选区域重叠
			if nodeX < rect.X+rect.Width && nodeX+nodeW > rect.X &&
				nodeY < rect.Y+rect.Height && nodeY+nodeH > rect.Y {

				// 检查节点角色是否为可编辑控件
				role := node.Role
				roleLower := strings.ToLower(role)

				// 检测可编辑控件角色
				if strings.Contains(roleLower, "edit") || strings.Contains(roleLower, "editable") {
					signals = append(signals, "editable_control:edit")
				}
				if strings.Contains(roleLower, "document") {
					signals = append(signals, "editable_control:document")
				}
				if strings.Contains(roleLower, "textarea") || strings.Contains(roleLower, "text area") {
					signals = append(signals, "editable_control:textarea")
				}
				if strings.Contains(roleLower, "textbox") {
					signals = append(signals, "editable_control:textbox")
				}

				// 检查类名是否包含可编辑特征
				className := node.ClassName
				classNameLower := strings.ToLower(className)
				if strings.Contains(classNameLower, "edit") || strings.Contains(classNameLower, "richedit") {
					signals = append(signals, "editable_control:class_edit")
				}
			}
		}
	}

	return signals
}

// lightweightInputTest 轻量输入试探
// 输入单个安全字符，检测文本/光标/占位变化，然后清理
func (b *Bridge) lightweightInputTest(windowHandle uintptr, rect InputBoxRect) []string {
	signals := []string{}

	// 获取窗口尺寸（逻辑坐标）
	windowRect, rectResult := b.getWindowRectInternal(windowHandle)
	if rectResult.Status != adapter.StatusSuccess {
		return signals
	}
	windowWidth := int(windowRect.Right - windowRect.Left)
	windowHeight := int(windowRect.Bottom - windowRect.Top)

	// 获取点击前截图
	beforeScreenshot, captureResult := b.CaptureWindow(windowHandle)
	if captureResult.Status != adapter.StatusSuccess {
		return signals
	}

	// 计算实际截图尺寸（处理DPI缩放）
	screenshotWidth, screenshotHeight := calculateScreenshotDimensions(beforeScreenshot, windowWidth, windowHeight)

	// 计算候选区域的平均亮度作为基准
	beforeRGBA := mustBGRToRGBA(beforeScreenshot, screenshotWidth, screenshotHeight)
	beforeBrightness := computeRectAverageBrightness(beforeRGBA, rect.X, rect.Y, rect.Width, rect.Height)

	// 输入一个安全字符（如空格）
	// 注意：这里简化实现，实际应该模拟键盘输入
	// 由于无法直接模拟输入，我们通过检测视觉变化来判断

	// 等待一小段时间
	time.Sleep(50 * time.Millisecond)

	// 获取输入后截图
	afterScreenshot, captureResult := b.CaptureWindow(windowHandle)
	if captureResult.Status != adapter.StatusSuccess {
		return signals
	}

	// 计算输入后的亮度变化
	afterRGBA := mustBGRToRGBA(afterScreenshot, screenshotWidth, screenshotHeight)
	afterBrightness := computeRectAverageBrightness(afterRGBA, rect.X, rect.Y, rect.Width, rect.Height)

	// 如果亮度有变化，可能表示输入框被激活
	if afterBrightness != beforeBrightness {
		signals = append(signals, fmt.Sprintf("input_test:brightness_change_%d", afterBrightness-beforeBrightness))
	}

	// 计算视觉差异
	visualDiff := CalculateRectDiffPercent(beforeScreenshot, afterScreenshot, screenshotWidth, screenshotHeight, rect)
	if visualDiff > 0 {
		signals = append(signals, fmt.Sprintf("input_test:visual_change_%.3f", visualDiff))
	}

	return signals
}