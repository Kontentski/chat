package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"mime/multipart"
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
	IsUserExists(username string) bool
	AddUserToTheChatRoom(ctx context.Context, userID string, chatRoomID uint) error
	SearchUsers(ctx context.Context, q string) ([]models.Users, error)
	DeleteUserFromChatRoom(ctx context.Context, IntuserID, chatRoomID uint) error
	DeleteMessage(ctx context.Context, messageID, chatRoomID uint) error
	GetMessages(ctx context.Context, userID uint, chatRoomID string) ([]models.Messages, error)
	FetchUserChatRooms(userID uint) ([]models.ChatRooms, error)
	UploadFileToBucket(file multipart.File, originalFileName, filePath string, c context.Context) (string, error)
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

func (r *UserQuery) SearchUsers(ctx context.Context, q string) ([]models.Users, error) {
	rows, err := r.DB.Query(ctx, SearchUsersQuery, "%"+q+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.Users
	for rows.Next() {
		var user models.Users
		if err := rows.Scan(&user.ID, &user.Username, &user.Name); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
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

func (r *UserQuery) IsUserExists(username string) bool {
	var count int
	err := database.DB.QueryRow(context.Background(), IsUserExistsQuery, "%"+username+"%").Scan(&count)
	if err != nil {
		log.Printf("Failed to find user with username %s: %v", username, err)
		return false
	}
	return count > 0
}

func (r *UserQuery) AddUserToTheChatRoom(ctx context.Context, userID string, chatRoomID uint) error {
	_, err := database.DB.Exec(ctx, AddUserToTheChatRoomQuery, userID, chatRoomID)
	return err
}

func (r *UserQuery) DeleteUserFromChatRoom(ctx context.Context, userID, chatRoomID uint) error {
	_, err := r.DB.Exec(ctx, DeleteUserFromChatRoomQuery, userID, chatRoomID)
	return err
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

func (r *UserQuery) GetMessages(ctx context.Context, userID uint, chatRoomID string) ([]models.Messages, error) {
	// Log the start of the query
	log.Printf("GetMessages: Fetching messages for userID: %d in chatRoomID: %s", userID, chatRoomID)

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
		var msgType sql.NullString  // Use sql.NullString for the type field

		// Scan the row into the message struct
		if err := rows.Scan(
			&msg.MessageID, 
			&msg.SenderID, 
			&msg.Sender.Username, 
			&msg.Sender.Name, 
			&msg.Content, 
			&msg.Timestamp, 
			&msg.ChatRoomID, 
			&msg.IsDM, 
			&msgType,  // Scan into msgType (sql.NullString)
			&readAt,
		); err != nil {
			log.Printf("GetMessages: Error scanning row - %v", err)
			return nil, err
		}

		// Handle readAt field
		if readAt.Valid {
			msg.ReadAt = readAt.Time.Format(time.RFC3339)
		} else {
			msg.ReadAt = "1970-01-01T00:00:00Z"
		}

		// Handle msgType field
		if msgType.Valid {
			msg.Type = msgType.String
		} else {
			msg.Type = ""  // or any default value you prefer
		}

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

func (r *UserQuery) FetchUserChatRooms(userID uint) ([]models.ChatRooms, error) {
	rows, err := r.DB.Query(context.Background(), FetchUserChatRoomsQuery, userID)
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
