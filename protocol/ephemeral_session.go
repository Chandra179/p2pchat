// protocol/ephemeral_session.go
package protocol

import (
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/crypto/curve25519"
)

// SessionKeyExchange represents the key exchange message
type SessionKeyExchange struct {
	EphemeralPubKey [32]byte  `json:"ephemeral_pub_key"`
	Signature       []byte    `json:"signature"`
	Timestamp       time.Time `json:"timestamp"`
	PeerID          string    `json:"peer_id"`
}

// SessionKey holds the ephemeral session information
type SessionKey struct {
	SharedKey     []byte
	EphemeralPriv [32]byte
	EphemeralPub  [32]byte
	CreatedAt     time.Time
	PeerID        string
}

// SessionManager manages ephemeral session keys
type SessionManager struct {
	sessions map[string]*SessionKey // peerID -> SessionKey
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*SessionKey),
	}
}

// GenerateEphemeralKeyPair generates a new X25519 ephemeral key pair
func GenerateEphemeralKeyPair() ([32]byte, [32]byte, error) {
	var priv, pub [32]byte

	if _, err := rand.Read(priv[:]); err != nil {
		return priv, pub, err
	}

	curve25519.ScalarBaseMult(&pub, &priv)
	return priv, pub, nil
}

// CreateKeyExchangeMessage creates a signed key exchange message
func CreateKeyExchangeMessage(ephemeralPub [32]byte, identityPriv crypto.PrivKey, peerID string) (*SessionKeyExchange, error) {
	msg := &SessionKeyExchange{
		EphemeralPubKey: ephemeralPub,
		Timestamp:       time.Now(),
		PeerID:          peerID,
	}

	// Create signature data: ephemeral_pub_key || timestamp || peer_id
	sigData := append(ephemeralPub[:], []byte(fmt.Sprintf("%d%s", msg.Timestamp.Unix(), peerID))...)

	signature, err := identityPriv.Sign(sigData)
	if err != nil {
		return nil, fmt.Errorf("failed to sign key exchange: %w", err)
	}

	msg.Signature = signature
	return msg, nil
}

// VerifyKeyExchangeMessage verifies the signature on a key exchange message
func VerifyKeyExchangeMessage(msg *SessionKeyExchange, identityPub crypto.PubKey) error {
	// Check timestamp (reject if too old - 5 minutes)
	if time.Since(msg.Timestamp) > 5*time.Minute {
		return errors.New("key exchange message too old")
	}

	// Verify signature
	sigData := append(msg.EphemeralPubKey[:], []byte(fmt.Sprintf("%d%s", msg.Timestamp.Unix(), msg.PeerID))...)

	valid, err := identityPub.Verify(sigData, msg.Signature)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	if !valid {
		return errors.New("invalid signature on key exchange")
	}

	return nil
}

// EstablishSession performs the key exchange and establishes a session
func (sm *SessionManager) EstablishSession(
	peerID string,
	localIdentityPriv crypto.PrivKey,
	remoteIdentityPub crypto.PubKey,
	isInitiator bool,
) (*SessionKey, *SessionKeyExchange, error) {

	// Generate ephemeral key pair
	ephemeralPriv, ephemeralPub, err := GenerateEphemeralKeyPair()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ephemeral keys: %w", err)
	}

	// Create signed key exchange message
	keyExchangeMsg, err := CreateKeyExchangeMessage(ephemeralPub, localIdentityPriv, peerID)
	if err != nil {
		return nil, nil, err
	}

	session := &SessionKey{
		EphemeralPriv: ephemeralPriv,
		EphemeralPub:  ephemeralPub,
		CreatedAt:     time.Now(),
		PeerID:        peerID,
	}

	return session, keyExchangeMsg, nil
}

// CompleteSession completes the session establishment with remote ephemeral key
func (sm *SessionManager) CompleteSession(
	session *SessionKey,
	remoteKeyExchange *SessionKeyExchange,
	remoteIdentityPub crypto.PubKey,
) error {

	// Verify the remote key exchange message
	if err := VerifyKeyExchangeMessage(remoteKeyExchange, remoteIdentityPub); err != nil {
		return fmt.Errorf("key exchange verification failed: %w", err)
	}

	// Derive shared secret using X25519 ECDH
	sharedSecret, err := curve25519.X25519(session.EphemeralPriv[:], remoteKeyExchange.EphemeralPubKey[:])
	if err != nil {
		return fmt.Errorf("ECDH failed: %w", err)
	}

	// Ensure we didn't get a weak shared secret (all zeros)
	var zero [32]byte
	if subtle.ConstantTimeCompare(sharedSecret, zero[:]) == 1 {
		return errors.New("weak shared secret detected")
	}

	// Derive session key using HKDF or simple hash
	// For simplicity, we'll use the shared secret directly
	// In production, you should use HKDF with proper salt and info
	session.SharedKey = sharedSecret

	// Store the session
	sm.sessions[session.PeerID] = session

	return nil
}

// GetSession retrieves an active session for a peer
func (sm *SessionManager) GetSession(peerID string) (*SessionKey, bool) {
	session, exists := sm.sessions[peerID]
	if !exists {
		return nil, false
	}

	// Check if session is too old (expire after 1 hour)
	if time.Since(session.CreatedAt) > time.Hour {
		delete(sm.sessions, peerID)
		return nil, false
	}

	return session, true
}

// RemoveSession removes a session for a peer
func (sm *SessionManager) RemoveSession(peerID string) {
	delete(sm.sessions, peerID)
}

// ClearExpiredSessions removes expired sessions
func (sm *SessionManager) ClearExpiredSessions() {
	for peerID, session := range sm.sessions {
		if time.Since(session.CreatedAt) > time.Hour {
			delete(sm.sessions, peerID)
		}
	}
}
