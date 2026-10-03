package keys

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fakeEnv(goos, home string, vars map[string]string) Env {
	return Env{GOOS: goos, Home: home, Getenv: func(k string) string { return vars[k] }}
}

func writeKey(t *testing.T, path string) *Key {
	t.Helper()
	k, err := Generate(path)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestFindPerOS(t *testing.T) {
	tests := []struct {
		name string
		goos string
		vars func(home string) map[string]string
		// key is where the only key file lives, relative to home
		key string
	}{
		{"macos ~/.config", "darwin", nil, ".config/sops/age/keys.txt"},
		{"macos application support", "darwin", nil, "Library/Application Support/sops/age/keys.txt"},
		{"macos XDG_CONFIG_HOME", "darwin", func(h string) map[string]string {
			return map[string]string{"XDG_CONFIG_HOME": filepath.Join(h, "xdg")}
		}, "xdg/sops/age/keys.txt"},
		{"linux ~/.config", "linux", nil, ".config/sops/age/keys.txt"},
		{"linux XDG_CONFIG_HOME", "linux", func(h string) map[string]string {
			return map[string]string{"XDG_CONFIG_HOME": filepath.Join(h, "xdg")}
		}, "xdg/sops/age/keys.txt"},
		{"windows AppData", "windows", func(h string) map[string]string {
			return map[string]string{"AppData": filepath.Join(h, "AppData", "Roaming")}
		}, "AppData/Roaming/sops/age/keys.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			vars := map[string]string{}
			if tt.vars != nil {
				vars = tt.vars(home)
			}
			want := filepath.Join(home, filepath.FromSlash(tt.key))
			writeKey(t, want)
			got, err := fakeEnv(tt.goos, home, vars).Find()
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("Find() = %s, want %s", got, want)
			}
		})
	}
}

func TestExplicitKeyFileWins(t *testing.T) {
	home := t.TempDir()
	writeKey(t, filepath.Join(home, ".config", "sops", "age", "keys.txt"))
	explicit := filepath.Join(home, "elsewhere.txt")
	writeKey(t, explicit)
	got, err := fakeEnv("darwin", home, map[string]string{EnvKeyFile: explicit}).Find()
	if err != nil {
		t.Fatal(err)
	}
	if got != explicit {
		t.Fatalf("Find() = %s, want %s", got, explicit)
	}
}

func TestExplicitKeyFileMissingDoesNotFallBack(t *testing.T) {
	home := t.TempDir()
	writeKey(t, filepath.Join(home, ".config", "sops", "age", "keys.txt"))
	missing := filepath.Join(home, "missing.txt")
	_, err := fakeEnv("linux", home, map[string]string{EnvKeyFile: missing}).Find()
	var nf *NotFoundError
	if !errors.As(err, &nf) || !nf.Explicit {
		t.Fatalf("want explicit NotFoundError, got %v", err)
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("error %q does not name %s", err, missing)
	}
}

func TestNotFoundListsEverySearchedPath(t *testing.T) {
	home := t.TempDir()
	env := fakeEnv("darwin", home, nil)
	_, err := env.Find()
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("want NotFoundError, got %v", err)
	}
	if len(nf.Searched) != 2 {
		t.Fatalf("searched %v, want two locations on macOS", nf.Searched)
	}
	for _, p := range nf.Searched {
		if !strings.Contains(err.Error(), p) {
			t.Errorf("error does not name %s", p)
		}
	}
}

func TestLinuxCandidatesAreDeduplicated(t *testing.T) {
	if got := fakeEnv("linux", "/home/u", nil).Candidates(); len(got) != 1 {
		t.Fatalf("Candidates() = %v, want one entry", got)
	}
}

func TestGenerateAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sops", "age", "keys.txt")
	k := writeKey(t, path)
	if !strings.HasPrefix(k.Public, "age1") {
		t.Fatalf("public key %q", k.Public)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Fatalf("key file mode %o, want 600", mode)
		}
	}
	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Public != k.Public || !loaded.IsRecipient(k.Public) {
		t.Fatalf("loaded public key %q, want %q", loaded.Public, k.Public)
	}
	if _, err := Generate(path); err == nil {
		t.Fatal("Generate overwrote an existing key file")
	}
}

func TestLoadInvalidFileHidesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.txt")
	if err := os.WriteFile(path, []byte("AGE-SECRET-KEY-NOT-REALLY\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFile(path)
	if err == nil || strings.Contains(err.Error(), "NOT-REALLY") {
		t.Fatalf("want error without file content, got %v", err)
	}
}
