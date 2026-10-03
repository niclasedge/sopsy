//go:build windows

package cli

import (
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
)

// TestRunCtrlBreakReachesChild sends Ctrl+Break to a new console process
// group holding sopsy and its child, the way a console delivers Ctrl+C.
// sopsy must survive it and exit with the child's code.
func TestRunCtrlBreakReachesChild(t *testing.T) {
	p := startSopsy(t, &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP})
	if err := windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(p.cmd.Process.Pid)); err != nil {
		t.Fatal(err)
	}
	p.wantInterrupted(t)
}
