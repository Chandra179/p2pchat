// cryptoutils/x25519_chacha_crypto.go
package cryptoutils

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"fmt"

	"filippo.io/edwards25519"
	"github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/crypto/chacha20poly1305"
)

// X25519ChaChaEncrypt encrypts plaintext using the shared key and ChaCha20-Poly1305.
func X25519ChaChaEncrypt(sharedKey, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(sharedKey)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil // prepend nonce to ciphertext
}

// X25519ChaChaDecrypt decrypts ciphertext using the shared key and ChaCha20-Poly1305.
func X25519ChaChaDecrypt(sharedKey, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(sharedKey)
	if err != nil {
		return nil, err
	}
	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	return aead.Open(nil, nonce, ciphertext, nil)
}

// PrivToX25519 converts a libp2p Ed25519 private key to an X25519 private key.
func PrivToX25519(priv crypto.PrivKey) ([32]byte, error) {
	var xpriv [32]byte

	// Extract raw Ed25519 private key
	raw, err := priv.Raw()
	if err != nil {
		return xpriv, err
	}
	if len(raw) != 64 {
		return xpriv, errors.New("invalid ed25519 private key length")
	}

	// First 32 bytes of raw is the private seed
	h := sha512.Sum512(raw[:32])
	h[0] &= 248
	h[31] &= 127
	h[31] |= 64
	copy(xpriv[:], h[:32]) // X25519 private scalar

	return xpriv, nil
}

// PubToX25519 converts a libp2p Ed25519 public key to an X25519 public key.
func PubToX25519(pub crypto.PubKey) ([32]byte, error) {
	var xpub [32]byte

	raw, err := pub.Raw()
	if err != nil {
		return xpub, err
	}
	if len(raw) != ed25519.PublicKeySize {
		return xpub, errors.New("invalid ed25519 pubkey length")
	}

	var A edwards25519.Point
	if _, err := A.SetBytes(raw); err != nil {
		return xpub, err
	}
	A.MultByCofactor(&A)
	copy(xpub[:], A.BytesMontgomery())

	return xpub, nil
}
