package auth_test

import (
	"os"
	"testing"

	"github.com/kontentski/chat/internal/auth"
	"github.com/markbates/goth/gothic"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	// Mock environment variables
	os.Setenv("CookieSecret", "test-secret")
	os.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
	os.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret")
	os.Setenv("GOOGLE_CALLBACK_URL", "http://localhost:8080/auth/google/callback")

	// Call Init
	auth.Init()

	// Check if Store is initialized
	assert.NotNil(t, auth.Store, "Cookie store should be initialized")
	assert.Equal(t, auth.MaxAge, auth.Store.Options.MaxAge, "MaxAge should be set to expected value")
	assert.Equal(t, "/", auth.Store.Options.Path, "Cookie Path should be '/'")

	// Check if Gothic store is set
	assert.Equal(t, auth.Store, gothic.Store, "Gothic Store should be the same as auth.Store")

	// Clean up
	os.Unsetenv("CookieSecret")
	os.Unsetenv("GOOGLE_CLIENT_ID")
	os.Unsetenv("GOOGLE_CLIENT_SECRET")
	os.Unsetenv("GOOGLE_CALLBACK_URL")
}
