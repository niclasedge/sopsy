package cli

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/niclasedge/sopsy/internal/testutil"
)

// sopsyProcess is sopsy running as a real process (the test binary in
// SOPSY_TEST_MAIN=sopsy mode) with a helper child that waits for an
// interrupt.
type sopsyProcess struct {
	cmd    *exec.Cmd
	stderr *bytes.Buffer
	marker string
}

func startSopsy(t *testing.T, attr *syscall.SysProcAttr) *sopsyProcess {
	t.Helper()
	w := runWorld(t)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	p := &sopsyProcess{stderr: &bytes.Buffer{}, marker: filepath.Join(w.home, "interrupted")}
	p.cmd = exec.Command(self, append([]string{"run", "--"}, helperCmd("trap", p.marker)...)...)
	p.cmd.Dir = w.dir
	p.cmd.Env = append(w.environ(), "SOPSY_TEST_MAIN=sopsy", "HOME="+w.home)
	p.cmd.SysProcAttr = attr
	p.cmd.Stderr = p.stderr
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := p.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.cmd.Process.Kill() })
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || line != "ready\n" {
		t.Fatalf("helper did not get ready: %q, %v\nstderr: %s", line, err, p.stderr)
	}
	return p
}

func (p *sopsyProcess) wantInterrupted(t *testing.T) {
	t.Helper()
	_ = p.cmd.Wait()
	if code := p.cmd.ProcessState.ExitCode(); code != 42 {
		t.Fatalf("sopsy exited %d, want the child's 42\nstderr: %s", code, p.stderr)
	}
	if _, err := os.Stat(p.marker); err != nil {
		t.Fatal("the child's interrupt handler did not run")
	}
	testutil.AssertNoSecret(t, p.stderr.String())
}
