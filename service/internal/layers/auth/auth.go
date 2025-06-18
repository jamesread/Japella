package auth

import (
	"context"
	"net/http"
	log "github.com/sirupsen/logrus"
	"os"

	"github.com/jamesread/japella/internal/db"

	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"

	"connectrpc.com/authn"
)

type AuthenticatedUser struct {
	Username string
}

type AuthLayer struct {
	DB *db.DB
	AuthChain []AuthFunc
}

var allowList = map[string]bool{
	controlv1.JapellaControlApiServiceLoginWithUsernameAndPasswordProcedure: true,
	controlv1.JapellaControlApiServiceGetStatusProcedure: true,
}

type AuthFunc func(ctx context.Context, db *db.DB, req *http.Request) (*AuthenticatedUser, error)

func (al *AuthLayer) Handle(ctx context.Context, req *http.Request) (any, error) {
	log.Infof("Handling auth request: %s %s", req.Method, req.URL.Path)

	procedureName, _ := authn.InferProcedure(req.URL)

	if allowList[procedureName] {
		log.Infof("Allowing unauthenticated access to %s", procedureName)
		return nil, nil
	} else {
		for _, authFunc := range al.AuthChain {
			user, err := authFunc(ctx, al.DB, req)

			if err == nil && user != nil {
				return user, nil
			}
		}
	}

	return nil, authn.Errorf("Authentication Required")
}

func (al *AuthLayer) WrapHandler(in http.Handler) http.Handler {
	authMiddleware := authn.NewMiddleware(al.Handle)
	authHandler := authMiddleware.Wrap(in)

	return authHandler
}

func CheckAuthAllowAll(ctx context.Context, db *db.DB, req *http.Request) (*AuthenticatedUser, error) {
	return &AuthenticatedUser{Username: "anonymous"}, nil
}

func DefaultAuthLayer(db *db.DB) *AuthLayer {
	authChain := []AuthFunc{
		CheckAuthSessionCookie,
		CheckAuthApiKey,
		CheckAuthTrustedHeader,
	}

	if os.Getenv("JAPELLA_DISABLE_AUTH") == "true" {
		authChain = []AuthFunc{
			CheckAuthAllowAll,
		}
	}

	return &AuthLayer{
		DB: db,
		AuthChain: authChain,
	}
}
