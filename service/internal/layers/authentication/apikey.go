package authentication

import (
	"context"
	"net/http"
	"github.com/jamesread/japella/internal/db"
	"connectrpc.com/authn"
	log "github.com/sirupsen/logrus"
)

func CheckAuthApiKey(ctx context.Context, db *db.DB, req *http.Request) (*AuthenticatedUser, error) {
	token, ok := authn.BearerToken(req)

	if ok {
		log.Infof("Checking API key: %s", token)

		user := db.GetUserByApiKey(token)

		if user != nil {
			log.Infof("API key authenticated for user: %s", user.UserAccount.Username)
			return &AuthenticatedUser{Username: user.UserAccount.Username}, nil
		} else {
			log.Warnf("API key not found or invalid: %s", token)
			return nil, authn.Errorf("Invalid API key")
		}
	}

	return nil, nil
}
