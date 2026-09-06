package iam

import (
	"context"
	"os"

	"github.com/goccy/go-yaml"
	armlayer "github.com/jamesread/armature-iam/layer"
	"github.com/jamesread/armature-iam/store"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
)

const (
	CookieName = "japella-sid"
)

// NewStore wraps the application database as an armature-iam store.Store.
func NewStore(dbc *db.DB) store.Store {
	return NewDBStore(dbc)
}

// NewAuthLayer wires armature-iam Connect auth for Japella.
func NewAuthLayer(st store.Store, requiredPermission func(string) string) (*AuthLayer, error) {
	cfg := armlayer.Config{
		CookieName: CookieName,
		AuthYAML:   authYAMLFromRuntime(),
		Logger:     log.StandardLogger(),
		AllowUnauthenticated: []string{
			controlv1.JapellaControlApiServiceLoginWithUsernameAndPasswordProcedure,
			controlv1.JapellaControlApiServiceGetStatusProcedure,
		},
		RequiredPermission: requiredPermission,
		DevDisableAuth:     os.Getenv("JAPELLA_DEV_DISABLE_AUTH") == "true",
		SecureCookies:      os.Getenv("JAPELLA_SECURE_COOKIES") != "false",
	}
	inner, err := armlayer.New(st, cfg)
	if err != nil {
		return nil, err
	}
	return &AuthLayer{Layer: inner, Store: st}, nil
}

// AuthLayer wraps armature-iam/layer.Layer for Japella-specific helpers.
type AuthLayer struct {
	*armlayer.Layer
	Store store.Store
}

func authYAMLFromRuntime() map[string]any {
	rc := runtimeconfig.Get()
	if rc == nil || rc.Auth == nil {
		return nil
	}
	data, err := yaml.Marshal(rc.Auth)
	if err != nil {
		log.Warnf("armature-iam: marshal auth config: %v", err)
		return nil
	}
	var out map[string]any
	if err := yaml.Unmarshal(data, &out); err != nil {
		log.Warnf("armature-iam: unmarshal auth config: %v", err)
		return nil
	}
	return out
}

// EnsureBootstrap runs RBAC bootstrap via the IAM store (called from db connection chain).
func EnsureBootstrap(st store.Store) error {
	return st.EnsureRBACBootstrap(context.Background())
}
