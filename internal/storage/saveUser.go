package storage

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/kontentski/chat/internal/database"
	"github.com/kontentski/chat/internal/models"
)

func SaveUser(user *models.Users) (*models.Users, error) {
	log.Printf("Attempting to save user with Email: %s, Username: %s, Name: %s\n", user.Email, user.Username, user.Name)

	existingUser, err := GetUserByEmail(user.Email)
	if err != nil {
		log.Printf("Error when checking if user exists: %v\n", err)
		if err.Error() != "no rows in result set" {
			return nil, err
		}
	}

	if existingUser != nil {
		log.Printf("User already exists with ID: %d, updating last seen\n", existingUser.ID)
		user.ID = existingUser.ID
		err = UpdateLastSeen(user.ID)
		if err != nil {
			log.Printf("Error updating last seen: %v\n", err)
			return nil, err
		}

		// Return the existing user, including their password and username
		return existingUser, nil
	}

	log.Println("User does not exist, inserting a new user")

	query := `INSERT INTO users (name, email, username, password, profile_picture, last_seen, created_at) 
                VALUES ($1, $2, $3, $4, $5, $6, $7)
                RETURNING id, username, password`

	err = database.DB.QueryRow(context.Background(), query,
		user.Name,
		user.Email,
		"system_default", // Ensure default username
		"system_default", // Ensure default password
		user.ProfilePicture,
		time.Now(),
		time.Now(),
	).Scan(&user.ID, &user.Username, &user.Password) // Scan username and password

	if err != nil {
		log.Printf("Error saving user to the database: %v\n", err)
		return nil, err
	}

	log.Printf("User saved successfully with ID: %v, Username: %s, Password: %s", user.ID, user.Username, user.Password)
	return user, nil
}

// GetUserByEmail fetches a user by email from the database
func GetUserByEmail(email string) (*models.Users, error) {
	log.Printf("Fetching user by email: %s\n", email)

	var user models.Users
	query := `SELECT id, username, name, email, profile_picture, password, created_at 
                FROM users WHERE email = $1`

	err := database.DB.QueryRow(context.Background(), query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Name,
		&user.Email,
		&user.ProfilePicture,
		&user.Password, // Add password to the query
		&user.CreatedAt,
	)

	// Log the result of the query
	if err != nil {
		if err.Error() == "no rows in result set" {
			log.Printf("No user found with email: %s\n", email)
			return nil, errors.New("no rows in result set")
		}
		log.Printf("Error fetching user by email: %v\n", err)
		return nil, err
	}

	log.Printf("User found with ID: %d, Username: %s, Password: %s\n", user.ID, user.Username, user.Password)
	return &user, nil
}

// UpdateLastSeen updates the last seen timestamp for the user
func UpdateLastSeen(userID uint) error {
	log.Printf("Updating last seen for user ID: %d\n", userID)

	query := `UPDATE users SET last_seen = $1 WHERE id = $2`
	_, err := database.DB.Exec(context.Background(), query, time.Now(), userID)
	if err != nil {
		log.Printf("Error updating last seen for user ID %d: %v", userID, err)
		return err
	}

	log.Printf("Last seen updated for user ID: %d\n", userID)
	return nil
}
