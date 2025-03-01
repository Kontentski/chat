package services

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/storage"
)

type ChatRoomService interface {
	FetchUserChatRooms(req *http.Request) ([]models.ChatRooms, error)
	FetchUserChatRoomsByUserID(userID uint) ([]models.ChatRooms, error)
	GetMessages(c *gin.Context) ([]models.Messages, error)
	DeleteMessage(c *gin.Context) (*DeleteMessageResponse, error)
	LeaveChatRoom(c *gin.Context) error
	SearchUsers(c *gin.Context) (*[]UsersListResponse, error)
	AddUserToChatRoom(c *gin.Context) error
	UploadMedia(c *gin.Context) (string, error)
	GenerateSignedURL(filePath string) (string, error)
	CreateUser(user *models.Users) error
}

type UserChatRoomServiceImpl struct {
    UserRepo     storage.UserRepository
    AuthRepo     storage.AuthRepository
    MediaStorage storage.BucketStorage
}

type DeleteMessageResponse struct {
	MessageID  uint
	ChatRoomID uint
	SenderID   uint
}
type UsersListResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

const UserIDKey = "userID"


func NewUserChatRoomService(userRepo storage.UserRepository, authRepo storage.AuthRepository, mediaStorage storage.BucketStorage) ChatRoomService {
	return &UserChatRoomServiceImpl{
		UserRepo:     userRepo,
		AuthRepo:     authRepo,
		MediaStorage: mediaStorage,
	}
}

// FetchUserChatRooms retrieves the user's chat rooms by processing the session.
func (s *UserChatRoomServiceImpl) FetchUserChatRooms(req *http.Request) ([]models.ChatRooms, error) {
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

	return s.UserRepo.FetchUserChatRooms(userID)
}

func (s *UserChatRoomServiceImpl) FetchUserChatRoomsByUserID(userID uint) ([]models.ChatRooms, error) {
	// Fetch chat rooms for the user from the repository
	return s.UserRepo.FetchUserChatRooms(userID)
}

func (s *UserChatRoomServiceImpl) GetMessages(c *gin.Context) ([]models.Messages, error) {
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

	messages, err := s.UserRepo.GetMessages(c.Request.Context(), IntuserID, chatRoomID)
	if err != nil {
		return nil, err
	}

	// Process media messages
	for i, msg := range messages {
		if msg.Type == "media" {
			signedURL, err := s.MediaStorage.GenerateSignedURL(msg.Content)
			if err != nil {
				log.Printf("Error generating signed URL for message %d: %v", msg.MessageID, err)
				continue
			}
			messages[i].Content = signedURL
		}
	}

	return messages, nil
}

func (s *UserChatRoomServiceImpl) DeleteMessage(c *gin.Context) (*DeleteMessageResponse, error) {
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

	if err := s.UserRepo.DeleteMessage(c.Request.Context(), uint(messageID), uint(chatRoomID)); err != nil {
		return nil, fmt.Errorf("failed to delete message: %w", err)
	}
	return &DeleteMessageResponse{
		MessageID:  uint(messageID),
		ChatRoomID: uint(chatRoomID),
		SenderID:   IntuserID,
	}, nil
}

func (s *UserChatRoomServiceImpl) LeaveChatRoom(c *gin.Context) error {
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

func (s *UserChatRoomServiceImpl) SearchUsers(c *gin.Context) (*[]UsersListResponse, error) {
	query := c.Query("q")

	if query == "" {
		return nil, fmt.Errorf("missing search query")
	}

	users, err := s.UserRepo.SearchUsers(c, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	var usersListResponse []UsersListResponse
	for _, user := range users {
		usersListResponse = append(usersListResponse, UsersListResponse{
			UserID:   fmt.Sprint(user.ID),
			Username: user.Username,
			Name:     user.Name,
		})
	}
	return &usersListResponse, nil
}

func (s *UserChatRoomServiceImpl) AddUserToChatRoom(c *gin.Context) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	fmt.Printf("Raw request body: %s\n", string(body))
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	var input struct {
		UserID     string `json:"user_id" binding:"required"`
		ChatRoomID uint   `json:"chat_room_id" binding:"required"`
	}
	fmt.Printf("input waluesn4 %v\n\n:", input)

	if err := c.ShouldBindJSON(&input); err != nil {
		fmt.Printf("input walues: %v\n\n\n", input)
		return err
	}

	err = s.UserRepo.AddUserToTheChatRoom(c, input.UserID, input.ChatRoomID)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserChatRoomServiceImpl) UploadMedia(c *gin.Context) (string, error) {
	chatRoomID := c.PostForm("chat_room_id")
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return "", fmt.Errorf("failed to get file: %v", err)
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		return "", fmt.Errorf("no file extension found for filename: %s", header.Filename)
	}

	generatedFileName := fmt.Sprintf("%d%s", time.Now().Unix(), ext)
	//filepath == chatrooms/1/09238450934.jpg
	filePath := fmt.Sprintf("chatrooms/%s/%s", chatRoomID, generatedFileName)

	filePath, err = s.MediaStorage.UploadFileToBucket(file, header.Filename, filePath, c.Request.Context())
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %v", err)
	}

	return filePath, nil
}

func (s *UserChatRoomServiceImpl) GenerateSignedURL(filePath string) (string, error) {
	return s.MediaStorage.GenerateSignedURL(filePath)
}

func (s *UserChatRoomServiceImpl) CreateUser(user *models.Users) error {
	return s.UserRepo.CreateUser(user)
}
