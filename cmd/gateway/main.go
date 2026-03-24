package main

import (
	"log"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/gateway/server"
)

func main() {
	// 创建 HTTP/WebSocket 服务器
	srv := server.NewServer(":8080")

	// 启动服务器
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
