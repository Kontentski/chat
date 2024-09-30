package services

import (
	"errors"
	"fmt"
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

const UserIDKey = "userID"

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
	userID, ok := c.Get(UserIDKey)
	if !ok {
		return []models.Messages{}, nil
	}
	strchatroomID, err := strconv.Atoi(chatRoomID)
	if err != nil {
		return nil, fmt.Errorf("invalid chatRoomID")
	}
	IntuserID := userID.(uint)
	// Ensure user has permission to delete the message
	if !s.UserRepo.IsUserInChatRoom(IntuserID, uint(strchatroomID)) {
		return nil, errors.New("user not authorized")
	}


	return s.UserRepo.GetMessages(c.Request.Context(), IntuserID, chatRoomID)
}

func (s *UserChatRoomService) DeleteMessage(c *gin.Context) (*DeleteMessageResponse, error) {
	messageIDStr := c.Param("messageID")
	chatRoomIDStr := c.Query("chat_room_id")
	userID, ok := c.Get(UserIDKey)
	if !ok {
		return nil, nil
	}

	// Validate the request parameters
	if messageIDStr == "" || chatRoomIDStr == "" || userID == "" {
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
	IntuserID := userID.(uint)
	// Ensure user has permission to delete the message
	if !s.UserRepo.IsUserInChatRoom(IntuserID, uint(chatRoomID)) {
		return nil, errors.New("user not authorized")
	}

	// Delete the message
	if err := s.UserRepo.DeleteMessage(c.Request.Context(), uint(messageID), uint(chatRoomID)); err != nil {
		return nil, fmt.Errorf("failed to delete message: %w", err)
	}
	return &DeleteMessageResponse{
		MessageID:  uint(messageID),
		ChatRoomID: uint(chatRoomID),
		SenderID:   IntuserID,
	}, nil
}

func (s *UserChatRoomService) LeaveChatRoom(c *gin.Context) error {
	chatRoomIDStr := c.Param("chatRoomID")
	userID, ok := c.Get(UserIDKey)
	if !ok {
		return fmt.Errorf("no userID")
	}
	fmt.Printf("chatroooooooooooon %s\n\n\n", chatRoomIDStr)
	IntuserID := userID.(uint)
	chatRoomID, err := strconv.Atoi(chatRoomIDStr)
	if err != nil {
		return fmt.Errorf("invalid chatRoomID")
	}
	if !s.UserRepo.IsUserInChatRoom(IntuserID, uint(chatRoomID)) {
		return errors.New("user is not part of the chat room")
	}

	err = s.UserRepo.DeleteUserFromChatRoom(c, IntuserID, uint(chatRoomID))
	if err != nil {
		return errors.New("failed to leave the chat room")
	}
	return nil
}