package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kontentski/chat/internal/auth"
	"github.com/kontentski/chat/internal/models"
	"github.com/kontentski/chat/internal/storage"
	"github.com/markbates/goth/gothic"
)

func AuthHandler(c *gin.Context) {
	provider := c.Query("provider")
	if provider == "" {
		c.String(http.StatusBadRequest, "You must select a provider")
		return
	}
	c.Set("provider", provider) // Set the provider in Gin context
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

// CallbackHandler handles the callback from the provider
func CallbackHandler(c *gin.Context) {
	userGoth, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		log.Println("Authentication failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user := &models.Users{
		Username:       userGoth.NickName,
		Name:           userGoth.Name,
		Email:          userGoth.Email,
		ProfilePicture: userGoth.AvatarURL,
		LastSeen:       time.Now(),
	}

	// Save or update the user in the database
	savedUser, err := storage.SaveUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user to the database"})
		return
	}

	// Create the session if the user has a proper password
	session, err := auth.Store.Get(c.Request, "auth-session")
	if err != nil {
		log.Println("Error getting session:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session error"})
		return
	}

	session.Values["userID"] = savedUser.ID
	session.Values["username"] = savedUser.Username
	if err := session.Save(c.Request, c.Writer); err != nil {
		log.Println("Error saving session:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session save error"})
		return
	}

	// Check if the saved user's password is "system_default"
	if savedUser.Password == "system_default" {
		// Redirect to /register if the user has the default password
		c.Redirect(http.StatusFound, "/auth/register")
		return
	}
	// Redirect to /home after successful login
	c.Redirect(http.StatusFound, "/homepage")
}


func LogoutHandler(c *gin.Context) {
	session, _ := auth.Store.Get(c.Request, "auth-session")
	session.Options.MaxAge = -1 
	session.Save(c.Request, c.Writer)
	c.Redirect(http.StatusFound, "/homepage")
}

