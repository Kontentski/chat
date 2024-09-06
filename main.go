package main

import (
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kontentski/chat/internal/auth"
	"github.com/kontentski/chat/internal/handlers"
	"github.com/kontentski/chat/internal/storage"
)

func main() {
	storage.Init()
	storage.RunMigrations()

	auth.Init()

	r := gin.Default()


	r.Static("/home", "./static")

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		handlers.HandleWebSocket(c.Writer, c.Request)
	})

	// Authentication routes
	r.GET("/auth", handlers.AuthHandler)
	r.GET("/auth/callback", handlers.CallbackHandler)
	r.GET("/auth/logout", handlers.LogoutHandler)



	r.POST("/users", handlers.CreateUser)

    // Message endpoints
    r.POST("/messages", handlers.SendMessage)
	r.GET("/messages/:chatRoomID", handlers.GetMessages)
	r.DELETE("/messages/:messageID", handlers.DeleteMessage)



    // Chat room endpoints
    r.GET("/api/chatrooms", handlers.GetUserChatRooms)

	r.GET("/hello", func(c *gin.Context) {
		c.String(200, "Hello, World!")
	})
	// Start the server
	log.Println("Server started on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
