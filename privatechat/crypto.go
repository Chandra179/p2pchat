package privatechat

import (
	"crypto/rand"

	"golang.org/x/crypto/curve25519"
)

// generateKeyPair (for initial key-exchange) with automatic clamping
func generateKeyPair() (*KeyPair, error) {
	priv := make([]byte, 32)
	if _, err := rand.Read(priv); err != nil {
		return nil, err
	}
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		return nil, err
	}

	var privArr, pubArr [32]byte
	copy(privArr[:], priv)
	copy(pubArr[:], pub)
	return &KeyPair{PrivateKey: privArr, PublicKey: pubArr}, nil
}

// generateDHKeyPair (for the ratchet’s ephemeral DH) — **also** use X25519!
func generateDHKeyPair() (*X25519KeyPair, error) {
	priv := make([]byte, 32)
	if _, err := rand.Read(priv); err != nil {
		return nil, err
	}
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		return nil, err
	}

	var privArr, pubArr [32]byte
	copy(privArr[:], priv)
	copy(pubArr[:], pub)
	return &X25519KeyPair{privateKey: privArr, publicKey: pubArr}, nil
}

// computeSharedSecret — again X25519 for BOTH peers
func computeSharedSecret(privateKey, publicKey [32]byte) ([32]byte, error) {
	shared, err := curve25519.X25519(privateKey[:], publicKey[:])
	if err != nil {
		return [32]byte{}, err
	}
	var secret [32]byte
	copy(secret[:], shared)
	return secret, nil
}
