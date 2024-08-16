package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/storage"
)

// CreateUser creates a new user
func CreateUser(c *gin.Context) {
	var user models.Users
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `INSERT INTO users (username, name, password, email, profile_picture, created_at) 
            VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING id`
	err := storage.DB.QueryRow(context.Background(), query, user.Username, user.Name, user.Password, user.Email, user.ProfilePicture).Scan(&user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User created successfully", "user_id": user.ID})
}

func SendMessage(c *gin.Context) {
	var messages models.Messages
	if err := c.ShouldBindJSON(&messages); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `INSERT INTO messages (sender_id, content, chat_room_id, is_dm) 
            VALUES ($1, $2, $3, $4) RETURNING id, timestamp`
	err := storage.DB.QueryRow(context.Background(), query, messages.SenderID, messages.Content, messages.ChatRoomID, messages.IsDM).Scan(&messages.ID, &messages.Timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully", "message_id": messages.ID, "timestamp": messages.Timestamp})
}

func GetMessages(c *gin.Context) {
	chatRoomID := c.Param("chatRoomID")

	query := `
	SELECT m.id, m.sender_id, u.username, u.name, m.content, m.timestamp, m.chat_room_id, m.is_dm 
	FROM messages m 
	JOIN users u ON m.sender_id = u.id 
	WHERE m.chat_room_id = $1
	ORDER BY m.id ASC
`
	rows, err := storage.DB.Query(context.Background(), query, chatRoomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var messages []models.Messages
	for rows.Next() {
		var msg models.Messages
		if err := rows.Scan(&msg.ID, &msg.SenderID, &msg.Sender.Username, &msg.Sender.Name, &msg.Content, &msg.Timestamp, &msg.ChatRoomID, &msg.IsDM); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		messages = append(messages, msg)
	}

	c.JSON(http.StatusOK, messages)
}

func GetChatRooms(c *gin.Context) {
	query := `SELECT id, name, description, type FROM chat_rooms`
	rows, err := storage.DB.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var chatRooms []models.ChatRooms
	for rows.Next() {
		var chatRoom models.ChatRooms
		if err := rows.Scan(&chatRoom.ID, &chatRoom.Name, &chatRoom.Description, &chatRoom.Type); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		chatRooms = append(chatRooms, chatRoom)
	}

	c.JSON(http.StatusOK, chatRooms)
}
