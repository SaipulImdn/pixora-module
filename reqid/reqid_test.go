package reqid

import "testing"

func TestFromHeaderOrGenerate_PassesThroughExisting(t *testing.T) {
	got := FromHeaderOrGenerate("abc-123")
	if got != "abc-123" {
		t.Errorf("got %q, want passthrough of existing header value", got)
	}
}

func TestFromHeaderOrGenerate_GeneratesWhenMissing(t *testing.T) {
	got := FromHeaderOrGenerate("")
	if len(got) != idLen*2 {
		t.Errorf("generated ID length = %d, want %d hex chars", len(got), idLen*2)
	}
}

func TestGenerate_ProducesDistinctIDs(t *testing.T) {
	a := Generate()
	b := Generate()
	if a == b {
		t.Errorf("two calls to Generate() produced the same ID: %q", a)
	}
}

func TestContext_RoundTrip(t *testing.T) {
	ctx := WithID(t.Context(), "req-42")
	if got := FromContext(ctx); got != "req-42" {
		t.Errorf("FromContext = %q, want req-42", got)
	}
}

func TestFromContext_EmptyWhenUnset(t *testing.T) {
	if got := FromContext(t.Context()); got != "" {
		t.Errorf("FromContext on bare context = %q, want empty", got)
	}
}
