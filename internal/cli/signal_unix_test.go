//go:build unix

package cli

import (
	"syscall"
	"testing"
)

// TestRunForwardsInterrupt sends SIGINT to sopsy only. The process runs in its
// own session without a controlling terminal, so the child can only see the
// interrupt if sopsy forwards it.
func TestRunForwardsInterrupt(t *testing.T) {
	p := startSopsy(t, &syscall.SysProcAttr{Setsid: true})
	if err := p.cmd.Process.Signal(syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	p.wantInterrupted(t)
}
