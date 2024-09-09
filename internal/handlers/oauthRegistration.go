package handlers

import (
	"context"
	"log"
	"net/http"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/kontentski/chat/internal/auth"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(c *gin.Context) {
	c.File("./homepage/register/register.html")
}

func RegisterPostHandler(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	session, err := auth.Store.Get(c.Request, "auth-session")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session error"})
		return
	}

	// Get the user ID from the session
	userID, ok := session.Values["userID"].(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Validate username and password
	if !isValidPassword(password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be 8 characters long and contain at least one uppercase letter and one digit"})
		return
	}

	exists, err := UsernameExists(username)
	if err != nil || exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already taken"})
		return
	}
	if username == "system_default" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reserved username"})
		return
	}
	// Hash password before saving
	hashedPassword := hashPassword(password)

	// Prepare new user with username and hashed password
	user := &models.Users{
		Username: username,
		Password: hashedPassword,
		ID:       userID,
	}

	// Save the new user and handle both return values
	err = UpdateUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Log the saved user's details for verification
	log.Printf("User saved successfully: ID=%d, Username=%s", user.ID, user.Username)

	// Redirect to home or wherever needed after registration
	c.Redirect(http.StatusFound, "/homepage/")
}

func UsernameExists(username string) (bool, error) {
	var count int
	err := storage.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE username = $1", username).Scan(&count)
	return count > 0, err
}

func isValidPassword(password string) bool {
	var hasUpper, hasDigit bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}
	return hasUpper && hasDigit && len(password) >= 8
}

func hashPassword(password string) string {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	return string(hashedPassword)
}

// In storage package

func UpdateUser(user *models.Users) error {
	query := `UPDATE users SET username = $1, password = $2 WHERE id = $3`
	_, err := storage.DB.Exec(context.Background(), query, user.Username, user.Password, user.ID)
	return err
}
