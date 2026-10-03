//go:build unix

package runner

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// forwardSignals relays SIGINT, SIGTERM and SIGHUP to the child until stop is
// called. sopsy itself never dies from them: it waits for the child.
func forwardSignals(cmd *exec.Cmd) (stop func()) {
	ch := make(chan os.Signal, 4)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case sig := <-ch:
				if cmd.Process == nil {
					continue
				}
				// Ctrl+C in a terminal already reached the child, which
				// shares sopsy's foreground process group. Forwarding it
				// again would deliver it twice, and many programs treat a
				// second interrupt as "force quit".
				if sig == syscall.SIGINT && interruptFromTerminal() {
					continue
				}
				_ = cmd.Process.Signal(sig)
			case <-done:
				return
			}
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
	}
}

// interruptFromTerminal reports whether sopsy is in the foreground process
// group of its controlling terminal, where the kernel sends Ctrl+C to the
// whole group.
func interruptFromTerminal() bool {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return false
	}
	defer func() { _ = tty.Close() }()
	pgrp, err := unix.IoctlGetInt(int(tty.Fd()), unix.TIOCGPGRP) //nolint:gosec // file descriptors fit in an int
	return err == nil && pgrp == syscall.Getpgrp()
}

func exitCode(state *os.ProcessState) int {
	if ws, ok := state.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal())
	}
	return state.ExitCode()
}
