package web

import (
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/niclasedge/sopsy/internal/doctor"
	"github.com/niclasedge/sopsy/internal/hint"
	"github.com/niclasedge/sopsy/internal/recipients"
	"github.com/niclasedge/sopsy/internal/setup"
	"github.com/niclasedge/sopsy/internal/sopsconfig"
	"github.com/niclasedge/sopsy/internal/store"
)

//go:embed templates static
var assets embed.FS

type page struct{ tmpl *template.Template }

func loadPages() map[string]*page {
	pages := map[string]*page{}
	for _, name := range []string{"setup", "keys", "recipients", "retrieve", "confirm"} {
		t := template.Must(template.ParseFS(assets, "templates/layout.html", "templates/"+name+".html"))
		pages[name] = &page{t}
	}
	return pages
}

// flash is a message for the next page view.
type flash struct {
	Error bool
	Text  string
	Lines []string
}

// view is what every template gets.
type view struct {
	Nav   string
	CSRF  string
	File  string
	Flash *flash
	Data  any
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(assets, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("GET /setup", s.setupPage)
	mux.HandleFunc("/setup/{step}", s.post(s.setupStep))
	mux.HandleFunc("GET /keys", s.keysPage)
	mux.HandleFunc("/keys/set", s.post(s.keySet))
	mux.HandleFunc("/keys/delete", s.post(s.keyDelete))
	mux.HandleFunc("GET /recipients", s.recipientsPage)
	mux.HandleFunc("/recipients/add", s.post(s.recipientAdd))
	mux.HandleFunc("/recipients/remove", s.post(s.recipientRemove))
	mux.HandleFunc("GET /retrieve", s.retrievePage)
	return mux
}

func (s *Server) render(w http.ResponseWriter, name, nav string, data any) {
	s.mu.Lock()
	f := s.flash
	s.flash = nil
	s.mu.Unlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	v := view{Nav: nav, CSRF: s.csrf, File: s.cfg.SecretsFile, Flash: f, Data: data}
	if err := s.pages[name].tmpl.ExecuteTemplate(w, "layout", v); err != nil {
		s.log.Printf("rendering %s: %v", name, err)
	}
}

// done stores a message and redirects, so reloading the page does not
// submit the form again. Call with s.mu held.
func (s *Server) done(w http.ResponseWriter, r *http.Request, to string, f *flash) {
	s.flash = f
	http.Redirect(w, r, to, http.StatusSeeOther)
}

func failure(err error) *flash {
	msg, steps := hint.Explain(err)
	return &flash{Error: true, Text: msg, Lines: steps}
}

func (s *Server) checks() []doctor.Result {
	return doctor.Run(doctor.Input{
		Keys:        s.cfg.Keys,
		Dir:         s.cfg.Dir,
		SecretsFile: s.cfg.SecretsFile,
		LookPath:    s.cfg.LookPath,
	})
}

// home opens the setup page while any check fails, else the keys page.
func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	to := "/keys"
	if doctor.Failed(s.checks()) {
		to = "/setup"
	}
	http.Redirect(w, r, to, http.StatusSeeOther)
}

// step is one `sopsy init` step on the setup page.
type step struct {
	Name, Path, Action, Blocked string
	Done                        bool
}

func (s *Server) steps() []step {
	keyPath, keyErr := s.cfg.Keys.Find()
	confPath, confErr := sopsconfig.Find(s.cfg.Dir)
	_, fileErr := os.Stat(s.cfg.SecretsFile)
	steps := []step{
		{Name: "age key", Path: keyPath, Action: "/setup/key", Done: keyErr == nil},
		{Name: ".sops.yaml", Path: confPath, Action: "/setup/config", Done: confErr == nil},
		{Name: "secrets file", Path: s.cfg.SecretsFile, Action: "/setup/file", Done: fileErr == nil},
	}
	if !steps[0].Done {
		steps[0].Path, _ = s.cfg.Keys.DefaultPath()
		steps[1].Blocked = "needs the age key"
	}
	if !steps[1].Done {
		steps[1].Path = filepath.Join(s.cfg.Dir, sopsconfig.FileName)
		steps[2].Blocked = "needs .sops.yaml"
	}
	return steps
}

