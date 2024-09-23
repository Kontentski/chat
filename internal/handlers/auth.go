package handlers
/* 
import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// GenerateToken generates a JWT token for the user
func GenerateToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("secret-key"))
}

// ParseToken extracts userID from the Authorization token
func ParseToken(c *gin.Context) (uint, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return 0, errors.New("authorization header missing")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret-key"), nil
	})
	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, errors.New("invalid token")
	}

	userID, ok := claims["userID"].(float64)
	if !ok {
		return 0, errors.New("userID not found in token")
	}

	return uint(userID), nil
}

// GetChatRoomsHandler handles the request to fetch chat rooms for a user
func GetChatRoomsHandler(c *gin.Context) {
	userID, err := ParseToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	chatRooms, err := getUserChatRooms(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch chat rooms"})
		return
	}

	c.JSON(http.StatusOK, chatRooms)
}
 */