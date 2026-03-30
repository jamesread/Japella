package authentication

import (
	"os"
	"path/filepath"

	japauth "github.com/jamesread/httpauthshim"
	"github.com/jamesread/httpauthshim/authpublic"
	"github.com/jamesread/httpauthshim/sessions"
	"github.com/jamesread/japella/internal/db"
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
	ctx.AddProvider(japellaSessionCookieProvider(dbc))
	return ctx, nil
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
			Provider: "japella-session",
			SID:      c.Value,
		}
	}
}
