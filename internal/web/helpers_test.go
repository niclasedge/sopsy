package web

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/store"
	"github.com/niclasedge/sopsy/internal/testutil"
)

// ui is one UI server on an isolated machine: its own home, project
// directory and environment. Requests go straight to the handler.
type ui struct {
	t      *testing.T
	s      *Server
	home   string
	dir    string
	vars   map[string]string
	host   string
	cookie *http.Cookie
	log    bytes.Buffer
}

func newUI(t *testing.T) *ui {
	t.Helper()
	u := &ui{t: t, home: t.TempDir(), dir: t.TempDir(), vars: map[string]string{}}
	if runtime.GOOS == "windows" {
		u.vars["AppData"] = filepath.Join(u.home, "AppData", "Roaming")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	getenv := func(k string) string { return u.vars[k] }
	s, err := New(Config{
		Keys:        keys.Env{GOOS: runtime.GOOS, Home: u.home, Getenv: getenv},
		Dir:         u.dir,
		SecretsFile: filepath.Join(u.dir, "secrets.env"),
		Executable:  "/usr/local/bin/sopsy",
		Log:         &u.log,
		LookPath:    func(string) (string, error) { return "", errors.New("not installed") },
	}, ln)
	if err != nil {
		t.Fatal(err)
	}
	u.s = s
	u.host = "127.0.0.1:" + strconv.Itoa(s.port)
	return u
}

// useFixture installs the test-only key and the sops-CLI-encrypted fixture
// (API_TOKEN, DB_PASSWORD) with a matching .sops.yaml.
func (u *ui) useFixture() {
	u.t.Helper()
	u.vars[keys.EnvKeyFile] = testutil.KeyFile()
	testutil.CopyFixture(u.t, u.secrets())
	u.write(".sops.yaml", "creation_rules:\n  - age: "+testutil.PublicKey+"\n")
}

func (u *ui) secrets() string { return filepath.Join(u.dir, "secrets.env") }

func (u *ui) write(name, content string) {
	u.t.Helper()
	if err := os.WriteFile(filepath.Join(u.dir, name), []byte(content), 0o644); err != nil {
		u.t.Fatal(err)
	}
}

func (u *ui) read(path string) string {
	u.t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		u.t.Fatal(err)
	}
	return string(data)
}

// login exchanges the URL token for the session cookie.
func (u *ui) login() {
	u.t.Helper()
	r := u.request(httptest.NewRequest(http.MethodGet, "/?t="+u.s.token, nil))
	for _, c := range r.Result().Cookies() {
		if c.Name == u.s.cookieName() {
			u.cookie = c
		}
	}
	if u.cookie == nil {
		u.t.Fatalf("login set no session cookie; status %d", r.Code)
	}
}

// request sends req with the server's Host and the session cookie (if
// logged in) and checks that no secret value is in the response.
func (u *ui) request(req *http.Request) *httptest.ResponseRecorder {
	u.t.Helper()
	if req.Host == "example.com" { // httptest's default
		req.Host = u.host
	}
	if u.cookie != nil {
		req.AddCookie(u.cookie)
	}
	rec := httptest.NewRecorder()
	u.s.Handler().ServeHTTP(rec, req)
	testutil.AssertNoSecret(u.t, rec.Body.String(), rec.Header().Get("Location"))
	return rec
}

func (u *ui) get(path string) *httptest.ResponseRecorder {
	u.t.Helper()
	return u.request(httptest.NewRequest(http.MethodGet, path, nil))
}

// post submits a form with the session's CSRF token.
func (u *ui) post(path string, form url.Values) *httptest.ResponseRecorder {
	u.t.Helper()
	if form == nil {
		form = url.Values{}
	}
	if !form.Has("csrf") {
		form.Set("csrf", u.s.csrf)
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return u.request(req)
}

// follow posts a form, expects the redirect and returns the page it leads to.
func (u *ui) follow(path string, form url.Values) string {
	u.t.Helper()
	r := u.post(path, form)
	if r.Code != http.StatusSeeOther {
		u.t.Fatalf("POST %s: status %d, want 303\n%s", path, r.Code, r.Body)
	}
	return u.get(r.Header().Get("Location")).Body.String()
}

func (u *ui) entries() map[string]string {
	u.t.Helper()
	f, err := store.Load(u.secrets())
	if err != nil {
		u.t.Fatal(err)
	}
	k, err := keys.LoadFile(testutil.KeyFile())
	if err != nil {
		u.t.Fatal(err)
	}
	p, err := f.Decrypt(k.Identities, k.Public)
	if err != nil {
		u.t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range p.Entries() {
		out[e.Name] = e.Value
	}
	return out
}

func wantStatus(t *testing.T, r *httptest.ResponseRecorder, code int) {
	t.Helper()
	if r.Code != code {
		t.Fatalf("status %d, want %d\n%s", r.Code, code, r.Body)
	}
}

func wantContains(t *testing.T, s string, subs ...string) {
	t.Helper()
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			t.Errorf("missing %q in:\n%s", sub, s)
		}
	}
}
