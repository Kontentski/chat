package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

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
	err := storage.DB.QueryRow(context.Background(), query, messages.SenderID, messages.Content, messages.ChatRoomID, messages.IsDM).Scan(&messages.MessageID, &messages.Timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully", "message_id": messages.MessageID, "timestamp": messages.Timestamp})
}

func GetMessages(c *gin.Context) {
	chatRoomID := c.Param("chatRoomID")
	userID := c.Query("userID")

	// Log the parameters
	log.Printf("chatRoomID: %s, userID: %s", chatRoomID, userID)

	query := `
    SELECT m.message_id, m.sender_id, u.username, u.name, m.content, m.timestamp, m.chat_room_id, m.is_dm,
    COALESCE(r.read_at, '1970-01-01T00:00:00Z') AS read_at
    FROM messages m
    JOIN users u ON m.sender_id = u.id
    LEFT JOIN read_messages r ON m.message_id = r.message_id AND r.user_id = $1 AND m.chat_room_id = r.chat_room_id
    WHERE m.chat_room_id = $2
    ORDER BY m.timestamp ASC
    `

	rows, err := storage.DB.Query(context.Background(), query, userID, chatRoomID)
	if err != nil {
		log.Printf("Query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	log.Println("Query executed successfully")

	var messages []models.Messages

	for rows.Next() {
		var msg models.Messages
		var readAt sql.NullTime
		if err := rows.Scan(&msg.MessageID, &msg.SenderID, &msg.Sender.Username, &msg.Sender.Name, &msg.Content, &msg.Timestamp, &msg.ChatRoomID, &msg.IsDM, &readAt); err != nil {
			log.Printf("Row scan error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if readAt.Valid {
			msg.ReadAt = readAt.Time.Format(time.RFC3339)
		} else {
			msg.ReadAt = "1970-01-01T00:00:00Z" // Default for unread messages
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Rows iteration error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Messages: %v", messages)
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
