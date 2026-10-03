package cli

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/sopsconfig"
	"github.com/niclasedge/sopsy/internal/store"
)

// Error is a failure with a message and the steps that fix it. Messages
// never contain a secret value.
type Error struct {
	Msg  string
	Hint []string
	// Code is the exit code; 0 means ExitFail.
	Code int
}

func (e *Error) Error() string { return e.Msg }

// explain turns any error into an *Error with an actionable hint.
func explain(err error) *Error {
	var (
		e   *Error
		nf  *keys.NotFoundError
		nr  *store.NotRecipientError
		tam *store.TamperedError
		pe  *store.ParseError
		pa  *fs.PathError
	)
	switch {
	case errors.As(err, &e):
		return e
	case errors.As(err, &nf) && nf.Explicit:
		return &Error{Msg: nf.Error(), Hint: []string{
			"create that file, or unset " + keys.EnvKeyFile + " to use the default locations",
		}}
	case errors.As(err, &nf):
		hint := []string{"searched:"}
		for _, p := range nf.Searched {
			hint = append(hint, "  "+p)
		}
		hint = append(hint, "run `sopsy init` to create a key")
		return &Error{Msg: "no age key found", Hint: hint}
	case errors.As(err, &nr):
		return &Error{Msg: nr.Error(), Hint: []string{
			"send the output of `sopsy pubkey` to someone who can already decrypt the file;",
			"they run `sopsy recipients add " + nr.Public + "`",
		}}
	case errors.As(err, &tam):
		return &Error{Msg: tam.Error(), Hint: []string{
			"restore the file from version control, e.g. `git checkout -- " + tam.Path + "`",
		}}
	case errors.As(err, &pe):
		return &Error{Msg: pe.Error(), Hint: []string{
			"the file must contain only KEY=VALUE lines written by sops or sopsy",
		}}
	case errors.Is(err, sopsconfig.ErrNotFound):
		return &Error{Msg: "no " + sopsconfig.FileName + " in this directory or any parent", Hint: []string{
			"run `sopsy init` to create one",
		}}
	case errors.Is(err, store.ErrNotEncrypted):
		return &Error{Msg: err.Error(), Hint: []string{
			"sopsy only works with files encrypted by sops or sopsy",
		}}
	case errors.As(err, &pa) && errors.Is(err, fs.ErrNotExist):
		return &Error{Msg: fmt.Sprintf("%s does not exist", pa.Path), Hint: []string{
			"run `sopsy init` to create it, or point --file / $SOPSY_FILE at an existing file",
		}}
	default:
		return &Error{Msg: err.Error()}
	}
}
