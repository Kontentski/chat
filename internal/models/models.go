package models

import "time"

type Users struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	Name           string    `json:"name"`
	Password       string    `json:"password"`
	Email          string    `json:"email"`
	ProfilePicture string    `json:"profile_picture"`
	LastSeen       time.Time `json:"last_seen"`
	CreatedAt      time.Time `json:"created_at"`
}

type Messages struct {
	MessageID  uint      `json:"message_id"`
	SenderID   uint      `json:"sender_id"`
	Content    string    `json:"content"`
	Timestamp  time.Time `json:"timestamp"`
	ChatRoomID uint      `json:"chat_room_id"`
	IsDM       bool      `json:"is_dm"`
	ReadAt     string    `json:"read_at"`
	Sender     Users     `json:"sender"`
	Type       string    `json:"type,omitempty"`
}

type ChatRooms struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type ChatRoomMembers struct {
	ChatRoomID uint `json:"chat_room_id"`
	UserID     uint `json:"user_id"`
}

type ReadMessages struct {
	UserID     uint   `json:"user_id"`
	MessageID  uint   `json:"message_id"`
	ChatRoomID uint   `json:"chat_room_id"`
	ReadAt     string `json:"read_at"`
}
