package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kontentski/chat/internal/database"
	"github.com/kontentski/chat/internal/models"
)

type UserStorage interface {
	CreateUser(user *models.Users) error
	IsUserInChatRoom(userID, chatRoomID uint) bool
	DeleteMessage(ctx context.Context, messageID, chatRoomID uint) error
	GetMessages(ctx context.Context ,userID string, chatRoomID string) ([]models.Messages, error)
}

type AuthInterface interface {
	GetSession(req *http.Request) (map[string]interface{}, error)
}

type UserQuery struct {
	DB *pgxpool.Pool
}

func (r *UserQuery) CreateUser(user *models.Users) error {
	err := r.DB.QueryRow(context.Background(), CreateUserQuery, user.Username, user.Name, user.Password, user.Email, user.ProfilePicture).Scan(&user.ID)

	return err
}

func (r *UserQuery) IsUserInChatRoom(userID, chatRoomID uint) bool {

	log.Println("query chat room")

	var count int
	err := database.DB.QueryRow(context.Background(), IsUserInChatRoomQuery, userID, chatRoomID).Scan(&count)
	if err != nil {
		log.Printf("Error checking user access: %v", err)
		return false
	}
	return count > 0

}

func (r *UserQuery) DeleteMessage(ctx context.Context, messageID, chatRoomID uint) error {
	// Begin a transaction to ensure atomic operation
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error: Failed to start transaction %w ", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	_, err = tx.Exec(ctx, DeleteMessageQuery, messageID, chatRoomID)

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("error: Failed to commit transaction %w ", err)
	}
	return nil
}

func (r *UserQuery) GetMessages(ctx context.Context, userID string, chatRoomID string) ([]models.Messages, error) {
	// Log the start of the query
	log.Printf("GetMessages: Fetching messages for userID: %s in chatRoomID: %s", userID, chatRoomID)

	// Execute the query
	rows, err := r.DB.Query(ctx, GetMessagesQuery, userID, chatRoomID)
	if err != nil {
		log.Printf("GetMessages: Query error - %v", err)
		return nil, err
	}
	defer rows.Close()


	var messages []models.Messages
	for rows.Next() {
		var msg models.Messages
		var readAt sql.NullTime

		// Scan the row into the message struct
		if err := rows.Scan(&msg.MessageID, &msg.SenderID, &msg.Sender.Username, &msg.Sender.Name, &msg.Content, &msg.Timestamp, &msg.ChatRoomID, &msg.IsDM, &readAt); err != nil {
			log.Printf("GetMessages: Error scanning row - %v", err)
			return nil, err
		}

		// Handle readAt field
		if readAt.Valid {
			msg.ReadAt = readAt.Time.Format(time.RFC3339)
		} else {
			msg.ReadAt = "1970-01-01T00:00:00Z"
		}

		// Log each message
		log.Printf("GetMessages: Retrieved message - ID: %d, SenderID: %d, Content: %s, Timestamp: %s, ReadAt: %s", 
			msg.MessageID, msg.SenderID, msg.Content, msg.Timestamp.Format(time.RFC3339), msg.ReadAt)

		messages = append(messages, msg)
	}

	// Check for errors after iteration
	if err := rows.Err(); err != nil {
		log.Printf("GetMessages: Rows error - %v", err)
		return nil, err
	}

	// Log the number of messages retrieved
	log.Printf("GetMessages: Total messages retrieved - %d", len(messages))

	return messages, nil
}

type RealAuth struct {
	Store *sessions.CookieStore
}

func (r *RealAuth) GetSession(req *http.Request) (map[string]interface{}, error) {
	// Get the session from the store
	session, err := r.Store.Get(req, "auth-session")
	if err != nil {
		return nil, err
	}
	sessionValues := make(map[string]interface{})
	for k, v := range session.Values {
		if keyStr, ok := k.(string); ok {
			sessionValues[keyStr] = v
		} else {
			return nil, fmt.Errorf("invalid session key type: %T", k)
		}
	}

	return sessionValues, nil
}
