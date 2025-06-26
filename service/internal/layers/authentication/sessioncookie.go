package authentication

import (
	"context"
	"github.com/jamesread/japella/internal/db"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func CheckAuthSessionCookie(ctx context.Context, db *db.DB, req *http.Request) (*AuthenticatedUser, error) {
	cookie, err := req.Cookie("japella-sid")

	log.Debugf("Checking session cookie: %v", cookie)

	if err != nil {
		return nil, nil // No session cookie found
	}

	if cookie.Value == "" {
		return nil, nil
	}

	sessionID := cookie.Value
	session := db.GetSessionByID(sessionID)

	if session == nil {
		return nil, nil // No session found for the given ID
	}

	return &AuthenticatedUser{User: session.UserAccount}, nil
}
