package web

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/niclasedge/sopsy/internal/testutil"
)

func TestForeignHostForbidden(t *testing.T) {
	u := newUI(t)
	u.login()
	for _, host := range []string{"attacker.example:" + u.host[len("127.0.0.1:"):], "127.0.0.1:1", "localhost", "[::1]:1"} {
		req := httptest.NewRequest(http.MethodGet, "/keys", nil)
		req.Host = host
		wantStatus(t, u.request(req), http.StatusForbidden)
	}
	req := httptest.NewRequest(http.MethodGet, "/?t="+u.s.token, nil)
	req.Host = "attacker.example"
	r := u.request(req)
	wantStatus(t, r, http.StatusForbidden)
	if len(r.Result().Cookies()) > 0 {
		t.Error("a foreign Host got a session cookie")
	}
}

func TestLocalhostAllowed(t *testing.T) {
	u := newUI(t)
	u.login()
	req := httptest.NewRequest(http.MethodGet, "/retrieve", nil)
	req.Host = "localhost:" + u.host[len("127.0.0.1:"):]
	wantStatus(t, u.request(req), http.StatusOK)
}

func TestNoSessionForbidden(t *testing.T) {
	u := newUI(t)
	for _, path := range []string{"/", "/keys", "/setup", "/static/style.css", "/?t=wrong-token"} {
		wantStatus(t, u.get(path), http.StatusForbidden)
	}
	u.cookie = &http.Cookie{Name: u.s.cookieName(), Value: "forged"}
	wantStatus(t, u.get("/keys"), http.StatusForbidden)
}

func TestTokenExchange(t *testing.T) {
	u := newUI(t)
	r := u.get("/?t=" + u.s.token)
	wantStatus(t, r, http.StatusSeeOther)
	if loc := r.Header().Get("Location"); loc != "/" {
		t.Errorf("redirect to %q, want / without the token", loc)
	}
	var c *http.Cookie
	for _, rc := range r.Result().Cookies() {
		if rc.Name == u.s.cookieName() {
			c = rc
		}
	}
	if c == nil || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Value == u.s.token {
		t.Fatalf("session cookie %+v: want HttpOnly, SameSite=Strict, not the URL token", c)
	}
	u.cookie = c
	wantStatus(t, u.get("/keys"), http.StatusOK)
	// The printed URL keeps working for reopening the tab.
	wantStatus(t, u.get("/?t="+u.s.token), http.StatusSeeOther)
}

func TestTokenIs128Bits(t *testing.T) {
	u := newUI(t)
	for _, v := range []string{u.s.token, u.s.session, u.s.csrf} {
		if len(v) < 22 { // 16 bytes, unpadded base64
			t.Errorf("token %q is shorter than 128 bits", v)
		}
	}
	if u.s.token == u.s.session || u.s.session == u.s.csrf {
		t.Error("tokens are not independent")
	}
	if !strings.HasPrefix(u.s.URL(), "http://127.0.0.1:") {
		t.Errorf("URL %q is not on 127.0.0.1", u.s.URL())
	}
}

type fakeListener struct{ net.Listener }

func (fakeListener) Addr() net.Addr { return &net.TCPAddr{IP: net.IPv4zero, Port: 8080} }

func TestNewRefusesNonLoopback(t *testing.T) {
	if _, err := New(Config{}, fakeListener{}); err == nil {
		t.Fatal("New accepted a listener on 0.0.0.0")
	}
}

func TestPostGuardsLeaveFileUnchanged(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	u.login()
	before := u.read(u.secrets())
	form := url.Values{"name": {"API_TOKEN"}, "value": {"s3cr3t-test-value"}, "confirm": {"s3cr3t-test-value"}}

	req := func(method, contentType string, f url.Values) *http.Request {
		r := httptest.NewRequest(method, "/keys/set", strings.NewReader(f.Encode()))
		if contentType != "" {
			r.Header.Set("Content-Type", contentType)
		}
		return r
	}
	withCSRF := url.Values{"csrf": {u.s.csrf}}
	for k, v := range form {
		withCSRF[k] = v
	}
	wrongCSRF := url.Values{"csrf": {"forged"}}
	for k, v := range form {
		wrongCSRF[k] = v
	}
	tests := []struct {
		name string
		req  *http.Request
		code int
	}{
		{"missing CSRF token", req(http.MethodPost, "application/x-www-form-urlencoded", form), http.StatusForbidden},
		{"wrong CSRF token", req(http.MethodPost, "application/x-www-form-urlencoded", wrongCSRF), http.StatusForbidden},
		{"GET", req(http.MethodGet, "", withCSRF), http.StatusMethodNotAllowed},
		{"PUT", req(http.MethodPut, "application/x-www-form-urlencoded", withCSRF), http.StatusMethodNotAllowed},
		{"JSON", req(http.MethodPost, "application/json", withCSRF), http.StatusUnsupportedMediaType},
		{"multipart", req(http.MethodPost, "multipart/form-data; boundary=x", withCSRF), http.StatusUnsupportedMediaType},
		{"text/plain", req(http.MethodPost, "text/plain", withCSRF), http.StatusUnsupportedMediaType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantStatus(t, u.request(tt.req), tt.code)
			if u.read(u.secrets()) != before {
				t.Fatal("secrets file changed")
			}
		})
	}
	// The same form with everything right does change the file.
	wantStatus(t, u.request(req(http.MethodPost, "application/x-www-form-urlencoded", withCSRF)), http.StatusSeeOther)
	if u.read(u.secrets()) == before {
		t.Fatal("valid request did not change the file")
	}
}

