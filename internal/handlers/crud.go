package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/services"
)

//  CreateUser godoc
//	@Summary		Create a new user
//	@Description	creates a new user in the system
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body	models.Users	true	"User information"
//	@Security		ApiKeyAuth
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/users [post]
func CreateUser(service services.ChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.Users

		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := service.CreateUser(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User created successfully", "user_id": user.ID})
	}
}

//  GetUserChatRooms godoc
//	@Summary		Get user chat rooms
//	@Description	retrieve chat rooms for the authenticated user
//	@Tags			users
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{array}		models.ChatRooms
//	@Failure		401	{object}	map[string]interface{}	"Unauthenticated"
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/api/chatrooms [get]
func GetUserChatRoomsHandler(service services.ChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {

		chatRooms, err := service.FetchUserChatRooms(c.Request)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, chatRooms)
	}
}

//  GetMessages godoc
//	@Summary		Get messages
//	@Description	retrieve messages from a specific chat
//	@Tags			messages
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200			{array}		models.Messages
//	@Failure		401			{object}	map[string]interface{}	"Unauthenticated"
//	@Failure		500			{object}	map[string]interface{}
//	@Param			chatRoomID	path		int	true	"Chat Room ID"
//	@Router			/messages/{chatRoomID} [get]
func GetMessagesHandler(service services.ChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		messages, err := service.GetMessages(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, messages)
	}
}


//  DeleteMessagesHandler godoc
//	@Summary		Delete messages
//	@Description	Deletes selected message
//	@Tags			messages
//	@Produce		json
//	@Param			messageID		path	int	true	"message to delete"
//	@Param			chat_room_id	query	int	true	"chatroom id"
//	@Param			userID			query	int	true	"user id"
//	@Security		ApiKeyAuth
//	@Success		200			{object}	map[string]interface{}
//	@Failure		400,401,500	{object}	map[string]interface{}
//	@Router			/messages/{messageID} [delete]
func DeleteMessageHandler(service services.ChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		response, err := service.DeleteMessage(c)
		if err != nil {
			if err.Error() == "missing required parameters" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters"})
			} else {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
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

//  LeaveTheChatRoomHandler godoc
//	@Summaty		Leave the chat room
//	@Description	Leaves 
//	@Tags			chatrooms
//	@Produce		json
//	@Param			chatRoomID	path	int	true	"chatroom to leave"
//	@Security		ApiKeyAuth
//	@Success		200	{object}	map[string]interface{}	"message: User left the chat room successfully"
//	@Failure		401	{object}	map[string]interface{}	"Unauthorized"
//	@Failure		500	{object}	map[string]interface{}	"Internal Server Error"
//	@Router			/api/chatrooms/leave/{chatRoomID} [post]
func LeaveTheChatRoomHandler(service services.ChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := service.LeaveChatRoom(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User left the chat room successfully"})
	}
}

//  SearchUsersHandler godoc
//	@Summary		Search users
//	@Description	Search for users by query string
//	@Tags			users
//	@Produce		json
//	@Security		ApiKeyAuth
//	@param			q	query		string	true	"Search users"
//	@Success		200	{array}		services.UsersListResponse
//	@Failure		401	{object}	map[string]interface{}	"Unauthorized"
//	@Failure		500	{object}	map[string]interface{}	"Internal Server Error"
//	@Router			/api/chatrooms/search-users [get]
func SearchUsersHandler(service services.ChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := service.SearchUsers(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, users)
	}
}

// AddUserHandler godoc
//	@Summary		Add user to chat room
//	@Description	Add user to an existing chat room
//	@Tags			chatrooms
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			request	body		object					true	"Add user request"	
//	@Success		200		{object}	map[string]interface{}	"message: User added successfully"
//	@Failure		401		{object}	map[string]interface{}	"Unauthorized"
//	@Failure		500		{object}	map[string]interface{}	"Internal Server Error"
//	@Router			/api/chatrooms/add-user [post]
func AddUserHandler(service services.ChatRoomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := service.AddUserToChatRoom(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User added successfully"})
	}
}

func UploadMediaHandler(service services.ChatRoomService) gin.HandlerFunc {
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
