package handler

import (
	"net/http"

	"github.com/liveagent/agent-gateway/internal/auth"
	"github.com/liveagent/agent-gateway/internal/observability"
	"github.com/liveagent/agent-gateway/internal/session"
)

// statusResponse 是全局鉴权检查与 Agent 目录响应，不承担具体 Agent 寻址。
type statusResponse struct {
	Agents        []session.Status `json:"agents"`
	ProtocolUsage map[string]int64 `json:"protocol_usage"`
}

func Status(sm *session.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agents := sm.AgentStatuses()
		if principal, ok := auth.FromContext(r.Context()); ok && !principal.Admin {
			filtered := agents[:0]
			for _, agent := range agents {
				if auth.AllowsAgent(principal, agent.AgentID) {
					filtered = append(filtered, agent)
				}
			}
			agents = filtered
		}
		writeJSON(w, http.StatusOK, statusResponse{
			Agents:        agents,
			ProtocolUsage: observability.Usage.Snapshot(),
		})
	}
}
