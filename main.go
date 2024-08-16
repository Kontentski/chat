package main

import (
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kontentski/chat/internal/handlers"
	"github.com/kontentski/chat/internal/storage"
)

func main() {
	storage.Init()
	storage.RunMigrations()

	r := gin.Default()

	// Serve static files from the "static" folder at the "/static" route

	r.Static("/static", "./static")

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		handlers.HandleWebSocket(c.Writer, c.Request)
	})
	r.POST("/users", handlers.CreateUser)

    // Message endpoints
    r.POST("/messages", handlers.SendMessage)
    r.GET("/messages/:chatRoomID", handlers.GetMessages)


    // Chat room endpoints
    r.GET("/api/chatrooms", handlers.GetChatRoomsHandler)

	r.GET("/hello", func(c *gin.Context) {
		c.String(200, "Hello, World!")
	})
	// Start the server
	log.Println("Server started on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