func (s *Server) setupPage(w http.ResponseWriter, _ *http.Request) {
	steps := s.steps()
	missing := slices.ContainsFunc(steps, func(st step) bool { return !st.Done })
	s.render(w, "setup", "setup", struct {
		Steps   []step
		Missing bool
		Checks  []doctor.Result
		Command string
	}{steps, missing, s.checks(), initCommand(s.cfg)})
}

func (s *Server) setupStep(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var steps []func() (setup.Result, error)
	switch r.PathValue("step") {
	case "key":
		steps = []func() (setup.Result, error){s.setupKey}
	case "config":
		steps = []func() (setup.Result, error){s.setupConfig}
	case "file":
		steps = []func() (setup.Result, error){s.setupFile}
	case "all":
		steps = []func() (setup.Result, error){s.setupKey, s.setupConfig, s.setupFile}
	default:
		http.NotFound(w, r)
		return
	}
	var lines []string
	for _, run := range steps {
		res, err := run()
		if err != nil {
			f := failure(err)
			f.Lines = append(lines, f.Lines...)
			s.done(w, r, "/setup", f)
			return
		}
		if res.Created {
			lines = append(lines, "created "+res.Path)
		} else {
			lines = append(lines, "already present: "+res.Path)
		}
	}
	s.done(w, r, "/setup", &flash{Text: "setup updated", Lines: lines})
}

func (s *Server) setupKey() (setup.Result, error) {
	_, res, err := setup.Key(s.cfg.Keys)
	return res, err
}

func (s *Server) setupConfig() (setup.Result, error) {
	k, err := s.cfg.Keys.Load()
	if err != nil {
		return setup.Result{}, err
	}
	return setup.Config(s.cfg.Dir, s.cfg.SecretsFile, k)
}

func (s *Server) setupFile() (setup.Result, error) {
	conf, err := sopsconfig.Find(s.cfg.Dir)
	if err != nil {
		return setup.Result{}, err
	}
	return setup.SecretsFile(conf, s.cfg.SecretsFile)
}

func (s *Server) keysPage(w http.ResponseWriter, _ *http.Request) {
	var data struct {
		Names []string
		Err   *flash
	}
	f, err := store.Load(s.cfg.SecretsFile)
	if err != nil {
		data.Err = failure(err)
	} else {
		data.Names = f.Names()
	}
	s.render(w, "keys", "keys", data)
}

func (s *Server) decrypt() (*store.Plain, error) {
	f, err := store.Load(s.cfg.SecretsFile)
	if err != nil {
		return nil, err
	}
	k, err := s.cfg.Keys.Load()
	if err != nil {
		return nil, err
	}
	return f.Decrypt(k.Identities, k.Public)
}

// errBadName is shown instead of the name itself: a value pasted into the
// name field by mistake must not be echoed back.
var errBadName = errors.New("invalid key name: use letters, digits and _, not starting with a digit, and not the prefix sops_; nothing was written")

func (s *Server) keySet(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, value := r.PostForm.Get("name"), r.PostForm.Get("value")
	if store.ValidateName(name) != nil {
		s.done(w, r, "/keys", failure(errBadName))
		return
	}
	if value != r.PostForm.Get("confirm") {
		s.done(w, r, "/keys", &flash{Error: true, Text: "the two values for " + name + " differ; nothing was written"})
		return
	}
	if err := store.ValidateValue(value); err != nil {
		s.done(w, r, "/keys", &flash{Error: true, Text: err.Error() + "; nothing was written"})
		return
	}
	plain, err := s.decrypt()
	if err != nil {
		s.done(w, r, "/keys", failure(err))
		return
	}
	existed, err := plain.Set(name, value)
	if err == nil {
		err = plain.Save()
	}
	if err != nil {
		s.done(w, r, "/keys", failure(err))
		return
	}
	text := "added " + name
	if existed {
		text = "replaced " + name
	}
	s.done(w, r, "/keys", &flash{Text: text})
}

// confirmation is the page shown before a destructive action.
type confirmation struct {
	Title, Action, Field, Value, Back string
	Warning                           []string
}

