package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVersionDev(t *testing.T) {
	r := newWorld(t).run("version")
	r.wantCode(t, ExitOK)
	if r.stdout != "sopsy dev\n" {
		t.Fatalf("version printed %q", r.stdout)
	}
}

func TestVersionLine(t *testing.T) {
	tests := []struct{ version, commit, date, want string }{
		{"dev", "", "", "sopsy dev"},
		{"0.1.0", "1a2b3c4", "2026-10-04T12:00:00Z", "sopsy 0.1.0 (commit 1a2b3c4, built 2026-10-04T12:00:00Z)"},
		{"v0.1.0-rc.1", "", "", "sopsy 0.1.0-rc.1"},
	}
	for _, tt := range tests {
		if got := versionLine(tt.version, tt.commit, tt.date); got != tt.want {
			t.Errorf("versionLine(%q, %q, %q) = %q, want %q", tt.version, tt.commit, tt.date, got, tt.want)
		}
	}
}

func TestReleaseVersion(t *testing.T) {
	for v, want := range map[string]bool{
		"v0.1.0":                               true,
		"v0.1.0-rc.1":                          true,
		"(devel)":                              false,
		"v0.0.0-20261004120000-1a2b3c4d5e6f":   false,
		"v0.1.1-0.20261004120000-1a2b3c4d5e6f": false,
		"v0.1.0+dirty":                         false,
	} {
		if got := releaseVersion.MatchString(v); got != want {
			t.Errorf("releaseVersion(%q) = %v, want %v", v, got, want)
		}
	}
}

// TestVersionLdflags builds the real binary the way the release does.
func TestVersionLdflags(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	bin := filepath.Join(t.TempDir(), "sopsy")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	const pkg = "github.com/niclasedge/sopsy/internal/cli"
	ldflags := "-X " + pkg + ".version=1.2.3 -X " + pkg + ".commit=abc1234 -X " + pkg + ".date=2026-10-04T12:00:00Z"
	build := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", bin, "github.com/niclasedge/sopsy")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("go build: %v", err)
	}
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		t.Fatal(err)
	}
	if want := "sopsy 1.2.3 (commit abc1234, built 2026-10-04T12:00:00Z)\n"; string(out) != want {
		t.Fatalf("version printed %q, want %q", out, want)
	}
}
