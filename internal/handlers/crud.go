package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kontentski/chat/internal/auth"
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

	c.JSON(http.StatusOK, messages)
}

func DeleteMessage(c *gin.Context) {
	messageIDStr := c.Param("messageID")
	chatRoomIDStr := c.Query("chat_room_id")
	userIDStr := c.Query("user_id")

	// Validate the request parameters
	if messageIDStr == "" || chatRoomIDStr == "" || userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters"})
		return
	}

	// Convert messageID and chatRoomID from string to int
	messageIDInt, err := strconv.Atoi(messageIDStr)
	if err != nil {
		log.Printf("Invalid messageID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid messageID"})
		return
	}
	chatRoomIDInt, err := strconv.Atoi(chatRoomIDStr)
	if err != nil {
		log.Printf("Invalid chatRoomID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chatRoomID"})
		return
	}

	userIDInt, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Printf("Invalid chatRoomID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chatRoomID"})
		return
	}

	// Convert int to uint
	messageID := uint(messageIDInt)
	chatRoomID := uint(chatRoomIDInt)
	userID := uint(userIDInt)

	// Ensure user has permission to delete the message
	if !isUserInChatRoom(uint(userID), uint(chatRoomID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "User not authorized to delete message"})
		return
	}

	// Begin a transaction to ensure atomic operation
	tx, err := storage.DB.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback(context.Background())
		} else {
			tx.Commit(context.Background())
		}
	}()

	// Delete the message
	deleteQuery := `DELETE FROM messages WHERE message_id = $1 AND chat_room_id = $2`
	_, err = tx.Exec(context.Background(), deleteQuery, messageID, chatRoomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete message"})
		return
	}

	// Commit the transaction
	err = tx.Commit(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Broadcast the deletion to other connected clients
	broadcast <- models.Messages{
		MessageID:  messageID,
		ChatRoomID: chatRoomID,
		SenderID:   userID,
		Type:       "delete",
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message deleted successfully"})
}

// getUserChatRooms retrieves the chat rooms the user is part of
func GetUserChatRooms(c *gin.Context) {
	// Assuming you're retrieving the userID from the session
	session, err := auth.Store.Get(c.Request, "auth-session")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	userID, ok := session.Values["userID"].(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Call the original function to fetch chat rooms
	chatRooms, err := FetchUserChatRooms(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat rooms"})
		return
	}

	c.JSON(http.StatusOK, chatRooms)
}

func FetchUserChatRooms(userID uint) ([]models.ChatRooms, error) {
	query := `
	SELECT cr.id, cr.name, cr.description, cr.type
	FROM chat_rooms cr
	JOIN chat_room_members crm ON cr.id = crm.chat_room_id
	WHERE crm.user_id = $1
	`

	rows, err := storage.DB.Query(context.Background(), query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chatRooms []models.ChatRooms
	for rows.Next() {
		var room models.ChatRooms
		if err := rows.Scan(&room.ID, &room.Name, &room.Description, &room.Type); err != nil {
			return nil, err
		}
		chatRooms = append(chatRooms, room)
	}

	return chatRooms, nil
}
