package authentication

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	japauth "github.com/jamesread/httpauthshim"
	"github.com/jamesread/httpauthshim/authpublic"
	"github.com/jamesread/httpauthshim/sessions"
	"github.com/jamesread/japella/internal/db"
)

const (
	providerJapellaAPIKey  = "japella-api-key"
	providerJapellaSession = "japella-session"
)

// nopSessionPersistence satisfies httpauthshim session storage without persisting to disk.
// Japella keeps sessions in the database (see japella-session provider).
type nopSessionPersistence struct{}

func (nopSessionPersistence) Load(_, _ string, _ *sessions.SessionStorage) error {
	return nil
}

func (nopSessionPersistence) Save(_, _ string, _ *sessions.SessionStorage) error {
	return nil
}

func (nopSessionPersistence) RequiresFileLock() bool {
	return false
}

func newJapellaAuthShim(dbc *db.DB) (*japauth.AuthShimContext, error) {
	cfg := &authpublic.Config{
		BaseDir: filepath.Join(os.TempDir(), "japella-httpauthshim-unused"),
	}
	storage := sessions.NewSessionStorage(nopSessionPersistence{})
	ctx, err := japauth.NewAuthShimContext(cfg, storage)
	if err != nil {
		return nil, err
	}
	// Bearer API keys first, then cookie sessions.
	ctx.AddProvider(japellaAPIKeyBearerProvider(dbc))
	ctx.AddProvider(japellaSessionCookieProvider(dbc))
	return ctx, nil
}

// extractBearerToken returns the raw token from Authorization: Bearer <token>, or "".
func extractBearerToken(req *http.Request) string {
	if req == nil {
		return ""
	}
	h := req.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
}

// japellaAPIKeyBearerProvider validates Authorization Bearer tokens against DB API keys
// (httpauthshim provider).
func japellaAPIKeyBearerProvider(dbc *db.DB) func(*authpublic.AuthCheckingContext) *authpublic.AuthenticatedUser {
	return func(ac *authpublic.AuthCheckingContext) *authpublic.AuthenticatedUser {
		token := extractBearerToken(ac.Request)
		if token == "" {
			return nil
		}
		user := dbc.GetUserByApiKey(token)
		if user == nil {
			return nil
		}
		return &authpublic.AuthenticatedUser{
			Username: user.Username,
			Provider: providerJapellaAPIKey,
		}
	}
}

// japellaSessionCookieProvider resolves the japella-sid cookie via DB sessions (httpauthshim provider).
func japellaSessionCookieProvider(dbc *db.DB) func(*authpublic.AuthCheckingContext) *authpublic.AuthenticatedUser {
	return func(ac *authpublic.AuthCheckingContext) *authpublic.AuthenticatedUser {
		if ac.Request == nil {
			return nil
		}
		c, err := ac.Request.Cookie("japella-sid")
		if err != nil || c.Value == "" {
			return nil
		}
		sess := dbc.GetSessionByID(c.Value)
		if sess == nil || sess.UserAccount == nil {
			return nil
		}
		return &authpublic.AuthenticatedUser{
			Username: sess.UserAccount.Username,
			Provider: providerJapellaSession,
			SID:      c.Value,
		}
	}
}
