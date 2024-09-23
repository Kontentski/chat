package auth

import (
	"encoding/gob"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/kontentski/chat/internal/models"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

const (
	MaxAge = 86400 * 30
)

var Store *sessions.CookieStore

func Init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file, continuing without it")
	}

	CookieSecret := os.Getenv("CookieSecret")
	googleClientId := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	callbackURL := os.Getenv("GOOGLE_CALLBACK_URL")

	gob.Register(models.Users{})

	Store = sessions.NewCookieStore([]byte(CookieSecret))
	Store.MaxAge(MaxAge)
	Store.Options.Path = "/"

	gothic.Store = Store

	goth.UseProviders(
		google.New(
			googleClientId,
			googleClientSecret,
			callbackURL,
			"profile",
			"email",
		),
	)
}
