//go:build windows

package runner

import (
	"os"
	"os/exec"
	"os/signal"
)

// forwardSignals keeps Ctrl+C and Ctrl+Break from ending sopsy. Windows has
// no signals to forward: the console delivers the event to every process
// attached to it, the child included, and sopsy waits for the child.
func forwardSignals(*exec.Cmd) (stop func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	return func() { signal.Stop(ch) }
}

func exitCode(state *os.ProcessState) int {
	return state.ExitCode()
}
