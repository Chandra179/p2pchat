package utils

import (
	"encoding/base64"

	"github.com/libp2p/go-libp2p/core/crypto"
)

func DecodePrivateKey(base64Key string) (crypto.PrivKey, error) {
	privBytes, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, err
	}
	privKey, err := crypto.UnmarshalPrivateKey(privBytes)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}
