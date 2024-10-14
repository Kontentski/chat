package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/kontentski/chat/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockUser struct {
	CreateUserFn       func(user *models.Users) error
	IsUserInChatRoomFn func(userID, chatRoomID uint) bool
	DeleteMessageFn    func(ctx context.Context, messageID, chatRoomID uint) error
	mock.Mock
}

func (m *MockUser) CreateUser(user *models.Users) error {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(user)
	}
	return nil
}
func (m *MockUser) IsUserInChatRoom(userID, chatRoomID uint) bool {
	if m.IsUserInChatRoomFn != nil {
		return m.IsUserInChatRoomFn(userID, chatRoomID)
	}
	return false
}
func (m *MockUser) IsUserExists(username string) bool {
	args := m.Called(username)
	return args.Get(0).(bool)
}

func (m *MockUser) SearchUsers(ctx context.Context, query string) ([]models.Users, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]models.Users), args.Error(1)
}

func (m *MockUser) AddUserToTheChatRoom(ctx context.Context, userID string, chatRoomID uint) error {
	args := m.Called(ctx, userID, chatRoomID)
	return args.Error(0)
}

func (m *MockUser) DeleteUserFromChatRoom(ctx context.Context, userID, chatRoomID uint) error {
	args := m.Called(ctx, userID, chatRoomID)
	return args.Error(0)
}

func (m *MockUser) DeleteMessage(ctx context.Context, messageID, chatRoomID uint) error {
	if m.DeleteMessageFn != nil {
		return m.DeleteMessageFn(ctx, messageID, chatRoomID)
	}
	return nil
}

func (m *MockUser) GetMessages(ctx context.Context, userID uint, chatRoomID string) ([]models.Messages, error) {
	args := m.Called(ctx, userID, chatRoomID)
	return args.Get(0).([]models.Messages), args.Error(1)
}

func (m *MockUser) GetSession(req *http.Request) (map[string]interface{}, error) {
	args := m.Called(req)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockUser) FetchUserChatRooms(userID uint) ([]models.ChatRooms, error) {
	args := m.Called(userID)
	if chatRooms, ok := args.Get(0).([]models.ChatRooms); ok {
		return chatRooms, args.Error(1)
	}
	return nil, fmt.Errorf("database error")
}

func (m *MockUser) UploadFileToBucket(file multipart.File, originalFileName, filePath string, c context.Context) (string, error){
	if m.Called(file, originalFileName, filePath, c).Error(1)!= nil {
        return "", m.Called(file, originalFileName, filePath, c).Error(1)
    }
    return m.Called(file, originalFileName, filePath, c).String(0), nil
}

func (m *MockUser) GenerateSignedURL(filePath string) (string, error) {
	args := m.Called(filePath)
	return args.String(0), args.Error(1)
}


type MockTransaction struct {
	mock.Mock
}

func (m *MockTransaction) Exec(ctx context.Context, query string, args ...interface{}) (string, error) {
	argsList := m.Called(ctx, query, args)
	return argsList.String(0), argsList.Error(1)
}

func (m *MockTransaction) Commit(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockTransaction) Rollback(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}
