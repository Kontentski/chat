package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kontentski/chat/internal/auth"
	"github.com/kontentski/chat/internal/database"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/storage"
)

func CreateUser(userStorage storage.UserStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.Users

		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := userStorage.CreateUser(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User created successfully", "user_id": user.ID})
	}
}

func GetMessages(userStorage storage.UserStorage, auth storage.AuthInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log the start of the request
		log.Println("GetMessages: New request received")

		// Get parameters from the request
		chatRoomID := c.Param("chatRoomID")
		userID := c.Query("userID")
		log.Printf("Parameters - chatRoomID: %s, userID: %s", chatRoomID, userID)

		// Retrieve the session from the auth interface
		sessionValues, err := auth.GetSession(c.Request)
		if err != nil {
			log.Println("Error getting session:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Session error"})
			return
		}
		log.Printf("Session retrieved: %v", sessionValues)

		// Verify the session userID and compare it with the provided userID
		sessionUserID := fmt.Sprintf("%v", sessionValues["userID"])
		if sessionUserID != userID {
			log.Println("Error casting session userID to string")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		log.Printf("Session userID: %s", sessionUserID)

		if sessionUserID != userID {
			log.Printf("Unauthorized access - session userID: %s does not match userID: %s", sessionUserID, userID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Log before querying messages
		log.Printf("Fetching messages for userID: %s in chatRoomID: %s", userID, chatRoomID)

		// Query messages
		messages, err := userStorage.GetMessages(c, userID, chatRoomID)
		if err != nil {
			log.Printf("Query error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Log the result of the query
		log.Printf("Messages retrieved: %v", messages)

		// Send the messages as a response
		log.Println("Sending messages as JSON response")
		c.JSON(http.StatusOK, messages)
	}
}

func DeleteMessage(userStorage storage.UserStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
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
		if !userStorage.IsUserInChatRoom(uint(userID), uint(chatRoomID)) {
			c.JSON(http.StatusForbidden, gin.H{"error": "User not authorized to delete message"})
			return
		}

		log.Printf("messageID %d, chatroomID %d\n\n\n\n", messageID, chatRoomID)
		err = userStorage.DeleteMessage(c, messageID, chatRoomID)
		if err != nil {
			log.Printf("error: %s", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete message"})
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

	rows, err := database.DB.Query(context.Background(), query, userID)
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

func SendMessage(c *gin.Context) {
	var messages models.Messages
	if err := c.ShouldBindJSON(&messages); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `INSERT INTO messages (sender_id, content, chat_room_id, is_dm) 
            VALUES ($1, $2, $3, $4) RETURNING id, timestamp`
	err := database.DB.QueryRow(context.Background(), query, messages.SenderID, messages.Content, messages.ChatRoomID, messages.IsDM).Scan(&messages.MessageID, &messages.Timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully", "message_id": messages.MessageID, "timestamp": messages.Timestamp})
}
