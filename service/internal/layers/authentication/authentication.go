package authentication

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/jamesread/golure/pkg/redact"
	"github.com/jamesread/httpauthshim/authpublic"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/debuglog"

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
		debuglog.Authf("Checking API key: %s", redact.RedactString(token))
		user := al.DB.GetUserByApiKey(token)
		if user != nil {
			debuglog.Authf("API key authenticated for user: %s", user.Username)
			au := &AuthenticatedUser{User: user}
			return al.finishWithRBAC(au, procedureName)
		}
		// Not a Japella API key — fall through to httpauthshim (e.g. JWT Bearer).
		debuglog.Authf("Bearer token is not a Japella API key; trying httpauthshim providers")
	}

	shimUser, err := al.shim.AuthFromHttpReqWithError(req)
	if err != nil {
		debuglog.Authf("httpauthshim: %v", err)
		return nil, authn.Errorf("Authentication Required")
	}

	if shimUser.IsGuest() {
		if allowList[procedureName] {
			if req.Method != http.MethodPost {
				debuglog.Authf("Allowing unauthenticated access to %s", procedureName)
			}
			return nil, nil
		}
		return nil, authn.Errorf("Authentication Required")
	}

	dbUser, err := al.resolveDBUser(shimUser)
	if err != nil || dbUser == nil {
		log.Warnf("Session user %q not found in database", shimUser.Username)
		return nil, authn.Errorf("Authentication Required")
	}

	debuglog.Authf("Authenticated via %s as %q", shimUser.Provider, shimUser.Username)
	au := &AuthenticatedUser{User: dbUser}
	return al.finishWithRBAC(au, procedureName)
}

// resolveDBUser maps an httpauthshim identity to a Japella user_accounts row.
// JWT users are auto-provisioned when missing (created_by = sso-autocreated).
func (al *AuthLayer) resolveDBUser(shimUser *authpublic.AuthenticatedUser) (*db.UserAccount, error) {
	if shimUser == nil || shimUser.Username == "" {
		return nil, nil
	}

	dbUser := al.DB.GetUserByUsername(shimUser.Username)
	if dbUser != nil {
		return dbUser, nil
	}

	if shimUser.Provider != providerJWT {
		return nil, nil
	}

	created, err := al.DB.CreateUserAccount(shimUser.Username, "", db.UserCreatedBySSO)
	if err != nil {
		// Concurrent first login: another request may have created the row.
		if existing := al.DB.GetUserByUsername(shimUser.Username); existing != nil {
			return existing, nil
		}
		return nil, err
	}

	log.Infof("Auto-created user %q from JWT SSO (id %d)", created.Username, created.ID)

	if err := al.DB.EnsureUserInEveryoneGroup(created.ID); err != nil {
		log.Warnf("Could not add SSO user %d to Everyone group: %v", created.ID, err)
	}

	if reloaded := al.DB.GetUserByID(created.ID); reloaded != nil {
		return reloaded, nil
	}
	return created, nil
}

func (al *AuthLayer) WrapHandler(in http.Handler) http.Handler {
	authMiddleware := authn.NewMiddleware(al.Handle)
	return authMiddleware.Wrap(in)
}

// WrapMCPHandler requires a Bearer API key authenticated via httpauthshim and attaches
// the Japella AuthenticatedUser to the request context for MCP tool handlers.
func (al *AuthLayer) WrapMCPHandler(in http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		au, status, err := al.authenticateMCP(r)
		if err != nil {
			writeMCPAuthError(w, status, err.Error())
			return
		}
		ctx := authn.SetInfo(r.Context(), au)
		in.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeMCPAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (al *AuthLayer) authenticateMCP(r *http.Request) (*AuthenticatedUser, int, error) {
	if al.devNoAuth {
		return &AuthenticatedUser{
			User: &db.UserAccount{Username: "anonymous"},
			RBAC: &db.EffectiveRBAC{IsSuperuser: true, Permissions: map[string]bool{}},
		}, http.StatusOK, nil
	}

	if extractBearerToken(r) == "" {
		return nil, http.StatusUnauthorized, errMCPAuthRequired
	}

	shimUser, err := al.shim.AuthFromHttpReqWithError(r)
	if err != nil {
		debuglog.Authf("httpauthshim (mcp): %v", err)
		return nil, http.StatusUnauthorized, errMCPInvalidAPIKey
	}
	if shimUser.IsGuest() || shimUser.Provider != providerJapellaAPIKey {
		return nil, http.StatusUnauthorized, errMCPInvalidAPIKey
	}

	dbUser := al.DB.GetUserByUsername(shimUser.Username)
	if dbUser == nil {
		return nil, http.StatusUnauthorized, errMCPInvalidAPIKey
	}

	au := &AuthenticatedUser{User: dbUser}
	info, rbacErr := al.finishWithRBAC(au, "")
	if rbacErr != nil {
		return nil, http.StatusForbidden, errMCPForbidden
	}
	return info.(*AuthenticatedUser), http.StatusOK, nil
}

var (
	errMCPAuthRequired  = &mcpAuthError{msg: "Authorization required: Bearer API key"}
	errMCPInvalidAPIKey = &mcpAuthError{msg: "Invalid API key"}
	errMCPForbidden     = &mcpAuthError{msg: "Forbidden"}
)

type mcpAuthError struct{ msg string }

func (e *mcpAuthError) Error() string { return e.msg }

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
