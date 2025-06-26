package authentication

import (
	"context"
	"github.com/jamesread/japella/internal/db"
	"net/http"
)

func CheckAuthTrustedHeader(ctx context.Context, db *db.DB, req *http.Request) (*AuthenticatedUser, error) {
	return nil, nil
}
