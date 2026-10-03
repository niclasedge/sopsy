package cli

import (
	"strings"
	"testing"
)

func TestDoctorHealthy(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	w.useFixture()
	r := w.run("doctor")
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, "ok       age key", "ok       decrypt          2 keys")
	if strings.Contains(r.stdout, "fail") {
		t.Fatalf("healthy setup reports a failure:\n%s", r.stdout)
	}
}

func TestDoctorNotARecipientExitsOne(t *testing.T) {
	w := newWorld(t)
	w.useFixture()
	w.run("init").wantCode(t, ExitOK) // fresh key, not a recipient of the fixture
	r := w.run("doctor")
	r.wantCode(t, ExitCheckFailed)
	wantContains(t, r.stdout, "fail     recipient", "fix: send the output of `sopsy pubkey`", "skipped  decrypt")
}

func TestDoctorFreshDirectory(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	r := w.run("doctor")
	r.wantCode(t, ExitCheckFailed)
	wantContains(t, r.stdout, "fail     .sops.yaml", "fix: sopsy init", "skipped  recipient", "skipped  decrypt")
}
