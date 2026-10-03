package store

import (
	"errors"
	"fmt"
	"slices"

	"filippo.io/age"
	sopsage "github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/keys"
)

// ErrLastRecipient means a change would leave the file without any key that
// can decrypt it.
var ErrLastRecipient = errors.New("refusing to remove the last recipient")

// ValidateRecipient checks that recipient is an age public key SOPS accepts.
func ValidateRecipient(recipient string) error {
	if _, err := sopsage.MasterKeyFromRecipient(recipient); err != nil {
		return fmt.Errorf("%q is not a valid age public key", recipient)
	}
	return nil
}

// ChangeRecipients adds and removes age recipients of the file's data key,
// like `sops updatekeys`: values, their ciphertexts and the MAC stay as they
// are, only the data key is encrypted for the new recipient set. The file
// must decrypt with ids first, which also proves it is intact.
func (f *File) ChangeRecipients(ids []age.Identity, public string, add, remove []string) error {
	if len(f.tree.Metadata.KeyGroups) != 1 {
		return errors.New("files with several key groups are not supported; use `sops updatekeys`")
	}
	for _, r := range add {
		if err := ValidateRecipient(r); err != nil {
			return err
		}
	}
	// Decrypt a second copy: f must stay encrypted so it can be written back
	// unchanged apart from its metadata.
	check, err := Load(f.Path)
	if err != nil {
		return err
	}
	if check.hash != f.hash {
		return fmt.Errorf("%s: %w", f.Path, ErrChanged)
	}
	plain, err := check.Decrypt(ids, public)
	if err != nil {
		return err
	}

	group := slices.Clone(f.tree.Metadata.KeyGroups[0])
	group = slices.DeleteFunc(group, func(k keys.MasterKey) bool {
		ak, ok := k.(*sopsage.MasterKey)
		return ok && slices.Contains(remove, ak.Recipient)
	})
	if len(group) == 0 {
		return ErrLastRecipient
	}
	for _, r := range add {
		mk, err := sopsage.MasterKeyFromRecipient(r)
		if err != nil {
			return err
		}
		if err := mk.Encrypt(plain.dataKey); err != nil {
			return fmt.Errorf("encrypting the data key for %s: %w", r, err)
		}
		group = append(group, mk)
	}
	f.tree.Metadata.KeyGroups[0] = group

	data, err := dotenvStore.EmitEncryptedFile(f.tree)
	if err != nil {
		return err
	}
	return f.write(data)
}
