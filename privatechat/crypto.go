package privatechat

import (
	"crypto/rand"

	"golang.org/x/crypto/curve25519"
)

// generateKeyPair creates a new X25519 key pair
func generateKeyPair() (*KeyPair, error) {
	var private [32]byte
	if _, err := rand.Read(private[:]); err != nil {
		return nil, err
	}

	var public [32]byte
	curve25519.ScalarBaseMult(&public, &private)

	return &KeyPair{
		PrivateKey: private,
		PublicKey:  public,
	}, nil
}

// generateDHKeyPair creates a new X25519 key pair for double ratchet
func generateDHKeyPair() (*X25519KeyPair, error) {
	var private [32]byte
	if _, err := rand.Read(private[:]); err != nil {
		return nil, err
	}

	var public [32]byte
	curve25519.ScalarBaseMult(&public, &private)

	return &X25519KeyPair{
		privateKey: private,
		publicKey:  public,
	}, nil
}

// computeSharedSecret performs DH key exchange
func computeSharedSecret(privateKey, publicKey [32]byte) ([32]byte, error) {
	var sharedSecret [32]byte
	curve25519.ScalarMult(&sharedSecret, &privateKey, &publicKey)
	return sharedSecret, nil
}
