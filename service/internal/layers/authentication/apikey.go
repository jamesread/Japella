package authentication

import (
	"connectrpc.com/authn"
	"context"
	"github.com/jamesread/golure/pkg/redact"
	"github.com/jamesread/japella/internal/db"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func CheckAuthApiKey(ctx context.Context, db *db.DB, req *http.Request) (*AuthenticatedUser, error) {
	token, ok := authn.BearerToken(req)

	if ok {
		log.Infof("Checking API key: %s", redact.RedactString(token))

		user := db.GetUserByApiKey(token)

		if user != nil {
			log.Infof("API key authenticated for user: %s", user.Username)
			return &AuthenticatedUser{User: user}, nil
		} else {
			log.Warnf("API key not found or invalid: %s", redact.RedactString(token))
			return nil, authn.Errorf("Invalid API key")
		}
	}

	return nil, nil
}
