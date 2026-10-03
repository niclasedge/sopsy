package web

import (
	"os/exec"
	"slices"
)

// OpenBrowser opens url in the default browser without waiting for it.
func OpenBrowser(url string) error {
	cmd := exec.Command(browserCommand[0], append(slices.Clone(browserCommand[1:]), url)...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
