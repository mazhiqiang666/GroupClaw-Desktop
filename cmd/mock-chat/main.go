package main

import (
	"log"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/mockchat"
)

func main() {
	// 创建 Mock Chat App
	app := mockchat.NewMockChatApp()

	// 启动 GUI（包含 HTTP 服务器）
	if err := app.RunGUI(); err != nil {
		log.Fatal(err)
	}
}