func TestSecurityHeadersOnEveryRoute(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	check := func(t *testing.T, r *httptest.ResponseRecorder) {
		t.Helper()
		h := r.Header()
		csp := h.Get("Content-Security-Policy")
		for _, want := range []string{"default-src 'none'", "script-src 'self'", "style-src 'self'", "frame-ancestors 'none'"} {
			if !strings.Contains(csp, want) {
				t.Errorf("CSP %q lacks %q", csp, want)
			}
		}
		if strings.Contains(csp, "unsafe-inline") {
			t.Errorf("CSP allows inline code: %q", csp)
		}
		for k, v := range map[string]string{"Cache-Control": "no-store", "X-Content-Type-Options": "nosniff", "Referrer-Policy": "no-referrer"} {
			if h.Get(k) != v {
				t.Errorf("%s = %q, want %q", k, h.Get(k), v)
			}
		}
	}
	// Rejected requests carry the headers too.
	check(t, u.get("/keys"))
	check(t, u.get("/?t=wrong"))
	u.login()
	for _, path := range []string{"/", "/setup", "/keys", "/recipients", "/retrieve", "/static/style.css", "/static/copy.js", "/missing"} {
		t.Run(path, func(t *testing.T) { check(t, u.get(path)) })
	}
	check(t, u.post("/keys/set", url.Values{"csrf": {"forged"}}))
	check(t, u.post("/keys/delete", url.Values{"name": {"API_TOKEN"}}))
}

func TestPagesHaveNoInlineScript(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	u.login()
	for _, path := range []string{"/setup", "/keys", "/recipients", "/retrieve"} {
		body := u.get(path).Body.String()
		if strings.Contains(body, "<script>") || strings.Contains(body, " onclick=") || strings.Contains(body, "style=\"") {
			t.Errorf("%s has inline script or style, which the CSP blocks", path)
		}
	}
}

func TestLogHasNoFormFieldsOrToken(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	u.login()
	u.follow("/keys/set", url.Values{"name": {"NEW_KEY"}, "value": {"s3cr3t-test-value"}, "confirm": {"s3cr3t-test-value"}})
	log := u.log.String()
	wantContains(t, log, "POST /keys/set 303", "GET / 303")
	testutil.AssertNoSecret(t, log)
	for _, s := range []string{u.s.token, u.s.session, u.s.csrf, "NEW_KEY", "csrf="} {
		if strings.Contains(log, s) {
			t.Errorf("log contains %q:\n%s", s, log)
		}
	}
}

func serve(t *testing.T, idle time.Duration) (*Server, chan error, context.CancelFunc) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Idle: idle, Dir: t.TempDir()}, ln)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx) }()
	return s, done, cancel
}

func TestIdleShutdown(t *testing.T) {
	_, done, _ := serve(t, 100*time.Millisecond)
	select {
	case err := <-done:
		if !errors.Is(err, ErrIdle) {
			t.Fatalf("Serve returned %v, want ErrIdle", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop when idle")
	}
}

func TestRequestsKeepServerAlive(t *testing.T) {
	s, done, _ := serve(t, 300*time.Millisecond)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for range 5 {
		time.Sleep(150 * time.Millisecond)
		resp, err := client.Get(s.URL())
		if err != nil {
			t.Fatalf("server stopped while in use: %v", err)
		}
		_ = resp.Body.Close()
	}
	select {
	case err := <-done:
		t.Fatalf("server stopped early: %v", err)
	default:
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop after the requests ended")
	}
}

func TestServeStopsOnCancel(t *testing.T) {
	_, done, cancel := serve(t, time.Hour)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
}

func TestTokenRedirectStaysLocal(t *testing.T) {
	u := newUI(t)
	r := u.get("//attacker.example/?t=" + u.s.token)
	wantStatus(t, r, http.StatusSeeOther)
	if loc := r.Header().Get("Location"); loc != "/" {
		t.Fatalf("redirect to %q, want /", loc)
	}
}
