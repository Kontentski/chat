package main

import (
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kontentski/chat/internal/auth"
	"github.com/kontentski/chat/internal/database"
	"github.com/kontentski/chat/internal/handlers"
	"github.com/kontentski/chat/internal/middleware"
	"github.com/kontentski/chat/internal/services"
	"github.com/kontentski/chat/internal/storage"
)

func main() {
	database.Init()
	database.RunMigrations()
	userRepo := &storage.PostgresRepository{
		DB: database.DB,
	}
	authRepo := &storage.RealAuth{}
	bucketStorage := &storage.GoogleUpload{}
	userService := services.NewUserChatRoomService(userRepo, authRepo, bucketStorage)
	auth.Init()

	r := gin.Default()

	r.Static("/homepage", "./homepage")

	// WebSocket endpoint
	r.GET("/ws", middleware.AuthMiddleware(auth.Store), func(c *gin.Context) {
		handlers.HandleWebSocket(c.Writer, c.Request, userService)
	})

	// Authentication routes
	r.GET("/auth", handlers.AuthHandler)
	r.GET("/auth/callback", handlers.CallbackHandler)
	r.GET("/auth/register/", handlers.RegisterHandler)
	r.POST("/auth/register", handlers.RegisterPostHandler)
	r.POST("/auth/logout", handlers.LogoutHandler)

	r.Use(middleware.AuthMiddleware(auth.Store))
	r.POST("/users", handlers.CreateUser(userService))

	// Message endpoints
	r.GET("/messages/:chatRoomID", handlers.GetMessagesHandler(*userService))
	r.DELETE("/messages/:messageID", handlers.DeleteMessageHandler(userService))

	// Chat room endpoints
	r.GET("/api/chatrooms", handlers.GetUserChatRoomsHandler(*userService))
	r.POST("/api/chatrooms/leave/:chatRoomID", handlers.LeaveTheChatRoomHandler(userService))
	r.GET("/api/chatrooms/search-users", handlers.SearchUsersHandler(*userService))
	r.POST("/api/chatrooms/add-user", handlers.AddUserHandler(*userService))
	r.POST("/api/upload-media", handlers.UploadMediaHandler(*userService))
	r.GET("/hello", func(c *gin.Context) {
		c.String(200, "Hello, World!")
	})
	// Start the server
	log.Println("Server started on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
