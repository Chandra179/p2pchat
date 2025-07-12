package privatechat

import (
	"github.com/status-im/doubleratchet"
)

// PrivateMessage represents an encrypted message with metadata
type PrivateMessage struct {
	Type      string `json:"type"`                 // "key_exchange" or "encrypted_message"
	PublicKey []byte `json:"public_key,omitempty"` // For key exchange
	Payload   []byte `json:"payload"`              // Encrypted message or plaintext for key exchange
	Timestamp int64  `json:"timestamp"`
}

// KeyPair holds DH key pair for a peer
type KeyPair struct {
	PrivateKey [32]byte
	PublicKey  [32]byte
}

// X25519KeyPair implements doubleratchet.DHPair
type X25519KeyPair struct {
	privateKey [32]byte
	publicKey  [32]byte
}

// PrivateKey returns the private key bytes
func (kp *X25519KeyPair) PrivateKey() doubleratchet.Key {
	return doubleratchet.Key(kp.privateKey[:])
}

// PublicKey returns the public key bytes
func (kp *X25519KeyPair) PublicKey() doubleratchet.Key {
	return doubleratchet.Key(kp.publicKey[:])
}
