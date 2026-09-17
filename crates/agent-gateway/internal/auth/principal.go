package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/liveagent/agent-gateway/internal/auth/agenttoken"
)

type principalContextKey struct{}

// Principal 是已通过 HTTP/WS 鉴权的调用方。
// Admin 使用网关共享 Token；否则 AgentID 是手机用电脑标识登录后锁定的那一台。
type Principal struct {
	Admin   bool
	AgentID string
}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func FromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}

func AllowsAgent(principal Principal, agentID string) bool {
	if principal.Admin {
		return true
	}
	return principal.AgentID != "" && principal.AgentID == strings.TrimSpace(agentID)
}

// ResolveAccessToken 识别网关共享 Token（仅管理面），或已登记电脑的 Agent 标识（控制台登录）。
func ResolveAccessToken(rawToken, gatewayToken string, tokens *agenttoken.Store) (Principal, bool) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return Principal{}, false
	}
	if ValidateToken(rawToken, gatewayToken) {
		return Principal{Admin: true}, true
	}
	return ResolveClientAccessToken(rawToken, gatewayToken, tokens)
}

// ResolveClientAccessToken 只接受已登记 Agent 标识。网关共享 Token 不能登录控制台。
func ResolveClientAccessToken(rawToken, gatewayToken string, tokens *agenttoken.Store) (Principal, bool) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return Principal{}, false
	}
	if ValidateToken(rawToken, gatewayToken) {
		return Principal{}, false
	}
	agentID, err := agenttoken.NormalizeAgentID(rawToken)
	if err != nil || !tokens.IsRegistered(agentID) {
		return Principal{}, false
	}
	return Principal{AgentID: agentID}, true
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := FromContext(r.Context())
		if !ok || !principal.Admin {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}
