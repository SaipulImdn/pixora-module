package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReadiness_AllOK(t *testing.T) {
	h := Readiness(time.Second,
		Check{Name: "mysql", Ping: func(context.Context) error { return nil }},
	)
	rr := httptest.NewRecorder()
	h(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"status":"ready"`) {
		t.Fatalf("got %d %q", rr.Code, rr.Body.String())
	}
}

func TestReadiness_DepDown(t *testing.T) {
	h := Readiness(time.Second,
		Check{Name: "mysql", Ping: func(context.Context) error { return nil }},
		Check{Name: "redis", Ping: func(context.Context) error { return errors.New("nope") }},
	)
	rr := httptest.NewRecorder()
	h(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"redis":"down"`) {
		t.Fatalf("body = %q, want redis down", rr.Body.String())
	}
}

func TestLiveness_Static(t *testing.T) {
	rr := httptest.NewRecorder()
	Liveness(map[string]string{"version": "1.2.3"})(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"version":"1.2.3"`) {
		t.Fatalf("got %d %q", rr.Code, rr.Body.String())
	}
}
