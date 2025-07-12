package privatechat

import (
	"crypto/rand"

	"golang.org/x/crypto/curve25519"
)

// generateKeyPair creates a new X25519 key pair (with proper clamping)
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

// generateDHKeyPair creates the ratchet’s ephemeral X25519 key pair (also clamped)
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

// computeSharedSecret derives the shared 32‑byte secret (clamped on both sides)
func computeSharedSecret(privateKey, publicKey [32]byte) ([32]byte, error) {
	shared, err := curve25519.X25519(privateKey[:], publicKey[:])
	if err != nil {
		return [32]byte{}, err
	}
	var secret [32]byte
	copy(secret[:], shared)
	return secret, nil
}
