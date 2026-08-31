package recovery

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestMiddleware_RecoversPanic(t *testing.T) {
	h := Middleware(zaptest.NewLogger(t))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"response_code":"ISE"`) {
		t.Fatalf("body = %q, want ISE envelope", rr.Body.String())
	}
}

func TestMiddleware_PassesThrough(t *testing.T) {
	h := Middleware(zaptest.NewLogger(t))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "ok")
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rr.Code != http.StatusCreated || rr.Body.String() != "ok" {
		t.Fatalf("got %d %q, want 201 \"ok\"", rr.Code, rr.Body.String())
	}
}

func TestMiddleware_RePanicsAbortHandler(t *testing.T) {
	defer func() {
		if rec := recover(); rec != http.ErrAbortHandler {
			t.Fatalf("recovered %v, want ErrAbortHandler to propagate", rec)
		}
	}()
	h := Middleware(zaptest.NewLogger(t))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
}
