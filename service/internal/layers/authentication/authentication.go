package authentication

import (
	"context"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"

	"github.com/jamesread/japella/internal/db"

	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"

	"connectrpc.com/authn"
)

type AuthenticatedUser struct {
	User *db.UserAccount
}

type AuthLayer struct {
	DB        *db.DB
	AuthChain []AuthFunc
}

var allowList = map[string]bool{
	controlv1.JapellaControlApiServiceLoginWithUsernameAndPasswordProcedure: true,
	controlv1.JapellaControlApiServiceGetStatusProcedure:                    true,
}

type AuthFunc func(ctx context.Context, db *db.DB, req *http.Request) (*AuthenticatedUser, error)

func (al *AuthLayer) Handle(ctx context.Context, req *http.Request) (any, error) {
	procedureName, _ := authn.InferProcedure(req.URL)

	var user *AuthenticatedUser
	var err error

	for _, authFunc := range al.AuthChain {
		user, err = authFunc(ctx, al.DB, req)

		if err == nil && user != nil {
			return user, nil
		}
	}

	if user == nil && allowList[procedureName] {
		// We just log the unauthenticated access for allowed procedures as a helper
		// for debugging and development really. Filtering out GET and OPTIONS logs
		// as these are often used by browsers and other clients to check the API status.

		if req.Method != http.MethodPost {
			log.Debugf("Allowing unauthenticated access to %s", procedureName)
		}

		return nil, nil
	}

	return nil, authn.Errorf("Authentication Required")
}

func (al *AuthLayer) WrapHandler(in http.Handler) http.Handler {
	authMiddleware := authn.NewMiddleware(al.Handle)
	authHandler := authMiddleware.Wrap(in)

	return authHandler
}

func CheckAuthAllowAll(ctx context.Context, dbc *db.DB, req *http.Request) (*AuthenticatedUser, error) {
	return &AuthenticatedUser{User: &db.UserAccount{Username: "anonymous"}}, nil
}

func DefaultAuthLayer(db *db.DB) *AuthLayer {
	authChain := []AuthFunc{
		CheckAuthSessionCookie,
		CheckAuthApiKey,
		CheckAuthTrustedHeader,
	}

	if os.Getenv("JAPELLA_DEV_DISABLE_AUTH") == "true" {
		authChain = []AuthFunc{
			CheckAuthAllowAll,
		}
	}

	return &AuthLayer{
		DB:        db,
		AuthChain: authChain,
	}
}
