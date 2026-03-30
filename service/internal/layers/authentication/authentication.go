package authentication

import (
	"context"
	"net/http"
	"os"

	"github.com/jamesread/golure/pkg/redact"
	"github.com/jamesread/japella/internal/db"

	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"
	japauth "github.com/jamesread/httpauthshim"

	"connectrpc.com/authn"
	log "github.com/sirupsen/logrus"
)

type AuthLayer struct {
	DB        *db.DB
	shim      *japauth.AuthShimContext
	devNoAuth bool
}

var allowList = map[string]bool{
	controlv1.JapellaControlApiServiceLoginWithUsernameAndPasswordProcedure: true,
	controlv1.JapellaControlApiServiceGetStatusProcedure:                    true,
}

func (al *AuthLayer) finishWithRBAC(au *AuthenticatedUser, procedureName string) (any, error) {
	rb, err := al.DB.LoadEffectiveRBAC(au.User.ID)
	if err != nil {
		log.Errorf("LoadEffectiveRBAC: %v", err)
		return nil, authn.Errorf("Authentication Required")
	}
	au.RBAC = rb

	if allowList[procedureName] {
		return au, nil
	}

	req := RequiredPermission(procedureName)
	if req != "" && !au.HasPermission(req) {
		log.Warnf("RBAC denied user %q procedure %q needs %q", au.User.Username, procedureName, req)
		return nil, authn.Errorf("Forbidden")
	}
	return au, nil
}

func (al *AuthLayer) Handle(ctx context.Context, req *http.Request) (any, error) {
	procedureName, _ := authn.InferProcedure(req.URL)

	if al.devNoAuth {
		au := &AuthenticatedUser{
			User: &db.UserAccount{Username: "anonymous"},
			RBAC: &db.EffectiveRBAC{IsSuperuser: true, Permissions: map[string]bool{}},
		}
		if allowList[procedureName] {
			return au, nil
		}
		req := RequiredPermission(procedureName)
		if req != "" && !au.HasPermission(req) {
			return nil, authn.Errorf("Forbidden")
		}
		return au, nil
	}

	if token, ok := authn.BearerToken(req); ok {
		log.Infof("Checking API key: %s", redact.RedactString(token))
		user := al.DB.GetUserByApiKey(token)
		if user == nil {
			log.Warnf("API key not found or invalid: %s", redact.RedactString(token))
			return nil, authn.Errorf("Invalid API key")
		}
		log.Infof("API key authenticated for user: %s", user.Username)
		au := &AuthenticatedUser{User: user}
		return al.finishWithRBAC(au, procedureName)
	}

	shimUser, err := al.shim.AuthFromHttpReqWithError(req)
	if err != nil {
		log.Debugf("httpauthshim: %v", err)
		return nil, authn.Errorf("Authentication Required")
	}

	if shimUser.IsGuest() {
		if allowList[procedureName] {
			if req.Method != http.MethodPost {
				log.Debugf("Allowing unauthenticated access to %s", procedureName)
			}
			return nil, nil
		}
		return nil, authn.Errorf("Authentication Required")
	}

	dbUser := al.DB.GetUserByUsername(shimUser.Username)
	if dbUser == nil {
		log.Warnf("Session user %q not found in database", shimUser.Username)
		return nil, authn.Errorf("Authentication Required")
	}

	au := &AuthenticatedUser{User: dbUser}
	return al.finishWithRBAC(au, procedureName)
}

func (al *AuthLayer) WrapHandler(in http.Handler) http.Handler {
	authMiddleware := authn.NewMiddleware(al.Handle)
	return authMiddleware.Wrap(in)
}

// DefaultAuthLayer wires Connect-RPC auth to github.com/jamesread/httpauthshim for cookie sessions,
// with Bearer API keys checked against the database before the shim runs.
func DefaultAuthLayer(db *db.DB) *AuthLayer {
	if os.Getenv("JAPELLA_DEV_DISABLE_AUTH") == "true" {
		log.Warn("JAPELLA_DEV_DISABLE_AUTH is set: all API requests run as anonymous user")
		return &AuthLayer{DB: db, devNoAuth: true}
	}

	shim, err := newJapellaAuthShim(db)
	if err != nil {
		log.Fatalf("Failed to initialize httpauthshim: %v", err)
	}

	return &AuthLayer{DB: db, shim: shim}
}
