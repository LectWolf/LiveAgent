package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/liveagent/agent-gateway/internal/auth/agenttoken"
)

func HTTPMiddleware(expectedToken string, tokens *agenttoken.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := bearerToken(r.Header.Get("Authorization"))
		var principal Principal
		var ok bool
		if isAdminAPI(r.URL.Path) {
			if !ValidateToken(raw, expectedToken) {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			principal, ok = Principal{Admin: true}, true
		} else {
			principal, ok = ResolveClientAccessToken(raw, expectedToken, tokens)
		}
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
	})
}

func isAdminAPI(path string) bool {
	return path == "/api/agents" || strings.HasPrefix(path, "/api/agents/")
}

func bearerToken(headerValue string) string {
	headerValue = strings.TrimSpace(headerValue)
	if headerValue == "" {
		return ""
	}
	parts := strings.SplitN(headerValue, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func ValidateBearerHeader(headerValue, expectedToken string) bool {
	return ValidateToken(bearerToken(headerValue), expectedToken)
}

func ValidateToken(value, expectedToken string) bool {
	value = strings.TrimSpace(value)
	expectedToken = strings.TrimSpace(expectedToken)
	if value == "" || expectedToken == "" {
		return false
	}
	valueHash := sha256.Sum256([]byte(value))
	expectedHash := sha256.Sum256([]byte(expectedToken))
	return subtle.ConstantTimeCompare(valueHash[:], expectedHash[:]) == 1
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": message,
	})
}
