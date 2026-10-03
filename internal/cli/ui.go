package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/niclasedge/sopsy/internal/web"
)

func runUI(env Env, args []string) error {
	fs := flags(env, "ui")
	file := fileFlag(fs)
	port := fs.Int("port", 0, "port on 127.0.0.1 (default: a free one)")
	idle := fs.Duration("idle", web.DefaultIdle, "stop after this long without requests")
	if err := parse(fs, args); err != nil {
		return err
	}
	usage := []string{"usage: sopsy ui [--file F] [--port N] [--idle 15m]"}
	switch {
	case fs.NArg() > 0:
		return &Error{Msg: "ui takes no arguments", Hint: usage}
	case *port < 0 || *port > 65535:
		return &Error{Msg: "--port must be between 0 and 65535", Hint: usage}
	case *idle <= 0:
		return &Error{Msg: "--idle must be a positive duration such as 15m", Hint: usage}
	}

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(*port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return &Error{Msg: fmt.Sprintf("cannot listen on %s: %v", addr, err), Hint: []string{
			"choose another port with --port, or leave it out to use a free one",
		}}
	}
	exe, _ := os.Executable()
	srv, err := web.New(web.Config{
		Keys:        env.Keys,
		Dir:         env.Dir,
		SecretsFile: env.secretsFile(*file),
		Executable:  exe,
		Idle:        *idle,
		Log:         env.Stderr,
	}, ln)
	if err != nil {
		_ = ln.Close()
		return err
	}

	_, _ = fmt.Fprintf(env.Stdout, "sopsy ui running at\n  %s\n"+
		"Only this machine can connect, and the link works once. Ctrl+C stops it;\n"+
		"it also stops after %s without requests.\n", srv.URL(), *idle)
	open := env.OpenBrowser
	if open == nil {
		open = web.OpenBrowser
	}
	if err := open(srv.URL()); err != nil {
		_, _ = fmt.Fprintf(env.Stderr, "sopsy ui: could not open a browser (%v); open the link above\n", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err = srv.Serve(ctx)
	if errors.Is(err, web.ErrIdle) {
		_, err = fmt.Fprintf(env.Stdout, "sopsy ui stopped: no requests for %s\n", *idle)
		return err
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(env.Stdout, "sopsy ui stopped")
	return err
}
