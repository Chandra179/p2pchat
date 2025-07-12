package privatechat

import (
	"crypto/rand"

	"golang.org/x/crypto/curve25519"
)

// generateKeyPair creates a new X25519 key pair (with clamping)
func generateKeyPair() (*KeyPair, error) {
	priv := make([]byte, 32)
	if _, err := rand.Read(priv); err != nil {
		return nil, err
	}
	// curve25519.Basepoint is the standard little‑endian basepoint
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		return nil, err
	}
	var pubArr [32]byte
	copy(pubArr[:], pub)
	var privArr [32]byte
	copy(privArr[:], priv)
	return &KeyPair{
		PrivateKey: privArr,
		PublicKey:  pubArr,
	}, nil
}

// computeSharedSecret uses X25519 (with clamping) to derive the same 32‑byte secret
func computeSharedSecret(privateKey, publicKey [32]byte) ([32]byte, error) {
	shared, err := curve25519.X25519(privateKey[:], publicKey[:])
	if err != nil {
		return [32]byte{}, err
	}
	var secret [32]byte
	copy(secret[:], shared)
	return secret, nil
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
