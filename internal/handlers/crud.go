package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/services"
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

func GetUserChatRoomsHandler(service services.UserChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {

		chatRooms, err := service.FetchUserChatRooms(c.Request)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, chatRooms)
	}
}

func GetMessagesHandler(service services.UserChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		messages, err := service.GetMessages(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, messages)
	}
}

func DeleteMessageHandler(service *services.UserChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		response, err := service.DeleteMessage(c)
		if err != nil {
			if err.Error() == "missing required parameters" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters"})
			} else {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		// Broadcast the deletion to other connected clients
		Broadcast <- models.Messages{
			MessageID:  response.MessageID,
			ChatRoomID: response.ChatRoomID,
			SenderID:   response.SenderID,
			Type:       "delete",
		}

		c.JSON(http.StatusOK, gin.H{"message": "Message deleted successfully"})
	}
}

func LeaveTheChatRoomHandler(service *services.UserChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := service.LeaveChatRoom(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User left the chat room successfully"})
	}
}

func SearchUsersHandler(service services.UserChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := service.SearchUsers(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, users)
	}
}

func AddUserHandler(service services.UserChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := service.AddUserToChatRoom(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User added successfully"})
	}
}

func UploadMediaHandler(service services.UserChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		filePath, err := service.UploadMedia(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload media"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"filePath": filePath})
	}
}

/* func FetchUserChatRooms(userID uint) ([]models.ChatRooms, error) {
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
*/
/* func SendMessage(c *gin.Context) {
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
*/
