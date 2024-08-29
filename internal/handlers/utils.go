package handlers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/storage"
)

func readMessages(conn *websocket.Conn) {
	defer func() {
		conn.Close()          // Ensure connection is closed properly
		delete(clients, conn) // Clean up the clients map
	}()

	for {
		// Create a generic map to handle both message and read receipt data
		var data map[string]interface{}
		if err := conn.ReadJSON(&data); err != nil {
			log.Printf("Error reading message: %v", err)
			return
		}

		// Log the received data for debugging
		log.Printf("Received data: %v", data)

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

		// Log the parsed message
		log.Printf("Received message: %+v", msg)

		// Ensure user is a member of the chat room
		if !isUserInChatRoom(clientData.userID, msg.ChatRoomID) {
			log.Printf("User %d is not a member of chat room %d", clientData.userID, msg.ChatRoomID)
			continue
		}

		if err := saveMessageToDB(&msg); err != nil {
			log.Printf("Error saving message to DB: %v", err)
			continue
		}

		// Log the broadcast
		log.Printf("Broadcasting message: %+v", msg)

		// Broadcast the message
		broadcast <- msg
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

	_, err := storage.DB.Exec(context.Background(), query, userID, messageID, chatRoomID)
	return err
}

func mapToStruct(data map[string]interface{}, target interface{}) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, target)
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

		if msg.Type == "delete" {
			// Handle deletion event
			for client, clientData := range clients {
				accessibleChatRooms, err := getUserChatRooms(clientData.userID)
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
						Type:  "delete",
					}
					if err := client.WriteJSON(deletionMsg); err != nil {
						log.Printf("Error broadcasting message: %v", err)
						client.Close()
						delete(clients, client)
					}
				}
			}
		} else {
			// Handle regular messages ( i think the next 8 rows not needed here)
			var sender models.Users
			err := storage.DB.QueryRow(context.Background(), "SELECT id, username, name FROM users WHERE id = $1", msg.SenderID).Scan(&sender.ID, &sender.Username, &sender.Name)
			if err != nil {
				log.Printf("Error fetching sender details: %v", err)
				continue
			}
			msg.Sender = sender

			for client, clientData := range clients {
				accessibleChatRooms, err := getUserChatRooms(clientData.userID)
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
	// Begin a transaction to ensure atomic operation
	log.Println("Starting transaction to save message")
	tx, err := storage.DB.Begin(context.Background())
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
	log.Printf("Retrieving the highest message ID for chat room %d", msg.ChatRoomID)
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

	// Insert the new message
	log.Printf("Inserting new message: %+v", *msg)
	insertQuery := `INSERT INTO messages (message_id, sender_id, content, chat_room_id, is_dm, timestamp) 
                    VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING timestamp`
	err = tx.QueryRow(context.Background(), insertQuery, msg.MessageID, msg.SenderID, msg.Content, msg.ChatRoomID, msg.IsDM).Scan(&msg.Timestamp)
	if err != nil {
		log.Printf("Error inserting message into DB: %v", err)
		return err
	}
	log.Printf("Message inserted successfully with timestamp: %v", msg.Timestamp)

	// Commit the transaction
	log.Println("Transaction committed successfully")
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
