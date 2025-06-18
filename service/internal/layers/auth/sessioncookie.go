package auth

import (
	"context"
	"net/http"
	"github.com/jamesread/japella/internal/db"
	log "github.com/sirupsen/logrus"
)

func CheckAuthSessionCookie(ctx context.Context, db *db.DB, req *http.Request) (*AuthenticatedUser, error) {
	cookie, err := req.Cookie("japella-sid")

	log.Infof("Checking session cookie: %v", cookie)

	if err != nil {
		return nil, nil // No session cookie found
	}

	sessionID := cookie.Value
	session := db.GetSessionByID(sessionID)

	return &AuthenticatedUser{Username: session.UserAccount.Username}, nil
}
