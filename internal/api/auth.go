package api

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const agentIDKey contextKey = "agent_id"

// NewAuthMiddleware creates authentication middleware using a configured credential map.
// credentials maps API key -> stable Conductor actor ID.
// Credentials authenticate identity; credentials are not themselves identity.
func NewAuthMiddleware(credentials map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := extractAPIKey(r)
			if apiKey == "" {
				http.Error(w, "unauthorized: missing API key", http.StatusUnauthorized)
				return
			}

			actorID, ok := credentials[apiKey]
			if !ok {
				http.Error(w, "unauthorized: invalid API key", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), agentIDKey, actorID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractAPIKey extracts the API key from X-API-Key header or Authorization: Bearer header.
func extractAPIKey(r *http.Request) string {
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// agentIDFromContext extracts the authenticated actor ID from request context.
func agentIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(agentIDKey).(string); ok {
		return id
	}
	return "unknown"
}
