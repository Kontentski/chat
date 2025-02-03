package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kontentski/chat/internal/auth"
	"github.com/kontentski/chat/internal/database"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/services"
	"github.com/kontentski/chat/internal/storage"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all connections
		},
		ReadBufferSize:  1024, // Adjust buffer size as needed
		WriteBufferSize: 1024, // Adjust buffer size as needed
	}
	clients = make(map[*websocket.Conn]struct {
		userID uint
		name   string
	})
	Broadcast  = make(chan models.Messages, 100) // Broadcast channel
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	writeWait  = 10 * time.Second
)

// HandleWebSocket handles WebSocket requests
func HandleWebSocket(w http.ResponseWriter, r *http.Request, service services.ChatRoomService) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	defer conn.Close()

	go handleConnection(conn)


	session, err := auth.Store.Get(r, "auth-session")
	if err != nil {
		log.Printf("Failed to get session: %v", err)
		return
	}
	sess :=fmt.Sprintf("sessions/%v", session)
	fmt.Println(sess)

	userID, ok := session.Values["userID"].(uint)
	if !ok {
		log.Printf("userID %d\n\n\n", userID)
		log.Println("No userID in session")
		return
	}
	defer storage.UpdateLastSeen(userID)

	Username, ok := session.Values["username"].(string)
	if !ok {
		log.Println("No username in session")
		return
	}

	var name string
	err = database.DB.QueryRow(context.Background(), "SELECT name FROM users WHERE id = $1", userID).Scan(&name)
	if err != nil {
		log.Printf("Failed to find user with ID %d: %v", userID, err)
		return
	}

	clients[conn] = struct {
		userID uint
		name   string
	}{userID, name}

	log.Printf("Client connected: userID=%d", userID)

	userInfo := map[string]interface{}{
		"userID":   userID,
		"username": Username,
		"name":     name,
	}
	if err := conn.WriteJSON(userInfo); err != nil {
		log.Printf("Error sending user info: %v", err)
		return
	}
	messageStorage := &storage.PostgresRepository{DB: database.DB}

	readMessages(conn, messageStorage)
	handleMessages(service)

}