func (s *Server) keyDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PostForm.Get("name")
	if store.ValidateName(name) != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.done(w, r, "/keys", failure(errBadName))
		return
	}
	if r.PostForm.Get("confirm") != "yes" {
		s.render(w, "confirm", "keys", confirmation{
			Title: "Delete " + name + "?", Action: "/keys/delete", Field: "name", Value: name, Back: "/keys",
			Warning: []string{"The value is removed from " + s.cfg.SecretsFile + ". Anything that reads " + name + " stops working."},
		})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	plain, err := s.decrypt()
	if err != nil {
		s.done(w, r, "/keys", failure(err))
		return
	}
	if !plain.Unset(name) {
		s.done(w, r, "/keys", &flash{Error: true, Text: name + " is not in the file; nothing changed"})
		return
	}
	if err := plain.Save(); err != nil {
		s.done(w, r, "/keys", failure(err))
		return
	}
	s.done(w, r, "/keys", &flash{Text: "deleted " + name})
}

type recipient struct {
	Key  string
	Mine bool
}

func (s *Server) recipientsPage(w http.ResponseWriter, _ *http.Request) {
	var data struct {
		Own        string
		Recipients []recipient
		Err        *flash
	}
	k, kerr := s.cfg.Keys.Load()
	if kerr == nil {
		data.Own = k.Public
	}
	f, err := store.Load(s.cfg.SecretsFile)
	if err != nil {
		data.Err = failure(err)
	} else {
		for _, r := range f.Recipients() {
			data.Recipients = append(data.Recipients, recipient{r, k != nil && k.IsRecipient(r)})
		}
	}
	s.render(w, "recipients", "recipients", data)
}

// errBadRecipient replaces the store's message, which quotes the input: a
// private key pasted by mistake must not be echoed back.
var errBadRecipient = errors.New("not a valid age public key: it starts with age1; get it with `sopsy pubkey` " +
	"on the machine that should decrypt; nothing changed")

func (s *Server) recipientAdd(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := r.PostForm.Get("recipient")
	if store.ValidateRecipient(key) != nil {
		s.done(w, r, "/recipients", failure(errBadRecipient))
		return
	}
	o, err := recipients.Add(s.cfg.Keys, s.cfg.Dir, s.cfg.SecretsFile, key)
	switch {
	case err != nil:
		s.done(w, r, "/recipients", failure(err))
	case !o.Changed:
		s.done(w, r, "/recipients", &flash{Text: o.Recipient + " is already a recipient; nothing changed"})
	default:
		s.done(w, r, "/recipients", &flash{Text: "added " + o.Recipient + "; " + o.File + " and " + o.Config + " updated"})
	}
}

func (s *Server) recipientRemove(w http.ResponseWriter, r *http.Request) {
	key := r.PostForm.Get("recipient")
	if store.ValidateRecipient(key) != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.done(w, r, "/recipients", failure(errBadRecipient))
		return
	}
	if r.PostForm.Get("confirm") != "yes" {
		s.render(w, "confirm", "recipients", confirmation{
			Title: "Remove this recipient?", Action: "/recipients/remove", Field: "recipient", Value: key, Back: "/recipients",
			Warning: []string{
				key + " will no longer be able to decrypt new versions of the file.",
				"It could decrypt every value until now, and old versions stay readable in version control: rotate every value afterwards.",
			},
		})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := recipients.Remove(s.cfg.Keys, s.cfg.Dir, s.cfg.SecretsFile, key)
	switch {
	case err != nil:
		s.done(w, r, "/recipients", failure(err))
	case !o.Changed:
		s.done(w, r, "/recipients", &flash{Text: o.Recipient + " is not a recipient; nothing changed"})
	default:
		lines := []string{}
		if o.AnchorKept {
			lines = append(lines, "note: its anchor definition is still in "+o.Config+"; delete it by hand if nothing else uses it")
		}
		lines = append(lines, "WARNING: rotate every value. The removed key could decrypt them until now, "+
			"and the old file stays readable in version control history. Keys to rotate:")
		lines = append(lines, o.Names...)
		s.done(w, r, "/recipients", &flash{Text: "removed " + o.Recipient + "; " + o.File + " and " + o.Config + " updated", Lines: lines})
	}
}

func (s *Server) retrievePage(w http.ResponseWriter, _ *http.Request) {
	var names []string
	if f, err := store.Load(s.cfg.SecretsFile); err == nil {
		names = f.Names()
	}
	keyFile, _ := s.cfg.Keys.Find()
	s.render(w, "retrieve", "retrieve", examples(exampleInput{
		Dir: s.cfg.Dir, File: s.cfg.SecretsFile, Executable: s.cfg.Executable, KeyFile: keyFile, Names: names,
	}))
}
