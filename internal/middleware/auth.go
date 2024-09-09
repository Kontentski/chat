package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"

)

func AuthMiddleware(store *sessions.CookieStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _ := store.Get(c.Request, "auth-session")
		user := session.Values["userID"]
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}
