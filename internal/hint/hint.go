// Package hint turns errors into a message and the steps that fix them, for
// the CLI and the web UI alike. Messages never contain a secret value.
package hint

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/recipients"
	"github.com/niclasedge/sopsy/internal/sopsconfig"
	"github.com/niclasedge/sopsy/internal/store"
)

// Explain returns a user-facing message for err and the steps that fix it.
func Explain(err error) (msg string, steps []string) {
	var (
		nf  *keys.NotFoundError
		nr  *store.NotRecipientError
		tam *store.TamperedError
		pe  *store.ParseError
		inv *recipients.InvalidError
		par *recipients.PartialError
		pa  *fs.PathError
	)
	switch {
	case errors.As(err, &nf) && nf.Explicit:
		return nf.Error(), []string{
			"create that file, or unset " + keys.EnvKeyFile + " to use the default locations",
		}
	case errors.As(err, &nf):
		steps := []string{"searched:"}
		for _, p := range nf.Searched {
			steps = append(steps, "  "+p)
		}
		return "no age key found", append(steps, "run `sopsy init` to create a key")
	case errors.As(err, &nr):
		return nr.Error(), []string{
			"send the output of `sopsy pubkey` to someone who can already decrypt the file;",
			"they run `sopsy recipients add " + nr.Public + "`",
		}
	case errors.As(err, &tam):
		return tam.Error(), []string{
			"restore the file from version control, e.g. `git checkout -- " + tam.Path + "`",
		}
	case errors.As(err, &pe):
		return pe.Error(), []string{
			"the file must contain only KEY=VALUE lines written by sops or sopsy",
		}
	case errors.As(err, &inv):
		return inv.Error(), []string{
			"an age public key starts with age1; get it with `sopsy pubkey` on the machine that should decrypt",
		}
	case errors.As(err, &par):
		return par.Error(), []string{par.Manual}
	case errors.Is(err, sopsconfig.ErrNotFound):
		return "no " + sopsconfig.FileName + " in this directory or any parent", []string{
			"run `sopsy init` to create one",
		}
	case errors.Is(err, store.ErrChanged):
		return err.Error() + " while sopsy was working; nothing was written", []string{
			"run the command again",
		}
	case errors.Is(err, store.ErrLastRecipient):
		return "refusing to remove the last recipient: nobody could decrypt the file any more; nothing changed", []string{
			"add the new recipient first: sopsy recipients add AGE_PUBLIC_KEY",
		}
	case errors.Is(err, store.ErrNotEncrypted):
		return err.Error(), []string{
			"sopsy only works with files encrypted by sops or sopsy",
		}
	case errors.As(err, &pa) && errors.Is(err, fs.ErrNotExist):
		return fmt.Sprintf("%s does not exist", pa.Path), []string{
			"run `sopsy init` to create it, or point --file / $SOPSY_FILE at an existing file",
		}
	default:
		return err.Error(), nil
	}
}
