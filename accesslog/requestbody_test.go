package accesslog

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newReq(t *testing.T, contentType, body string, contentLength int64) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/x", io.NopCloser(strings.NewReader(body)))
	r.Header.Set("Content-Type", contentType)
	r.ContentLength = contentLength
	return r
}

func TestCaptureRequestBody_SmallJSON_CapturedAndRestored(t *testing.T) {
	body := `{"name":"a"}`
	r := newReq(t, "application/json", body, int64(len(body)))

	got, size := CaptureRequestBody(r, 0)
	if got != body {
		t.Errorf("body = %q, want %q", got, body)
	}
	if size != int64(len(body)) {
		t.Errorf("size = %d, want %d", size, len(body))
	}

	restored, _ := io.ReadAll(r.Body)
	if string(restored) != body {
		t.Errorf("r.Body after capture = %q, want %q (handler must still see it)", restored, body)
	}
}

func TestCaptureRequestBody_RedactsSensitiveFields(t *testing.T) {
	body := `{"email":"a@b.com","password":"hunter2","refreshToken":"abc.def"}`
	r := newReq(t, "application/json", body, int64(len(body)))

	got, _ := CaptureRequestBody(r, 0)
	if strings.Contains(got, "hunter2") || strings.Contains(got, "abc.def") {
		t.Errorf("body = %q, sensitive value leaked", got)
	}
	if !strings.Contains(got, `"password":"***"`) || !strings.Contains(got, `"refreshToken":"***"`) {
		t.Errorf("body = %q, want redacted password/refreshToken fields", got)
	}
	if !strings.Contains(got, `"email":"a@b.com"`) {
		t.Errorf("body = %q, non-sensitive field should survive", got)
	}
}

func TestCaptureRequestBody_RedactsSnakeCaseFields(t *testing.T) {
	body := `{"email":"a@b.com","access_token":"secret-at","refresh_token":"secret-rt","api_key":"secret-ak"}`
	r := newReq(t, "application/json", body, int64(len(body)))

	got, _ := CaptureRequestBody(r, 0)
	for _, leaked := range []string{"secret-at", "secret-rt", "secret-ak"} {
		if strings.Contains(got, leaked) {
			t.Errorf("body = %q, sensitive value %q leaked", got, leaked)
		}
	}
	for _, field := range []string{"access_token", "refresh_token", "api_key"} {
		if !strings.Contains(got, `"`+field+`":"***"`) {
			t.Errorf("body = %q, want redacted %q field", got, field)
		}
	}
}

func TestCaptureRequestBody_Multipart_NeverBuffered(t *testing.T) {
	body := "--boundary\r\nfake file bytes\r\n--boundary--"
	r := newReq(t, "multipart/form-data; boundary=boundary", body, int64(len(body)))

	got, size := CaptureRequestBody(r, 0)
	if got != "[binary/multipart]" {
		t.Errorf("body = %q, want placeholder", got)
	}
	if size != int64(len(body)) {
		t.Errorf("size = %d, want %d", size, len(body))
	}
	restored, _ := io.ReadAll(r.Body)
	if string(restored) != body {
		t.Errorf("r.Body was consumed for a multipart request; handler would see nothing")
	}
}

func TestCaptureRequestBody_OverLimit_ReportedBySizeOnly(t *testing.T) {
	body := strings.Repeat("x", 100)
	r := newReq(t, "application/json", body, int64(len(body)))

	got, size := CaptureRequestBody(r, 10) // maxBody smaller than body
	if !strings.Contains(got, "large body") {
		t.Errorf("body = %q, want large-body placeholder", got)
	}
	if size != int64(len(body)) {
		t.Errorf("size = %d, want %d", size, len(body))
	}
	restored, _ := io.ReadAll(r.Body)
	if string(restored) != body {
		t.Errorf("r.Body was consumed even though it was never read for capture")
	}
}

func TestCaptureRequestBody_UnknownLength_LeftUntouched(t *testing.T) {
	body := `{"a":1}`
	r := newReq(t, "application/json", body, -1) // chunked / unknown length

	got, size := CaptureRequestBody(r, 0)
	if got != "" || size != 0 {
		t.Errorf("got = (%q, %d), want (\"\", 0) for unknown-length body", got, size)
	}
	restored, _ := io.ReadAll(r.Body)
	if string(restored) != body {
		t.Errorf("r.Body was consumed despite unknown length")
	}
}

func TestCaptureRequestBody_NilBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.Body = nil

	got, size := CaptureRequestBody(r, 0)
	if got != "" || size != 0 {
		t.Errorf("got = (%q, %d), want (\"\", 0) for nil body", got, size)
	}
}
