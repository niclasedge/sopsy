package cli

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"testing"
	"time"
)

// startUI runs `sopsy ui` with the given flags and returns the URL it opened
// and a channel that receives the result once it stops.
func startUI(t *testing.T, w *world, browserErr error, args ...string) (string, chan result) {
	t.Helper()
	opened := make(chan string, 1)
	w.browser = func(url string) error {
		opened <- url
		return browserErr
	}
	done := make(chan result, 1)
	go func() { done <- w.exec(append([]string{"ui"}, args...)) }()
	select {
	case url := <-opened:
		return url, done
	case r := <-done:
		t.Fatalf("ui exited early with %d:\n%s%s", r.code, r.stdout, r.stderr)
	case <-time.After(10 * time.Second):
		t.Fatal("ui did not open a browser")
	}
	return "", nil
}

func waitUI(t *testing.T, done chan result) result {
	t.Helper()
	select {
	case r := <-done:
		return r
	case <-time.After(10 * time.Second):
		t.Fatal("ui did not stop")
	}
	return result{}
}

func TestUIServesAndStopsWhenIdle(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	w.useFixture()
	url, done := startUI(t, w, nil, "--idle", "500ms")
	if !strings.HasPrefix(url, "http://127.0.0.1:") || !strings.Contains(url, "/?t=") {
		t.Fatalf("opened %q", url)
	}
	jar, _ := cookiejar.New(nil)
	resp, err := (&http.Client{Jar: jar}).Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Request.URL.Path != "/keys" || strings.Contains(resp.Request.URL.RawQuery, "t=") {
		t.Fatalf("landed on %s with %d, want /keys without token", resp.Request.URL, resp.StatusCode)
	}
	wantContains(t, string(body), "API_TOKEN")

	r := waitUI(t, done)
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, url, "stopped: no requests for 500ms")
	if strings.Contains(r.stderr, "t=") {
		t.Errorf("the token reached the log:\n%s", r.stderr)
	}
}

func TestUIWithoutBrowserStillServes(t *testing.T) {
	w := newWorld(t)
	url, done := startUI(t, w, errBrowser, "--idle", "300ms")
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("server not running after the browser failed: %v", err)
	}
	_ = resp.Body.Close()
	r := waitUI(t, done)
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, url)
	wantContains(t, r.stderr, "could not open a browser", "open the link above")
}

var errBrowser = errors.New("no display")

func TestUIFixedPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	// Taken port: a clear error.
	r := newWorld(t).run("ui", "--port", port)
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, "cannot listen on 127.0.0.1:"+port, "--port")
	_ = ln.Close()

	url, done := startUI(t, newWorld(t), nil, "--port", port, "--idle", "200ms")
	if !strings.HasPrefix(url, "http://127.0.0.1:"+port+"/") {
		t.Fatalf("opened %q, want port %s", url, port)
	}
	waitUI(t, done).wantCode(t, ExitOK)
}

func TestUIRejectsBadFlags(t *testing.T) {
	for _, args := range [][]string{
		{"ui", "--port", "70000"},
		{"ui", "--idle", "0s"},
		{"ui", "extra"},
	} {
		r := newWorld(t).run(args...)
		r.wantCode(t, ExitFail)
		wantContains(t, r.stderr, "usage: sopsy ui")
	}
}
