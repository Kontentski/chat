package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateUser_Success(t *testing.T) {
	mockStorage := &storage.MockUser{
		CreateUserFn: func(user *models.Users) error {
			user.ID = 123
			return nil
		},
	}

	router := gin.New()
	router.POST("/users", CreateUser(mockStorage))

	reqBody := `{"username":"testuser","name":"Test User","password":"password123","email":"testuser@example.com"}`
	req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	expectedResponse := `{"message":"User created successfully","user_id":123}`
	assert.JSONEq(t, expectedResponse, w.Body.String(), "Expected response %v but got %v", expectedResponse, w.Body.String())
}

func TestCreateUser_Failure(t *testing.T) {
	mockStorage := &storage.MockUser{
		CreateUserFn: func(user *models.Users) error {
			return fmt.Errorf("database error")
		},
	}

	router := gin.New()
	router.POST("/users", CreateUser(mockStorage))

	reqBody := `{"username":"testuser","name":"Test User","password":"password123","email":"testuser@example.com"}`
	req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Record the response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	expectedResponse := `{"error":"database error"}`
	assert.JSONEq(t, expectedResponse, w.Body.String(), "Expected response %v but got %v", expectedResponse, w.Body.String())
}

func TestDeleteMessage_Success(t *testing.T) {
	mockStorage := &storage.MockUser{
		IsUserInChatRoomFn: func(userID, chatRoomID uint) bool {
			return true // Simulate user is authorized
		},
		DeleteMessageFn: func(ctx context.Context, messageID, chatRoomID uint) error {
			return nil // Simulate successful deletion
		},
	}

	router := gin.New()
	router.DELETE("/message/:messageID", DeleteMessage(mockStorage))

	req, _ := http.NewRequest("DELETE", "/message/1?chat_room_id=2&user_id=3", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"message": "Message deleted successfully"}`, w.Body.String())
}

func TestDeleteMessage_MissingParams(t *testing.T) {
	mockStorage := &storage.MockUser{}

	router := gin.New()
	router.DELETE("/message/:messageID", DeleteMessage(mockStorage))
	req, _ := http.NewRequest("DELETE", "/message/1?chat_room_id=&user_id=", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error": "Missing required parameters"}`, w.Body.String())
}

func TestDeleteMessage_Unauthorized(t *testing.T) {
	mockStorage := &storage.MockUser{
		IsUserInChatRoomFn: func(userID, chatRoomID uint) bool {
			return false // Simulate user is not authorized
		},
	}

	router := gin.New()
	router.DELETE("/message/:messageID", DeleteMessage(mockStorage))
	req, _ := http.NewRequest("DELETE", "/message/1?chat_room_id=2&user_id=3", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.JSONEq(t, `{"error": "User not authorized to delete message"}`, w.Body.String())
}

func TestDeleteMessage_Failure(t *testing.T) {
	mockStorage := &storage.MockUser{
		IsUserInChatRoomFn: func(userID, chatRoomID uint) bool {
			return true // Simulate user is authorized
		},
		DeleteMessageFn: func(ctx context.Context, messageID, chatRoomID uint) error {
			return errors.New("deletion failed") // Simulate failure
		},
	}

	router := gin.New()
	router.DELETE("/message/:messageID", DeleteMessage(mockStorage))
	req, _ := http.NewRequest("DELETE", "/message/1?chat_room_id=2&user_id=3", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error": "Failed to delete message"}`, w.Body.String())
}

func TestGetMessages(t *testing.T) {
    // Create a new Gin engine and route for testing
    r := gin.Default()
    mockStorage := new(storage.MockUser)

    // Define the mock behavior
    mockStorage.On("GetSession", mock.Anything).Return(map[string]interface{}{"userID": "user1"}, nil)
    mockStorage.On("GetMessages", "user1", "chatroom1").Return([]models.Messages{
        {
            MessageID:  1,
            SenderID:   1,
            Sender:     models.Users{Username: "testuser", Name: "Test User"},
            Content:    "Hello, World!",
            ChatRoomID: 1,
            IsDM:       false,
            ReadAt:     "1970-01-01T00:00:00Z",
        },
    }, nil)

    // Use the GetMessages function with the mock
    r.GET("/messages/:chatRoomID", GetMessages(mockStorage, mockStorage))

    // Create a request to test the route
    req, err := http.NewRequest(http.MethodGet, "/messages/chatroom1?userID=user1", nil)
    if err != nil {
        t.Fatalf("Could not create request: %v", err)
    }

    // Record the response
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    // Assert the status code
    assert.Equal(t, http.StatusOK, w.Code)

    // Assert the response body
    expectedResponse := `[{"chat_room_id":1, "content":"Hello, World!", "is_dm":false, "message_id":1, "read_at":"1970-01-01T00:00:00Z", "sender":{"created_at":"0001-01-01T00:00:00Z", "email":"", "id":0, "last_seen":"0001-01-01T00:00:00Z", "name":"Test User", "password":"", "profile_picture":"", "username":"testuser"}, "sender_id":1, "timestamp":"0001-01-01T00:00:00Z", "type":""}]`
    assert.JSONEq(t, expectedResponse, w.Body.String())

    // Assert that the expected methods were called
    mockStorage.AssertExpectations(t)
}
