package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/soockee/go-service-template/internal/config"
	"github.com/soockee/go-service-template/internal/notes"
)

// newTestServer builds the real handler (routes + middleware) over an
// in-memory store. Tests go through HTTP, not by calling handlers.
func newTestServer(t *testing.T, mods ...func(*Deps)) http.Handler {
	t.Helper()
	cfg, err := config.LoadFrom(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	deps := Deps{
		Notes:  notes.NewService(notes.NewMemStore()),
		Assets: fstest.MapFS{"css/output.css": {Data: []byte("body{}")}},
	}
	for _, m := range mods {
		m(&deps)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(cfg, log, deps).Handler()
}

func do(t *testing.T, h http.Handler, method, target, body string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

var jsonCT = map[string]string{"Content-Type": "application/json"}

func TestNotesAPI(t *testing.T) {
	h := newTestServer(t)

	rec := do(t, h, "POST", "/api/notes", `{"title":"hello","body":"world"}`, jsonCT)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: status %d, body %s", rec.Code, rec.Body)
	}
	if loc := rec.Header().Get("Location"); loc != "/api/notes/1" {
		t.Errorf("Location = %q", loc)
	}

	tests := []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"list", "GET", "/api/notes", "", 200, `"title":"hello"`},
		{"get", "GET", "/api/notes/1", "", 200, `"body":"world"`},
		{"get missing", "GET", "/api/notes/42", "", 404, `{"error":"note not found"}`},
		{"get bad id", "GET", "/api/notes/abc", "", 400, `id must be an integer`},
		{"create invalid", "POST", "/api/notes", `{"title":" "}`, 400, `title is required`},
		{"create unknown field", "POST", "/api/notes", `{"titel":"x"}`, 400, `invalid JSON body`},
		{"wrong method", "DELETE", "/api/notes/1", "", 405, ``},
		{"unknown api path", "GET", "/api/nope", "", 404, `{"error":"not found"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, h, tt.method, tt.target, tt.body, jsonCT)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body = %s, want it to contain %s", rec.Body, tt.wantBody)
			}
		})
	}
}

func TestPages(t *testing.T) {
	h := newTestServer(t)
	form := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	htmx := map[string]string{"Content-Type": "application/x-www-form-urlencoded", "HX-Request": "true"}

	rec := do(t, h, "POST", "/notes", url.Values{"title": {"from form"}}.Encode(), form)
	if rec.Code != http.StatusSeeOther {
		t.Errorf("form post: status %d, want 303", rec.Code)
	}

	rec = do(t, h, "POST", "/notes", url.Values{"title": {"<b>via htmx</b>"}}.Encode(), htmx)
	if rec.Code != http.StatusCreated || !strings.HasPrefix(strings.TrimSpace(rec.Body.String()), "<li") {
		t.Errorf("htmx post: status %d, body %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "<b>") {
		t.Error("htmx post: title was not HTML-escaped")
	}

	rec = do(t, h, "GET", "/", "", nil)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "from form") {
		t.Errorf("home: status %d, body %s", rec.Code, rec.Body)
	}

	rec = do(t, h, "GET", "/missing", "", nil)
	if rec.Code != 404 || !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Errorf("404 page: status %d, content-type %q", rec.Code, rec.Header().Get("Content-Type"))
	}

	rec = do(t, h, "GET", "/assets/css/output.css", "", nil)
	if rec.Code != 200 || rec.Body.String() != "body{}" {
		t.Errorf("asset: status %d, body %q", rec.Code, rec.Body)
	}
}

func TestCrossSiteFormPostRejected(t *testing.T) {
	h := newTestServer(t)
	rec := do(t, h, "POST", "/notes", "title=x", map[string]string{
		"Content-Type":   "application/x-www-form-urlencoded",
		"Sec-Fetch-Site": "cross-site",
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestHealth(t *testing.T) {
	h := newTestServer(t)
	if rec := do(t, h, "GET", "/healthz", "", nil); rec.Code != 200 {
		t.Errorf("healthz: %d", rec.Code)
	}
	if rec := do(t, h, "GET", "/readyz", "", nil); rec.Code != 200 {
		t.Errorf("readyz: %d", rec.Code)
	}

	down := newTestServer(t, func(d *Deps) {
		d.Ready = func(context.Context) error { return errors.New("db down") }
	})
	if rec := do(t, down, "GET", "/readyz", "", nil); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("readyz when down: %d", rec.Code)
	}
}

func TestMiddleware(t *testing.T) {
	h := newTestServer(t)

	rec := do(t, h, "GET", "/healthz", "", map[string]string{"X-Request-ID": "abc"})
	if got := rec.Header().Get("X-Request-ID"); got != "abc" {
		t.Errorf("X-Request-ID = %q, want echoed", got)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing security headers")
	}

	var s Server
	s.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	panicky := s.recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	if rec := do(t, panicky, "GET", "/", "", nil); rec.Code != 500 {
		t.Errorf("panic: status %d, want 500", rec.Code)
	}
}

func TestCORS(t *testing.T) {
	cfg, _ := config.LoadFrom(func(k string) string {
		if k == "CORS_ALLOWED_ORIGINS" {
			return "https://app.example"
		}
		return ""
	})
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(cfg, log, Deps{Notes: notes.NewService(notes.NewMemStore())}).Handler()

	rec := do(t, h, "OPTIONS", "/api/notes", "", map[string]string{
		"Origin": "https://app.example", "Access-Control-Request-Method": "POST",
	})
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Errorf("preflight: status %d, headers %v", rec.Code, rec.Header())
	}

	rec = do(t, h, "POST", "/api/notes", `{"title":"x"}`, map[string]string{
		"Content-Type": "application/json", "Origin": "https://app.example", "Sec-Fetch-Site": "cross-site",
	})
	if rec.Code != http.StatusCreated {
		t.Errorf("trusted cross-origin POST: status %d, body %s", rec.Code, rec.Body)
	}

	rec = do(t, h, "GET", "/api/notes", "", map[string]string{"Origin": "https://evil.example"})
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS header sent for disallowed origin")
	}
}

func TestServeGracefulShutdown(t *testing.T) {
	cfg, _ := config.LoadFrom(func(string) string { return "" })
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, log, Deps{Notes: notes.NewService(notes.NewMemStore())})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx, ln) }()

	resp, err := http.Get("http://" + ln.Addr().String() + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	_ = resp.Body.Close()
	if body["status"] != "ok" {
		t.Errorf("healthz body = %v", body)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Serve returned %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after cancel")
	}
}
