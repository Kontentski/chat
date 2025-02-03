package main

import (
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kontentski/chat/internal/auth"
	"github.com/kontentski/chat/internal/database"
	"github.com/kontentski/chat/internal/router"
	"github.com/kontentski/chat/internal/services"
	"github.com/kontentski/chat/internal/storage"
)

//@title chat API
//@version 1.4
//@description A chat service that uses websockets and gin
//@host localhost:8080
//@BasePath /
//@schemes http
//@securityDefinitions.apikey ApiKeyAuth
//@in header
//@name Cookie
//@description Session cookie for authentication. you need to manually add the session to the Cookie storage in your browser

func main() {
	//dependencies
	database.Init()
	database.RunMigrations()
	auth.Init()

	//repositories and services
	userRepo := &storage.PostgresRepository{DB: database.DB}
	authRepo := &storage.RealAuth{}
	bucketStorage := &storage.GoogleUpload{}
	userService := services.NewUserChatRoomService(userRepo, authRepo, bucketStorage)

	//router
	r := router.NewRouter(userService)
	r.SetupRoutes()

	// Start the server
	log.Println("Server started on :8080")
	if err := r.Start(":8080"); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
