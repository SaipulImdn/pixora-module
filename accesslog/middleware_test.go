package accesslog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/SaipulImdn/pixora-module/reqid"
)

func newTestMiddleware() (*observer.ObservedLogs, func(http.Handler) http.Handler) {
	core, logs := observer.New(zap.DebugLevel)
	cfg := Config{ServiceName: "test-svc", Logger: zap.New(core)}
	return logs, Middleware(cfg)
}

func TestMiddleware_GeneratesRequestIDWhenMissing(t *testing.T) {
	logs, mw := newTestMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reqid.FromContext(r.Context()) == "" {
			t.Error("expected request ID to be set in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get(reqid.Header) == "" {
		t.Error("expected X-Request-ID response header to be set")
	}
	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("got %d log entries, want 1", len(entries))
	}
	if id, ok := entries[0].ContextMap()["request_id"].(string); !ok || id == "" {
		t.Error("expected non-empty request_id field in log entry")
	}
}

func TestMiddleware_PropagatesIncomingRequestID(t *testing.T) {
	logs, mw := newTestMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	req.Header.Set(reqid.Header, "incoming-id-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(reqid.Header); got != "incoming-id-123" {
		t.Errorf("response X-Request-ID = %q, want incoming-id-123 (passthrough)", got)
	}
	if id := logs.All()[0].ContextMap()["request_id"]; id != "incoming-id-123" {
		t.Errorf("logged request_id = %v, want incoming-id-123", id)
	}
}

func TestMiddleware_SkipsConfiguredPaths(t *testing.T) {
	logs, mw := newTestMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if len(logs.All()) != 0 {
		t.Errorf("expected /health to be skipped, got %d log entries", len(logs.All()))
	}
}

func TestMiddleware_LogsErrorLevelOn5xx(t *testing.T) {
	logs, mw := newTestMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := logs.All()[0].Level; got != zap.ErrorLevel {
		t.Errorf("log level = %v, want error for a 5xx response", got)
	}
}

func TestExtractResponseEnvelope(t *testing.T) {
	body := []byte(`{"responseCode":"00","responseDesc":"success","responseData":{"id":1}}`)
	if got := extractStringField(body, "responseCode"); got != "00" {
		t.Errorf("responseCode = %q, want 00", got)
	}
	if got := extractStringField(body, "responseDesc"); got != "success" {
		t.Errorf("responseDesc = %q, want success", got)
	}
	if got := extractResponseField(body, "responseData"); got != `{"id":1}` {
		t.Errorf("responseData = %q, want {\"id\":1}", got)
	}
}
