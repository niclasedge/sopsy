// Package testutil holds fixtures and assertions shared by sopsy's tests.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// PublicKey is the recipient of the committed test-only key.
const PublicKey = "age1rmd3my2egua39knxm580y69zldjdarw8cfdzezaw2h799u59pveqzxqxa8"

// Secret values used by tests. AssertNoSecret fails when any of them shows up
// in output, so every test value that stands for a secret belongs here.
var Secrets = []string{
	"fixture-token-value-1", // API_TOKEN in sops-cli.env
	"fixture-db-value-2",    // DB_PASSWORD in sops-cli.env
	"s3cr3t-test-value",
	"second-s3cr3t-test-value",
}

// FixtureValues are the decrypted contents of sops-cli.env.
var FixtureValues = map[string]string{
	"API_TOKEN":   "fixture-token-value-1",
	"DB_PASSWORD": "fixture-db-value-2",
}

func dir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

// KeyFile returns the path of the committed test-only age key.
func KeyFile() string { return filepath.Join(dir(), "test-only.agekey") }

// CopyFixture copies sops-cli.env, a file encrypted by the sops CLI to the
// test-only key, to dst.
func CopyFixture(t *testing.T, dst string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir(), "sops-cli.env"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// AssertNoSecret fails the test if any known secret value appears in any of
// the outputs.
func AssertNoSecret(t *testing.T, outputs ...string) {
	t.Helper()
	for _, out := range outputs {
		for _, s := range Secrets {
			if strings.Contains(out, s) {
				t.Errorf("output contains secret test value %q:\n%s", s, out)
			}
		}
	}
}

// Sops returns the path of the sops CLI. Without it the test is skipped,
// unless SOPSY_REQUIRE_SOPS=1 (set in CI), where it fails.
func Sops(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("sops")
	if err != nil {
		if os.Getenv("SOPSY_REQUIRE_SOPS") == "1" {
			t.Fatal("sops CLI not found but SOPSY_REQUIRE_SOPS=1")
		}
		t.Skip("sops CLI not installed")
	}
	return p
}

// SopsDecrypt decrypts path with the real sops CLI and keyFile and returns
// the plaintext dotenv content.
func SopsDecrypt(t *testing.T, path, keyFile string) string {
	t.Helper()
	cmd := exec.Command(Sops(t), "decrypt", "--input-type", "dotenv", "--output-type", "dotenv", path)
	cmd.Env = append(cleanEnv(), "SOPS_AGE_KEY_FILE="+keyFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sops decrypt %s: %v\n%s", path, err, out)
	}
	return string(out)
}

// SopsEncrypt encrypts plaintext dotenv content to recipient with the real
// sops CLI and writes the result to path.
func SopsEncrypt(t *testing.T, path, recipient, plaintext string) {
	t.Helper()
	in := path + ".plain"
	if err := os.WriteFile(in, []byte(plaintext), 0o600); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(in) }()
	cmd := exec.Command(Sops(t), "encrypt", "--age", recipient,
		"--input-type", "dotenv", "--output-type", "dotenv", "--output", path, in)
	cmd.Env = cleanEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sops encrypt: %v\n%s", err, out)
	}
}

// cleanEnv drops variables that would make sops use a developer's real keys
// or config.
func cleanEnv() []string {
	var out []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "SOPS_") {
			continue
		}
		out = append(out, kv)
	}
	return out
}
