package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/kontentski/chat/internal/models"
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
	broadcast = make(chan models.Messages, 100) // Broadcast channel
)

// HandleWebSocket handles WebSocket requests
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Get the username from query parameters
	username := r.URL.Query().Get("username")
	if username == "" {
		log.Println("No username provided")
		return
	}

	// Lookup user ID based on username
	var userID uint
	var name string
	err = storage.DB.QueryRow(context.Background(), "SELECT id, name FROM users WHERE username = $1", username).Scan(&userID, &name)
	if err != nil {
		log.Printf("Failed to find user with username %s: %v", username, err)
		return
	}

	// Register new client with userID and chatRoomID
	clients[conn] = struct {
		userID uint
		name   string
	}{userID, name}

	token, err := GenerateToken(userID)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		return
	}

	// Send token to the client
	err = conn.WriteJSON(map[string]string{"token": token})
	if err != nil {
		log.Printf("Error sending token: %v", err)
		return
	}

	log.Printf("Client connected: userID=%d", userID)

	// Send chat room data to the client
	chatRooms, err := getUserChatRooms(userID)
	if err != nil {
		log.Printf("Error fetching chat rooms: %v", err)
		return
	}
	if err := conn.WriteJSON(chatRooms); err != nil {
		log.Printf("Error sending chat rooms: %v", err)
		return
	}
	log.Println("log send message hystory")

	// Send user info to the client
	userInfo := map[string]interface{}{
		"userID":   userID,
		"username": username,
		"name":     name,
	}
	if err := conn.WriteJSON(userInfo); err != nil {
		log.Printf("Error sending user info: %v", err)
		return
	}
	log.Println("log handle/read messages function")

	go readMessages(conn)
	handleMessages()
}
