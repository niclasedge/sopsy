// Package web is the local guided UI behind `sopsy ui`: a loopback-only HTTP
// server that lets a person set values straight into the encrypted file and
// never sends a value back.
package web

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/niclasedge/sopsy/internal/keys"
)

// DefaultIdle is how long the server waits for a request before it stops.
const DefaultIdle = 15 * time.Minute

// ErrIdle is returned by Serve when the server stopped for lack of requests.
var ErrIdle = errors.New("no requests")

// maxBody caps form submissions; a dotenv value is one line.
const maxBody = 1 << 20

// Config is what the UI works on.
type Config struct {
	Keys        keys.Env
	Dir         string
	SecretsFile string
	// Executable is the sopsy binary, used in the cron example.
	Executable string
	// Idle stops the server after this long without a request; 0 means
	// DefaultIdle.
	Idle time.Duration
	// Log receives one line per request (method, path, status) and server
	// errors. Query strings and bodies are never logged.
	Log io.Writer
	// LookPath finds external tools for the doctor checks; nil means
	// exec.LookPath.
	LookPath func(string) (string, error)
}

// Server is one `sopsy ui` run.
type Server struct {
	cfg  Config
	ln   net.Listener
	port int
	log  *log.Logger
	// token is the secret in the printed URL; session is the cookie value it
	// is exchanged for; csrf is the per-session form token.
	token, session, csrf string
	activity             chan struct{}
	pages                map[string]*page

	// mu serializes file changes and guards flash.
	mu    sync.Mutex
	flash *flash
}

// New prepares a server on ln, which must listen on a loopback address.
func New(cfg Config, ln net.Listener) (*Server, error) {
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok || !addr.IP.IsLoopback() {
		return nil, fmt.Errorf("refusing to serve on %s: only loopback addresses are allowed", ln.Addr())
	}
	if cfg.Idle == 0 {
		cfg.Idle = DefaultIdle
	}
	if cfg.Log == nil {
		cfg.Log = io.Discard
	}
	s := &Server{
		cfg:      cfg,
		ln:       ln,
		port:     addr.Port,
		log:      log.New(cfg.Log, "sopsy ui: ", 0),
		activity: make(chan struct{}, 1),
		pages:    loadPages(),
	}
	for _, p := range []*string{&s.token, &s.session, &s.csrf} {
		v, err := randomToken()
		if err != nil {
			return nil, err
		}
		*p = v
	}
	return s, nil
}

// randomToken returns 128 random bits, URL-safe encoded.
func randomToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// URL is the address to open, including the one-time token.
func (s *Server) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/?t=%s", s.port, s.token)
}

// Serve handles requests until ctx is done (returns nil) or no request
// arrived for the idle duration (returns ErrIdle).
func (s *Server) Serve(ctx context.Context) error {
	srv := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       time.Minute,
		ErrorLog:          s.log,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(s.ln) }()

	idle := time.NewTimer(s.cfg.Idle)
	defer idle.Stop()
	var reason error
wait:
	for {
		select {
		case <-s.activity:
			idle.Reset(s.cfg.Idle)
		case <-idle.C:
			reason = ErrIdle
			break wait
		case <-ctx.Done():
			break wait
		case err := <-errc:
			return err
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		return err
	}
	return reason
}

// Handler is the full middleware chain: headers → log → host → session →
// routes.
func (s *Server) Handler() http.Handler {
	return s.headers(s.logRequests(s.checkHost(s.authenticate(s.routes()))))
}

// headers hardens every response, including errors.
func (s *Server) headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; script-src 'self'; img-src 'self'; "+
			"form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
		h.Set("Cache-Control", "no-store")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// logRequests logs method, path and status — never the query string (it
// can hold the token) and never the body (it can hold a value).
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		s.log.Printf("%s %s %d", r.Method, r.URL.Path, sw.status)
	})
}

// checkHost rejects any Host but 127.0.0.1:<port> and localhost:<port>, so a
// page on another site cannot reach the server through DNS rebinding.
func (s *Server) checkHost(next http.Handler) http.Handler {
	port := strconv.Itoa(s.port)
	allowed := map[string]bool{"127.0.0.1:" + port: true, "localhost:" + port: true}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowed[r.Host] {
			http.Error(w, "forbidden: unexpected Host header", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cookieName() string { return "sopsy_" + strconv.Itoa(s.port) }

func equal(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

// authenticate exchanges the URL token for a session cookie (and redirects
// to / so the token leaves the address bar), or requires that cookie.
func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t := r.URL.Query().Get("t"); t != "" {
			if r.Method != http.MethodGet || !equal(t, s.token) {
				http.Error(w, "forbidden: invalid token", http.StatusForbidden)
				return
			}
			// No Secure flag: the UI is plain HTTP on loopback, where a Secure
			// cookie would not be stored by every browser.
			http.SetCookie(w, &http.Cookie{ //nolint:gosec // see above
				Name:     s.cookieName(),
				Value:    s.session,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			s.touch()
			// Always /, never the request path: //host would be an open redirect.
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		c, err := r.Cookie(s.cookieName())
		if err != nil || !equal(c.Value, s.session) {
			http.Error(w, "forbidden: open the URL printed by `sopsy ui`", http.StatusForbidden)
			return
		}
		s.touch()
		next.ServeHTTP(w, r)
	})
}

// touch resets the idle timer.
func (s *Server) touch() {
	select {
	case s.activity <- struct{}{}:
	default:
	}
}

// post guards a state-changing handler: POST only, form-encoded, valid CSRF
// token. Anything else is rejected before the handler runs.
func (s *Server) post(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/x-www-form-urlencoded" {
			http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if !equal(r.PostForm.Get("csrf"), s.csrf) {
			http.Error(w, "forbidden: missing or invalid form token; reload the page", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}
