package store

import (
	"context"
	"errors"
	"fmt"

	"filippo.io/age"
	sopsage "github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/keyservice"
)

// ageService is an in-process SOPS key service that only knows age and only
// the identities sopsy found itself. The default SOPS key service would load
// identities from environment variables and ~/.ssh on its own, which would be
// a silent second key lookup.
type ageService struct {
	keyservice.UnimplementedKeyServiceServer
	ids sopsage.ParsedIdentities
}

func clients(ids []age.Identity) []keyservice.KeyServiceClient {
	return []keyservice.KeyServiceClient{keyservice.NewCustomLocalClient(ageService{ids: ids})}
}

func (s ageService) Encrypt(_ context.Context, req *keyservice.EncryptRequest) (*keyservice.EncryptResponse, error) {
	k := req.GetKey().GetAgeKey()
	if k == nil {
		return nil, fmt.Errorf("unsupported key type %s", req.GetKey().String())
	}
	mk := sopsage.MasterKey{Recipient: k.GetRecipient()}
	if err := mk.Encrypt(req.GetPlaintext()); err != nil {
		return nil, err
	}
	return &keyservice.EncryptResponse{Ciphertext: mk.EncryptedDataKey()}, nil
}

func (s ageService) Decrypt(_ context.Context, req *keyservice.DecryptRequest) (*keyservice.DecryptResponse, error) {
	k := req.GetKey().GetAgeKey()
	if k == nil {
		return nil, errors.New("unsupported key type")
	}
	if len(s.ids) == 0 {
		// An empty identity list would make the SOPS age key load
		// identities from the environment.
		return nil, errors.New("no age identity loaded")
	}
	mk := sopsage.MasterKey{Recipient: k.GetRecipient(), EncryptedKey: string(req.GetCiphertext())}
	s.ids.ApplyToMasterKey(&mk)
	plaintext, err := mk.Decrypt()
	if err != nil {
		return nil, err
	}
	return &keyservice.DecryptResponse{Plaintext: plaintext}, nil
}
