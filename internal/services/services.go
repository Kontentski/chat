package services

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/storage"
)

type UserChatRoomService struct {
	UserRepo storage.UserStorage
	AuthRepo storage.AuthInterface
}
type DeleteMessageResponse struct {
	MessageID  uint
	ChatRoomID uint
	SenderID   uint
}

// FetchUserChatRooms retrieves the user's chat rooms by processing the session.
func (s *UserChatRoomService) FetchUserChatRooms(req *http.Request) ([]models.ChatRooms, error) {
	// Retrieve session values via the AuthInterface
	sessionValues, err := s.AuthRepo.GetSession(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Extract user ID from session values
	userID, ok := sessionValues["userID"].(uint)
	if !ok {
		return nil, fmt.Errorf("unauthorized: userID not found in session")
	}

	// Fetch chat rooms for the user from the repository
	return s.UserRepo.FetchUserChatRooms(userID)
}

func (s *UserChatRoomService) FetchUserChatRoomsByUserID(userID uint) ([]models.ChatRooms, error) {
	// Fetch chat rooms for the user from the repository
	return s.UserRepo.FetchUserChatRooms(userID)
}

func (s *UserChatRoomService) GetMessages(c *gin.Context) ([]models.Messages, error) {
	chatRoomID := c.Param("chatRoomID")
	userID := c.Query("userID")

	// Get session values
	sessionValues, err := s.AuthRepo.GetSession(c.Request)
	if err != nil {
		return nil, fmt.Errorf("session error: %w", err)
	}
fmt.Printf("session %v",sessionValues)
	// Validate user ID
	sessionUserID := fmt.Sprintf("%d", sessionValues["userID"])
	if sessionUserID != userID {
		log.Printf("Unauthorized access - session userID: %s does not match userID: %s", sessionUserID, userID)
		return nil, errors.New("unauthorized")
	}

	// Fetch messages from the repository
	return s.UserRepo.GetMessages(c.Request.Context(), userID, chatRoomID)
}

func (s *UserChatRoomService) DeleteMessage(c *gin.Context) (*DeleteMessageResponse, error) {
	messageIDStr := c.Param("messageID")
	chatRoomIDStr := c.Query("chat_room_id")
	userIDStr := c.Query("user_id")

	// Validate the request parameters
	if messageIDStr == "" || chatRoomIDStr == "" || userIDStr == "" {
		return nil, fmt.Errorf("missing required parameters")
	}

	// Convert messageID and chatRoomID from string to int
	messageID, err := strconv.Atoi(messageIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid messageID")
	}
	chatRoomID, err := strconv.Atoi(chatRoomIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid chatRoomID")
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid userID")
	}
	// Ensure user has permission to delete the message
	if !s.UserRepo.IsUserInChatRoom(uint(userID), uint(chatRoomID)) {
		return nil, errors.New("user not authorized")
	}

	// Delete the message
	if err := s.UserRepo.DeleteMessage(c.Request.Context(), uint(messageID), uint(chatRoomID)); err != nil {
		return nil, fmt.Errorf("failed to delete message: %w", err)
	}
	return &DeleteMessageResponse{
		MessageID:  uint(messageID),
		ChatRoomID: uint(chatRoomID),
		SenderID:   uint(userID),
	}, nil
}
