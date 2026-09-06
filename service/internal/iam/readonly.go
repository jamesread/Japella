package iam

import (
	"net/http"
	"strings"

	"connectrpc.com/authn"
	armlayer "github.com/jamesread/armature-iam/layer"
)

// ReadOnlyAllowedProcedure reports whether a read-only API key may call a Connect RPC.
// Read-only keys may invoke Get* and List* procedures only.
func ReadOnlyAllowedProcedure(procedure string) bool {
	if procedure == "" {
		return true
	}
	method := procedure
	if i := strings.LastIndex(procedure, "/"); i >= 0 {
		method = procedure[i+1:]
	}
	return strings.HasPrefix(method, "Get") || strings.HasPrefix(method, "List")
}

// ReadOnlyAllowsMCPTool reports whether a read-only API key may invoke an MCP tool.
func ReadOnlyAllowsMCPTool(toolName string) bool {
	switch toolName {
	case "japella_list_social_accounts", "japella_list_pending_approvals":
		return true
	default:
		return false
	}
}

func (l *AuthLayer) readOnlyGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if au := armlayer.UserFromContext(r.Context()); au != nil && au.ReadOnly {
			procedure, _ := authn.InferProcedure(r.URL)
			if !ReadOnlyAllowedProcedure(procedure) {
				http.Error(w, "Forbidden: read-only API key", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// WrapHandler applies armature-iam auth and blocks write RPCs for read-only API keys.
func (l *AuthLayer) WrapHandler(in http.Handler) http.Handler {
	return l.Layer.WrapHandler(l.readOnlyGuard(in))
}

// WrapMCPHandler applies Bearer API key auth for MCP (write tools enforce read-only in handlers).
func (l *AuthLayer) WrapMCPHandler(in http.Handler) http.Handler {
	return l.Layer.WrapMCPHandler(in)
}
