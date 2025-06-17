package auth

import (
	"context"
	"net/http"
	"github.com/jamesread/japella/internal/db"
)

func CheckAuthTrustedHeader(ctx context.Context, db *db.DB, req *http.Request) (*AuthenticatedUser, error) {
	return nil, nil
}
