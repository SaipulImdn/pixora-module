package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func newTestHTTP(t *testing.T, prefix string) *HTTP {
	t.Helper()
	// A fresh registry per test avoids "duplicate metrics collector
	// registration" panics against the shared global default registry.
	return NewHTTP(Config{Prefix: prefix, Registerer: prometheus.NewRegistry()})
}

func TestObserve_RecordsAllFourMetrics(t *testing.T) {
	h := newTestHTTP(t, "test")

	h.Observe("GET", "/api/v1/foo/:id", "200", "00", 150*time.Millisecond)

	if got := testutilCounterValue(t, h.RequestsTotal.WithLabelValues("GET", "/api/v1/foo/:id", "200", "00")); got != 1 {
		t.Errorf("RequestsTotal = %v, want 1", got)
	}
	if got := testutilCounterValue(t, h.ResponseCodeTotal.WithLabelValues("00", "/api/v1/foo/:id")); got != 1 {
		t.Errorf("ResponseCodeTotal = %v, want 1", got)
	}
}

func TestObserve_SkipsResponseCodeMetricWhenEmpty(t *testing.T) {
	h := newTestHTTP(t, "test2")

	h.Observe("GET", "/foo", "204", "", time.Millisecond)

	m := &dto.Metric{}
	if err := h.ResponseCodeTotal.WithLabelValues("", "/foo").Write(m); err != nil {
		t.Fatalf("write metric: %v", err)
	}
	if m.Counter.GetValue() != 0 {
		t.Errorf("expected no increment for empty response code, got %v", m.Counter.GetValue())
	}
}

func TestIncDecActive(t *testing.T) {
	h := newTestHTTP(t, "test3")

	h.IncActive()
	h.IncActive()
	h.DecActive()

	m := &dto.Metric{}
	if err := h.ActiveConnections.Write(m); err != nil {
		t.Fatalf("write metric: %v", err)
	}
	if got := m.Gauge.GetValue(); got != 1 {
		t.Errorf("ActiveConnections = %v, want 1", got)
	}
}

func TestNormalizePath(t *testing.T) {
	cases := map[string]string{
		"/api/v1/users/123": "/api/v1/users/:id",
		"/api/v1/files/550e8400-e29b-41d4-a716-446655440000": "/api/v1/files/:id",
		"/api/v1/profile": "/api/v1/profile",
	}
	for in, want := range cases {
		if got := NormalizePath(in); got != want {
			t.Errorf("NormalizePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func testutilCounterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	m := &dto.Metric{}
	if err := c.Write(m); err != nil {
		t.Fatalf("write metric: %v", err)
	}
	return m.Counter.GetValue()
}
