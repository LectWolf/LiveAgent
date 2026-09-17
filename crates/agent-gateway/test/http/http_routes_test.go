package httproutes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/liveagent/agent-gateway/internal/config"
	"github.com/liveagent/agent-gateway/internal/server"
	"github.com/liveagent/agent-gateway/internal/session"
)

func newHTTPTestHandler(sm *session.Manager) http.Handler {
	return server.NewHTTPServer(&config.Config{
		Token:          "dev-token",
		RequestTimeout: 500 * time.Millisecond,
	}, sm, nil)
}

func TestAPIRoutesRequireBearerToken(t *testing.T) {
	t.Parallel()

	handler := newHTTPTestHandler(session.NewManager())

	req := httptest.NewRequest(http.MethodGet, "http://gateway.test/api/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("content-type = %q, want JSON", contentType)
	}
	if !strings.Contains(rec.Body.String(), "unauthorized") {
		t.Fatalf("body = %q, want unauthorized error", rec.Body.String())
	}
}

func TestHealthRouteIsPublic(t *testing.T) {
	t.Parallel()

	handler := newHTTPTestHandler(session.NewManager())

	req := httptest.NewRequest(http.MethodGet, "http://gateway.test/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("body = %q, want health payload", rec.Body.String())
	}
}

func TestStatusRouteRejectsGatewayToken(t *testing.T) {
	t.Parallel()

	handler := newHTTPTestHandler(session.NewManager())

	req := httptest.NewRequest(http.MethodGet, "http://gateway.test/api/status", nil)
	req.Header.Set("Authorization", " bearer   dev-token ")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("gateway token console login status = %d, want %d, body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestSPAFallbackServesIndexWithoutAuthorization(t *testing.T) {
	t.Parallel()

	handler := newHTTPTestHandler(session.NewManager())

	req := httptest.NewRequest(http.MethodGet, "http://gateway.test/conversations/session-1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if location := rec.Header().Get("Location"); location != "" {
		t.Fatalf("expected no redirect, got Location=%q", location)
	}
	if !strings.Contains(rec.Body.String(), "<title>LiveAgent Gateway</title>") {
		t.Fatalf("expected embedded WebUI index.html, got %q", rec.Body.String())
	}
}
