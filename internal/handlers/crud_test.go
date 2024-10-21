package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

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

	// Create a UserChatRoomService with the mockStorage
	service := &services.UserChatRoomService{
		UserRepo: mockStorage,
		AuthRepo: mockStorage, // You might want to create a separate mock for AuthRepo if needed
	}

	router := gin.New()
	router.POST("/users", CreateUser(service))

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

	// Create a UserChatRoomService with the mockStorage
	service := &services.UserChatRoomService{
		UserRepo: mockStorage,
		AuthRepo: mockStorage, // You might want to create a separate mock for AuthRepo if needed
	}

	router := gin.New()
	router.POST("/users", CreateUser(service))

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

// ... existing code ...

func TestDeleteMessageHandler(t *testing.T) {
	// Initialize the Broadcast channel
	Broadcast = make(chan models.Messages, 100)

	t.Run("Success", func(t *testing.T) {
		mockStorage := &storage.MockUser{
			IsUserInChatRoomFn: func(userID, chatRoomID uint) bool {
				return true // Simulate user is authorized
			},
			DeleteMessageFn: func(ctx context.Context, messageID, chatRoomID uint) error {
				return nil // Simulate successful deletion
			},
		}
		mockAuth := &storage.MockUser{}
		service := &services.UserChatRoomService{
			UserRepo: mockStorage,
			AuthRepo: mockAuth,
		}

		router := gin.New()
		router.DELETE("/message/:messageID", DeleteMessageHandler(service))

		req, _ := http.NewRequest("DELETE", "/message/1?chat_room_id=2", nil)
		w := httptest.NewRecorder()

		// Set userID in the context
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "messageID", Value: "1"}}
		c.Request = req
		c.Set("userID", uint(3))

		// Start a goroutine to consume messages from the Broadcast channel
		go func() {
			<-Broadcast // Consume the message
		}()

		DeleteMessageHandler(service)(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"message": "Message deleted successfully"}`, w.Body.String())
	})

	t.Run("Failure", func(t *testing.T) {
		mockStorage := &storage.MockUser{
			IsUserInChatRoomFn: func(userID, chatRoomID uint) bool {
				return true // Simulate user is authorized
			},
			DeleteMessageFn: func(ctx context.Context, messageID, chatRoomID uint) error {
				return errors.New("deletion failed") // Simulate failure
			},
		}
		mockAuth := &storage.MockUser{}
		service := &services.UserChatRoomService{
			UserRepo: mockStorage,
			AuthRepo: mockAuth,
		}

		router := gin.New()
		router.DELETE("/message/:messageID", DeleteMessageHandler(service))

		req, _ := http.NewRequest("DELETE", "/message/1?chat_room_id=2", nil)
		w := httptest.NewRecorder()

		// Set userID in the context
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "messageID", Value: "1"}}
		c.Request = req
		c.Set("userID", uint(3))

		DeleteMessageHandler(service)(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.JSONEq(t, `{"error":"Something went wrong"}`, w.Body.String())
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockStorage := &storage.MockUser{
			IsUserInChatRoomFn: func(userID, chatRoomID uint) bool {
				return false // Simulate user is not authorized
			},
		}
		mockAuth := &storage.MockUser{}
		service := &services.UserChatRoomService{
			UserRepo: mockStorage,
			AuthRepo: mockAuth,
		}

		router := gin.New()
		router.DELETE("/message/:messageID", DeleteMessageHandler(service))

		req, _ := http.NewRequest("DELETE", "/message/1?chat_room_id=2", nil)
		w := httptest.NewRecorder()

		// Set userID in the context
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "messageID", Value: "1"}}
		c.Request = req
		c.Set("userID", uint(3))

		DeleteMessageHandler(service)(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.JSONEq(t, `{"error":"Something went wrong"}`, w.Body.String())
	})

	t.Run("Missing Parameters", func(t *testing.T) {
		mockStorage := &storage.MockUser{}
		mockAuth := &storage.MockUser{}
		service := &services.UserChatRoomService{
			UserRepo: mockStorage,
			AuthRepo: mockAuth,
		}

		router := gin.New()
		router.DELETE("/message/:messageID", DeleteMessageHandler(service))

		req, _ := http.NewRequest("DELETE", "/message/1", nil) // Missing chat_room_id
		w := httptest.NewRecorder()

		// Set userID in the context
		c, _ := gin.CreateTestContext(w)
		c.Set("userID", uint(3))
		c.Request = req

		DeleteMessageHandler(service)(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.JSONEq(t, `{"error":"Missing required parameters"}`, w.Body.String())
	})
}

func TestGetMessagesHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockAuth, mockRepo, service := initTest()
		log.Printf("mockAuth: %v\n", mockAuth)

		// Arrange
		userID := uint(123)
		chatRoomID := "456"
		messages := []models.Messages{
			{MessageID: 1, SenderID: 123, Content: "Hello", ChatRoomID: 456, Type: "text"},
			{MessageID: 2, SenderID: 456, Content: "Hi there", ChatRoomID: 456, Type: "text"},
		}

		// Mock dependencies
		mockRepo.On("IsUserInChatRoom", userID, uint(456)).Return(true)
		mockRepo.On("GetMessages", mock.Anything, userID, chatRoomID).Return(messages, nil)

		// Create a test context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "chatRoomID", Value: chatRoomID}}
		c.Set(services.UserIDKey, userID)

		// Create a new request and set it to the context
		req, _ := http.NewRequest("GET", "/messages/"+chatRoomID, nil)
		c.Request = req

		// Ensure that the UserRepo is set in the service
		service.UserRepo = mockRepo

		log.Println("TestGetMessagesHandler/Success: Calling handler")
		// Act
		GetMessagesHandler(*service)(c)

		// Assert
		log.Printf("TestGetMessagesHandler/Success: Response status: %d", w.Code)
		log.Printf("TestGetMessagesHandler/Success: Response body: %s", w.Body.String())

		assert.Equal(t, http.StatusOK, w.Code)
		var response []models.Messages
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			log.Printf("TestGetMessagesHandler/Success: Error unmarshaling response: %v", err)
		}
		assert.NoError(t, err)
		assert.Equal(t, messages, response)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockAuth, mockRepo, service := initTest()
		log.Printf("mockAuth: %v\n", mockAuth)

		// Arrange
		chatRoomID := "456"

		// Create a test context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "chatRoomID", Value: chatRoomID}}

		// Create a new request and set it to the context
		req, _ := http.NewRequest("GET", "/messages/"+chatRoomID, nil)
		c.Request = req

		// Act
		GetMessagesHandler(*service)(c)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		var response []models.Messages
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Empty(t, response)
		mockRepo.AssertNotCalled(t, "GetMessages")
	})

	t.Run("Error - Failed to Get Messages", func(t *testing.T) {
		mockAuth, mockRepo, service := initTest()
		log.Printf("mockAuth: %v\n", mockAuth)

		// Arrange
		userID := uint(123)
		chatRoomID := "456"
		expectedErr := errors.New("database error")

		// Mock dependencies
		mockRepo.On("IsUserInChatRoom", userID, uint(456)).Return(true)
		mockRepo.On("GetMessages", mock.Anything, userID, chatRoomID).Return(nil, expectedErr)

		// Create a test context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "chatRoomID", Value: chatRoomID}}
		c.Set(services.UserIDKey, userID)

		// Create a new request and set it to the context
		req, _ := http.NewRequest("GET", "/messages/"+chatRoomID, nil)
		c.Request = req

		// Ensure that the UserRepo is set in the service
		service.UserRepo = mockRepo

		// Act
		GetMessagesHandler(*service)(c)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var response gin.H
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, gin.H{"error": expectedErr.Error()}, response)
		mockRepo.AssertExpectations(t)
	})
}

func initTest() (*storage.MockUser, *storage.MockUser, *services.UserChatRoomService) {
	mockAuth := new(storage.MockUser)
	mockRepo := new(storage.MockUser)
	mediaStorage := new(storage.MockBucketStorage)
	service := &services.UserChatRoomService{
		UserRepo:     mockRepo,
		AuthRepo:     mockAuth,
		MediaStorage: mediaStorage,
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
