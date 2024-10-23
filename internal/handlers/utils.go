package handlers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kontentski/chat/internal/database"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/services"
	"github.com/kontentski/chat/internal/storage"
)

func handleConnection(conn *websocket.Conn) {
	log.Println("Setting up connection for client")

	conn.SetPongHandler(func(string) error {
		log.Println("Received pong from client, resetting read deadline")
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()

		for range ticker.C {
			log.Println("Sending ping to client")
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error sending ping: %v", err)
				return
			}
		}
	}()

	log.Println("Connection setup complete")
}

func readMessages(conn *websocket.Conn, messageStorage storage.UserRepository) {
	defer func() {
		storage.UpdateLastSeen(clients[conn].userID)
		conn.Close()
		delete(clients, conn)
	}()

	for {
		// Create a generic map to handle both message and read receipt data
		var data map[string]interface{}
		if err := conn.ReadJSON(&data); err != nil {
			log.Printf("Error reading message: %v", err)
			return
		}

		// Log the received data for debugging
		log.Printf("Received data: %+v", data)

		clientData, ok := clients[conn]
		if !ok {
			log.Println("Client data not found")
			continue
		}

		// Check if the data contains a read receipt
		if messageID, ok := data["message_id"].(float64); ok {
			chatRoomID := uint(data["chat_room_id"].(float64))
			log.Printf("Processing read receipt: message_id=%v, chat_room_id=%v", messageID, chatRoomID)

			err := markMessageAsRead(clientData.userID, uint(messageID), chatRoomID)
			if err != nil {
				log.Printf("Error marking message as read: %v", err)
			}
			continue // Skip further processing as it's a read receipt
		}

		// If it's a regular message, continue with the existing processing
		var msg models.Messages
		if err := mapToStruct(data, &msg); err != nil {
			log.Printf("Error mapping data to message struct: %v", err)
			continue
		}

		log.Printf("Received message")
		log.Printf("Received message data: %+v", msg)

		chatRoomID := uint(data["chat_room_id"].(float64))
		log.Printf("chat_room_id received message: %v", chatRoomID)

		// Handle media type messages
		if msg.Type == "media" {
			content, ok := data["content"].(string)
			if !ok {
				log.Printf("Error: content field missing for media message")
				log.Printf("Full data received for media message: %+v", data)
				continue
			}
			msg.Content = content
			log.Printf("Media message content: %s", msg.Content)
		}

		if !messageStorage.IsUserInChatRoom(clientData.userID, chatRoomID) {
			log.Printf("User %d is not a member of chat room %d", clientData.userID, msg.ChatRoomID)
			continue
		}

		log.Printf("Saving message:")
		if err := saveMessageToDB(&msg); err != nil {
			log.Printf("Error saving message to DB: %v", err)
			continue
		}

		// Log the broadcast
		log.Printf("Broadcasting message")

		// Broadcast the message
		Broadcast <- msg
	}
}

func markMessageAsRead(userID uint, messageID uint, chatRoomID uint) error {
	query := `
        INSERT INTO read_messages (user_id, message_id, chat_room_id, read_at) 
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (user_id, message_id, chat_room_id) 
        DO UPDATE SET read_at = CASE 
            WHEN read_messages.read_at IS NULL THEN NOW() 
            ELSE read_messages.read_at 
        END
    `

	_, err := database.DB.Exec(context.Background(), query, userID, messageID, chatRoomID)
	return err
}

func mapToStruct(data map[string]interface{}, target interface{}) error {
	log.Printf("Raw incoming data: %+v", data)

	encoded, err := json.Marshal(data)
	log.Printf("Raw incoming data: %+v", encoded)

	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, target)
}

func handleMessages(service *services.UserChatRoomService) {
	for msg := range Broadcast {

		if msg.Type == "delete" {
			log.Println("Handling delete message")
			// Handle deletion event
			for client, clientData := range clients {
				accessibleChatRooms, err := service.FetchUserChatRoomsByUserID(clientData.userID)
				if err != nil {
					log.Printf("Error fetching chat rooms for user %d: %v", clientData.userID, err)
					continue
				}

				chatRoomIDs := getChatRoomIDs(accessibleChatRooms)
				if contains(chatRoomIDs, msg.ChatRoomID) {
					// Send deletion event to the client
					deletionMsg := models.Messages{
						MessageID:  msg.MessageID,
						ChatRoomID: msg.ChatRoomID,
						SenderID:   msg.SenderID,
						Type:       "delete",
					}
					if err := client.WriteJSON(deletionMsg); err != nil {
						log.Printf("Error broadcasting message: %v", err)
						client.Close()
						delete(clients, client)
					}
				}
			}
		} else if msg.Type == "media" {
			log.Println("Handling media message")
			// Generate signed URL for media content
			signedURL, err := service.GenerateSignedURL(msg.Content)
			if err != nil {
				log.Printf("Error generating signed URL: %v", err)
				continue
			}
			msg.Content = signedURL
		}

		log.Println("Handling a message")

		// Handle regular messages ( i think the next 8 rows not needed here)
		var sender models.Users
		err := database.DB.QueryRow(context.Background(), "SELECT id, username, name FROM users WHERE id = $1", msg.SenderID).Scan(&sender.ID, &sender.Username, &sender.Name)
		if err != nil {
			log.Printf("Error fetching sender details: %v", err)
			continue
		}
		msg.Sender = sender

		for client, clientData := range clients {
			accessibleChatRooms, err := service.FetchUserChatRoomsByUserID(clientData.userID)
			if err != nil {
				log.Printf("Error fetching chat rooms for user %d: %v", clientData.userID, err)
				continue
			}

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
func saveMessageToDB(msg *models.Messages) error {
	log.Println("Starting transaction to save message")
	tx, err := database.DB.Begin(context.Background())
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return err
	}
	defer func() {
		if err != nil {
			log.Println("Rolling back transaction due to error")
			tx.Rollback(context.Background())
		} else {
			log.Println("Committing transaction")
			tx.Commit(context.Background())
		}
	}()

	// Retrieve the highest message ID for the chat room
	var lastMessageID uint
	query := `
		SELECT COALESCE(MAX(message_id), 0)
		FROM messages
		WHERE chat_room_id = $1
	`
	err = tx.QueryRow(context.Background(), query, msg.ChatRoomID).Scan(&lastMessageID)
	if err != nil {
		log.Printf("Error retrieving last message ID: %v", err)
		return err
	}
	log.Printf("Last message ID for chat room %d: %d", msg.ChatRoomID, lastMessageID)

	// Increment the message ID
	msg.MessageID = lastMessageID + 1
	log.Printf("New message ID: %d", msg.MessageID)

	insertQuery := `INSERT INTO messages (message_id, sender_id, content, chat_room_id, is_dm, timestamp, type) 
                    VALUES ($1, $2, $3, $4, $5, NOW(), $6) RETURNING timestamp`
	err = tx.QueryRow(context.Background(), insertQuery, msg.MessageID, msg.SenderID, msg.Content, msg.ChatRoomID, msg.IsDM, msg.Type).Scan(&msg.Timestamp)
	if err != nil {
		log.Printf("Error inserting message into DB: %v", err)
		return err
	}
	return nil
}
