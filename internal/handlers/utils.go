package handlers

import (
	"context"
	"log"

	"github.com/gorilla/websocket"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/storage"
)

func readMessages(conn *websocket.Conn) {
	defer func() {
		log.Println("log defer")
		conn.Close()          // Ensure connection is closed properly
		delete(clients, conn) // Clean up the clients map
	}()

	for {
		log.Println("log read for")
		var msg models.Messages
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("Error reading message: %v", err)
			return
		}
		log.Println("log client ")

		clientData, ok := clients[conn]
		if !ok {
			log.Println("Client data not found")
			continue
		}
		log.Println("client chat check")
		// Check if the user is a member of the chat room
		if !isUserInChatRoom(clientData.userID, msg.ChatRoomID) {
			log.Printf("User %d is not a member of chat room %d", clientData.userID, msg.ChatRoomID)
			continue
		}

		log.Printf("Clientdata: %v", clientData)
		log.Println("log save message err")
		if err := saveMessageToDB(msg); err != nil {
			log.Printf("Error saving message to DB: %v", err)
			continue
		}
		log.Println("log broadcast")

		broadcast <- msg
		log.Println("log after broadcast")

	}
}

func isUserInChatRoom(userID uint, chatRoomID uint) bool {
	log.Println("query chat room")
	query := `
	SELECT COUNT(*) 
	FROM chat_room_members 
	WHERE user_id = $1 AND chat_room_id = $2
	`
	var count int
	err := storage.DB.QueryRow(context.Background(), query, userID, chatRoomID).Scan(&count)
	if err != nil {
		log.Printf("Error checking user access: %v", err)
		return false
	}
	return count > 0
}


func handleMessages() {
	for msg := range broadcast {
		log.Println("Handling broadcast message")

		// Fetch the sender's details from the database
		var sender models.Users
		err := storage.DB.QueryRow(context.Background(), "SELECT id, username, name FROM users WHERE id = $1", msg.SenderID).Scan(&sender.ID, &sender.Username, &sender.Name)
		if err != nil {
			log.Printf("Error fetching sender details: %v", err)
			continue
		}
		msg.Sender = sender

		// Fetch the chat rooms the user has access to
		for client, clientData := range clients {
			accessibleChatRooms, err := getUserChatRooms(clientData.userID)
			if err != nil {
				log.Printf("Error fetching chat rooms for user %d: %v", clientData.userID, err)
				continue
			}

			// Check if the chat room is accessible to the user
			chatRoomIDs := getChatRoomIDs(accessibleChatRooms)
			if contains(chatRoomIDs, msg.ChatRoomID) {
				if err := client.WriteJSON(msg); err != nil {
					log.Printf("Error broadcasting message: %v", err)
					client.Close()
					delete(clients, client)
				}
			}
		}
	}
}

// Utility function to check if a slice contains a value
func contains(slice []uint, value uint) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func getChatRoomIDs(chatRooms []models.ChatRooms) []uint {
	var ids []uint
	for _, room := range chatRooms {
		ids = append(ids, room.ID)
	}
	return ids
}

// saveMessageToDB saves a message to the database with an incremented message ID unique to the chat room.
func saveMessageToDB(msg models.Messages) error {
	// Begin a transaction to ensure atomic operation
	tx, err := storage.DB.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	// Retrieve the highest message ID for the chat room
	var lastMessageID uint
	query := `
		SELECT COALESCE(MAX(message_id), 0)
		FROM messages
		WHERE chat_room_id = $1
	`
	err = tx.QueryRow(context.Background(), query, msg.ChatRoomID).Scan(&lastMessageID)
	if err != nil {
		return err
	}
	// Increment the message ID
	msg.MessageID = lastMessageID + 1
	// Insert the new message
	insertQuery := `INSERT INTO messages (message_id, sender_id, content, chat_room_id, is_dm, timestamp) 
                    VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING timestamp`
	err = tx.QueryRow(context.Background(), insertQuery, msg.MessageID, msg.SenderID, msg.Content, msg.ChatRoomID, msg.IsDM).Scan(&msg.Timestamp)
	if err != nil {
		return err
	}
	// Commit the transaction
	if err = tx.Commit(context.Background()); err != nil {
		return err
	}
	return nil
}

// getUserChatRooms retrieves the chat rooms the user is part of
func getUserChatRooms(userID uint) ([]models.ChatRooms, error) {
	query := `
	SELECT cr.id, cr.name, cr.description, cr.type
	FROM chat_rooms cr
	JOIN chat_room_members crm ON cr.id = crm.chat_room_id
	WHERE crm.user_id = $1
`

	rows, err := storage.DB.Query(context.Background(), query, userID)
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
