package main

import (
	"backend/internal/database"
	"log"

	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := database.Init(); err != nil {
		log.Fatalf("failed to initialize the database: %v", err)
	}
	// 这里开始注册路由、启动 HTTP 服务…

}
