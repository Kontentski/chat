package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/services"
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
	mockAuth := new(storage.MockUser)
	service := &services.UserChatRoomService{UserRepo: mockStorage, AuthRepo: mockAuth}

	router := gin.New()
	router.DELETE("/message/:messageID", DeleteMessageHandler(service))

	req, _ := http.NewRequest("DELETE", "/message/1?chat_room_id=2&user_id=3", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"message": "Message deleted successfully"}`, w.Body.String())
}

func TestDeleteMessage_MissingParams(t *testing.T) {
	mockStorage := new(storage.MockUser)
	mockAuth := new(storage.MockUser)
	service := &services.UserChatRoomService{UserRepo: mockStorage, AuthRepo: mockAuth}

	router := gin.New()
	router.DELETE("/message/:messageID", DeleteMessageHandler(service))
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
	mockAuth := new(storage.MockUser)
	service := &services.UserChatRoomService{UserRepo: mockStorage, AuthRepo: mockAuth}

	router := gin.New()
	router.DELETE("/message/:messageID", DeleteMessageHandler(service))
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
	mockAuth := new(storage.MockUser)
	service := &services.UserChatRoomService{UserRepo: mockStorage, AuthRepo: mockAuth}

	router := gin.New()
	router.DELETE("/message/:messageID", DeleteMessageHandler(service))
	req, _ := http.NewRequest("DELETE", "/message/1?chat_room_id=2&user_id=3", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"failed to delete message: deletion failed"}`, w.Body.String())
}

func TestGetMessages(t *testing.T) {
	// Initialize the mock objects
	mockAuth := new(storage.MockUser)
	mockRepo := new(storage.MockUser)
	service := &services.UserChatRoomService{
		UserRepo: mockRepo,
		AuthRepo: mockAuth,
	}

	// Create a test context
	gin.SetMode(gin.TestMode)

	t.Run("Success - Valid Session and Messages Fetched", func(t *testing.T) {
		// Arrange
		chatRoomID := "1"
		userID := "123"
		sessionValues := map[string]interface{}{
			"userID": 123,
		}
		messages := []models.Messages{
			{MessageID: 1, SenderID: 1, Content: "Hello", Timestamp: time.Now(), ChatRoomID: 1, IsDM: true},
		}

		// Mock dependencies
		mockAuth.On("GetSession", mock.Anything).Return(sessionValues, nil)
		mockRepo.On("GetMessages", mock.Anything, userID, chatRoomID).Return(messages, nil)

		// Create the test request and response recorder
		c, _ := createTestContextWithParams("chatRoomID", chatRoomID, "userID", userID)

		// Act
		actualMessages, err := service.GetMessages(c)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, messages, actualMessages)
		mockAuth.AssertExpectations(t)
		mockRepo.AssertExpectations(t)

		// Reset the mock after test run
		mockAuth = new(storage.MockUser)
		service.AuthRepo = mockAuth
	})

	t.Run("Unauthorized - UserID mismatch in session", func(t *testing.T) {
		// Arrange
		chatRoomID := "1"
		userID := "1232"
		sessionValues2 := map[string]interface{}{
			"userID": 999, // Different userID in session
		}

		// Mock dependencies
		mockAuth.On("GetSession", mock.Anything).Return(sessionValues2, nil)

		// Create the test request and response recorder
		c, _ := createTestContextWithParams("chatRoomID", chatRoomID, "userID", userID)

		// Act
		_, err := service.GetMessages(c)

		// Assert
		if err == nil {
			t.Error("An error is expected but got nil")
		} else {
			assert.Equal(t, "unauthorized", err.Error()) // Assuming this is the error message returned
		}
		mockAuth.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "GetMessages")
	})
}

// Helper function to create a test context
func createTestContextWithParams(paramKey, paramValue, queryKey, queryValue string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.URL.RawQuery = queryKey + "=" + queryValue
	c.Request = req
	c.Params = gin.Params{
		{Key: paramKey, Value: paramValue},
	}

	return c, w
}

func initTest()(*storage.MockUser, *storage.MockUser, *services.UserChatRoomService){
	mockAuth := new(storage.MockUser)
	mockRepo := new(storage.MockUser)
	service := &services.UserChatRoomService{
		UserRepo: mockRepo,
		AuthRepo: mockAuth,
	}

	return mockAuth, mockRepo, service
}
func TestFetchUserChatRooms(t *testing.T) {
    // Create a test context
    gin.SetMode(gin.TestMode)


    t.Run("Success - Valid Session and Chat Rooms Fetched", func(t *testing.T) {
		mockAuth, mockRepo, service := initTest()
		// Arrange
        userID := uint(123)
        sessionValues := map[string]interface{}{
            "userID": userID,
        }
        chatRooms := []models.ChatRooms{
            {ID: 1, Name: "General", Description: "General Chat", Type: "public"},
            {ID: 2, Name: "Tech Talk", Description: "Technology discussion", Type: "private"},
        }

        // Mock dependencies
        mockAuth.On("GetSession", mock.Anything).Return(sessionValues, nil)
        mockRepo.On("FetchUserChatRooms", userID).Return(chatRooms, nil)

        // Create the test request
        req, _ := http.NewRequest(http.MethodGet, "/", nil)

        // Act
        actualChatRooms, err := service.FetchUserChatRooms(req)

        // Assert
        assert.NoError(t, err)
        assert.Equal(t, chatRooms, actualChatRooms)
        mockAuth.AssertExpectations(t)
        mockRepo.AssertExpectations(t)
    })

    t.Run("Error - Invalid Session (userID not found)", func(t *testing.T) {
		mockAuth, mockRepo, service := initTest()

        // Arrange
        sessionValues := map[string]interface{}{
            // Missing userID or incorrect type
        }

        // Mock dependencies
        mockAuth.On("GetSession", mock.Anything).Return(sessionValues, nil)

        // Create the test request
        req, _ := http.NewRequest(http.MethodGet, "/", nil)

        // Act
        _, err := service.FetchUserChatRooms(req)

        // Assert
        if err == nil {
            t.Error("Expected error but got nil")
        } else {
            assert.Contains(t, err.Error(), "unauthorized")
        }
        mockAuth.AssertExpectations(t)
        mockRepo.AssertNotCalled(t, "FetchUserChatRooms")
    })

    t.Run("Error - Failed to Fetch Chat Rooms from Repo", func(t *testing.T) {
		mockAuth, mockRepo, service := initTest()
        // Arrange
        userID := uint(123)
        sessionValues := map[string]interface{}{
            "userID": userID,
        }
        expectedErr := fmt.Errorf("database error")

        // Mock dependencies
        mockAuth.On("GetSession", mock.Anything).Return(sessionValues, nil)
        mockRepo.On("FetchUserChatRooms", userID).Return(nil, expectedErr)

        // Create the test request
        req, _ := http.NewRequest(http.MethodGet, "/", nil)

        // Act
        _, err := service.FetchUserChatRooms(req)

        // Assert
        assert.Error(t, err)
        assert.EqualError(t, err, expectedErr.Error())

        // Verify expectations were met
        mockAuth.AssertExpectations(t)
        mockRepo.AssertExpectations(t)
    })
}
