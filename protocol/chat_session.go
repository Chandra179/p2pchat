// protocol/ephemeral_session.go
package protocol

import (
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/crypto/curve25519"
)

// Rekeying thresholds
const (
	MaxMessagesBeforeRekey = 1000             // Rekey after 1000 messages
	MaxTimeBeforeRekey     = 30 * time.Minute // Rekey after 30 minutes
	RekeyGracePeriod       = 5 * time.Minute  // Grace period for rekey completion
)

// SessionKeyExchange represents the key exchange message
type SessionKeyExchange struct {
	PubKey        [32]byte  `json:"pub_key"`
	Signature     []byte    `json:"signature"`
	Timestamp     time.Time `json:"timestamp"`
	PeerID        string    `json:"peer_id"`
	IsRekey       bool      `json:"is_rekey"`       // Indicates if this is a rekey operation
	RekeySequence uint64    `json:"rekey_sequence"` // Sequence number for rekey operations
}

// SessionKey holds the ephemeral session information
type SessionKey struct {
	// Current active key
	SharedKey []byte
	PrivKey   [32]byte
	PubKey    [32]byte

	// Session metadata
	CreatedAt     time.Time
	LastUsed      time.Time
	PeerID        string
	MessageCount  uint64
	RekeySequence uint64

	// Rekeying state
	IsRekeying bool
	PendingKey *PendingRekey // New key being negotiated

	// Synchronization
	mu sync.RWMutex
}

// PendingRekey holds the state during a rekey operation
type PendingRekey struct {
	NewSharedKey  []byte
	NewPriv       [32]byte
	NewPub        [32]byte
	RekeySequence uint64
	InitiatedAt   time.Time
	IsInitiator   bool
}

// SessionManager manages ephemeral session keys with automatic rekeying
type SessionManager struct {
	sessions map[string]*SessionKey // peerID -> SessionKey
	mu       sync.RWMutex
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
func CreateKeyExchangeMessage(ephemeralPub [32]byte, identityPriv crypto.PrivKey, peerID string, isRekey bool, rekeySequence uint64) (*SessionKeyExchange, error) {
	msg := &SessionKeyExchange{
		PubKey:        ephemeralPub,
		Timestamp:     time.Now(),
		PeerID:        peerID,
		IsRekey:       isRekey,
		RekeySequence: rekeySequence,
	}

	// Create signature data: ephemeral_pub_key || timestamp || peer_id || is_rekey || rekey_sequence
	sigData := append(ephemeralPub[:], []byte(fmt.Sprintf("%d%s%t%d",
		msg.Timestamp.Unix(), peerID, isRekey, rekeySequence))...)

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
	sigData := append(msg.PubKey[:], []byte(fmt.Sprintf("%d%s%t%d",
		msg.Timestamp.Unix(), msg.PeerID, msg.IsRekey, msg.RekeySequence))...)

	valid, err := identityPub.Verify(sigData, msg.Signature)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	if !valid {
		return errors.New("invalid signature on key exchange")
	}

	return nil
}

// InitiateSession creates initial session data and key exchange message to start session establishment
func (sm *SessionManager) InitiateSession(
	peerID string,
	localIdentityPriv crypto.PrivKey,
	isInitiator bool,
) (*SessionKey, *SessionKeyExchange, error) {
	// Generate ephemeral key pair
	ephemeralPriv, ephemeralPub, err := GenerateEphemeralKeyPair()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ephemeral keys: %w", err)
	}

	// Create signed key exchange message
	keyExchangeMsg, err := CreateKeyExchangeMessage(ephemeralPub, localIdentityPriv, peerID, false, 0)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	session := &SessionKey{
		PrivKey:       ephemeralPriv,
		PubKey:        ephemeralPub,
		CreatedAt:     now,
		LastUsed:      now,
		PeerID:        peerID,
		MessageCount:  0,
		RekeySequence: 0,
		IsRekeying:    false,
	}

	return session, keyExchangeMsg, nil
}

// InitiateRekey starts a rekey operation for an existing session
func (sm *SessionManager) InitiateRekey(
	peerID string,
	localIdentityPriv crypto.PrivKey,
) (*SessionKeyExchange, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[peerID]
	if !exists {
		return nil, errors.New("no active session found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check if already rekeying
	if session.IsRekeying {
		return nil, errors.New("rekey already in progress")
	}

	// Generate new ephemeral key pair
	newEphemeralPriv, newEphemeralPub, err := GenerateEphemeralKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new ephemeral keys: %w", err)
	}

	newRekeySequence := session.RekeySequence + 1

	// Create rekey message
	rekeyMsg, err := CreateKeyExchangeMessage(newEphemeralPub, localIdentityPriv, peerID, true, newRekeySequence)
	if err != nil {
		return nil, err
	}

	// Set up pending rekey state
	session.IsRekeying = true
	session.PendingKey = &PendingRekey{
		NewPriv:       newEphemeralPriv,
		NewPub:        newEphemeralPub,
		RekeySequence: newRekeySequence,
		InitiatedAt:   time.Now(),
		IsInitiator:   true,
	}

	return rekeyMsg, nil
}

// HandleRekeyRequest handles an incoming rekey request
func (sm *SessionManager) HandleRekeyRequest(
	peerID string,
	rekeyMsg *SessionKeyExchange,
	localIdentityPriv crypto.PrivKey,
	remoteIdentityPub crypto.PubKey,
) (*SessionKeyExchange, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[peerID]
	if !exists {
		return nil, errors.New("no active session found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Verify the rekey message
	if err := VerifyKeyExchangeMessage(rekeyMsg, remoteIdentityPub); err != nil {
		return nil, fmt.Errorf("rekey verification failed: %w", err)
	}

	// Check sequence number (should be greater than current)
	if rekeyMsg.RekeySequence <= session.RekeySequence {
		return nil, errors.New("invalid rekey sequence number")
	}

	// Generate our response ephemeral key pair
	newEphemeralPriv, newEphemeralPub, err := GenerateEphemeralKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate response ephemeral keys: %w", err)
	}

	// Create rekey response
	rekeyResponse, err := CreateKeyExchangeMessage(newEphemeralPub, localIdentityPriv, peerID, true, rekeyMsg.RekeySequence)
	if err != nil {
		return nil, err
	}

	// Derive new shared secret
	newSharedSecret, err := curve25519.X25519(newEphemeralPriv[:], rekeyMsg.PubKey[:])
	if err != nil {
		return nil, fmt.Errorf("ECDH failed during rekey: %w", err)
	}

	// Ensure we didn't get a weak shared secret
	var zero [32]byte
	if subtle.ConstantTimeCompare(newSharedSecret, zero[:]) == 1 {
		return nil, errors.New("weak shared secret detected during rekey")
	}

	// Set up pending rekey state
	session.IsRekeying = true
	session.PendingKey = &PendingRekey{
		NewSharedKey:  newSharedSecret,
		NewPriv:       newEphemeralPriv,
		NewPub:        newEphemeralPub,
		RekeySequence: rekeyMsg.RekeySequence,
		InitiatedAt:   time.Now(),
		IsInitiator:   false,
	}

	return rekeyResponse, nil
}

// CompleteRekey finalizes a rekey operation
func (sm *SessionManager) CompleteRekey(
	peerID string,
	rekeyResponse *SessionKeyExchange,
	remoteIdentityPub crypto.PubKey,
) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[peerID]
	if !exists {
		return errors.New("no active session found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if !session.IsRekeying || session.PendingKey == nil {
		return errors.New("no rekey in progress")
	}

	// Verify the rekey response
	if err := VerifyKeyExchangeMessage(rekeyResponse, remoteIdentityPub); err != nil {
		return fmt.Errorf("rekey response verification failed: %w", err)
	}

	// Check sequence number matches
	if rekeyResponse.RekeySequence != session.PendingKey.RekeySequence {
		return errors.New("rekey sequence mismatch")
	}

	var newSharedSecret []byte
	var err error

	if session.PendingKey.IsInitiator {
		// We initiated, so derive shared secret with their response
		newSharedSecret, err = curve25519.X25519(session.PendingKey.NewPriv[:], rekeyResponse.PubKey[:])
		if err != nil {
			return fmt.Errorf("ECDH failed during rekey completion: %w", err)
		}
	} else {
		// We responded, so use the already computed shared secret
		newSharedSecret = session.PendingKey.NewSharedKey
	}

	// Ensure we didn't get a weak shared secret
	var zero [32]byte
	if subtle.ConstantTimeCompare(newSharedSecret, zero[:]) == 1 {
		return errors.New("weak shared secret detected during rekey completion")
	}

	// Atomically switch to new key
	session.SharedKey = newSharedSecret
	session.PrivKey = session.PendingKey.NewPriv
	session.PubKey = session.PendingKey.NewPub
	session.RekeySequence = session.PendingKey.RekeySequence
	session.MessageCount = 0 // Reset message counter
	session.LastUsed = time.Now()

	// Clear rekey state
	session.IsRekeying = false
	session.PendingKey = nil

	return nil
}

// CompleteSession completes the initial session establishment
func (sm *SessionManager) CompleteSession(
	session *SessionKey,
	remoteKeyExchange *SessionKeyExchange,
	remoteIdentityPub crypto.PubKey,
) error {
	// Verify the remote key exchange message
	if err := VerifyKeyExchangeMessage(remoteKeyExchange, remoteIdentityPub); err != nil {
		return fmt.Errorf("key exchange verification failed: %w", err)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Derive shared secret using X25519 ECDH
	sharedSecret, err := curve25519.X25519(session.PrivKey[:], remoteKeyExchange.PubKey[:])
	if err != nil {
		return fmt.Errorf("ECDH failed: %w", err)
	}

	// Ensure we didn't get a weak shared secret (all zeros)
	var zero [32]byte
	if subtle.ConstantTimeCompare(sharedSecret, zero[:]) == 1 {
		return errors.New("weak shared secret detected")
	}

	session.SharedKey = sharedSecret
	session.LastUsed = time.Now()

	// Store the session
	sm.mu.Lock()
	sm.sessions[session.PeerID] = session
	sm.mu.Unlock()

	return nil
}

// GetSession retrieves an active session for a peer
func (sm *SessionManager) GetSession(peerID string) (*SessionKey, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[peerID]
	if !exists {
		return nil, false
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	// Check if session is too old (expire after 1 hour)
	if time.Since(session.CreatedAt) > time.Hour {
		delete(sm.sessions, peerID)
		return nil, false
	}

	return session, true
}

// UseSession increments message counter and checks if rekey is needed
func (sm *SessionManager) UseSession(peerID string) (*SessionKey, bool, error) {
	sm.mu.RLock()
	session, exists := sm.sessions[peerID]
	sm.mu.RUnlock()

	if !exists {
		return nil, false, nil
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Increment message counter
	session.MessageCount++
	session.LastUsed = time.Now()

	// Check if rekey is needed
	needsRekey := sm.needsRekey(session)

	return session, needsRekey, nil
}

// needsRekey checks if a session needs rekeying (must be called with session lock held)
func (sm *SessionManager) needsRekey(session *SessionKey) bool {
	// Don't rekey if already rekeying
	if session.IsRekeying {
		return false
	}

	// Check message count threshold
	if session.MessageCount >= MaxMessagesBeforeRekey {
		return true
	}

	// Check time threshold
	if time.Since(session.CreatedAt) >= MaxTimeBeforeRekey {
		return true
	}

	return false
}

// RemoveSession removes a session for a peer
func (sm *SessionManager) RemoveSession(peerID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, peerID)
}

// ClearExpiredSessions removes expired sessions and failed rekeys
func (sm *SessionManager) ClearExpiredSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for peerID, session := range sm.sessions {
		session.mu.Lock()

		shouldRemove := false

		// Remove if session is too old
		if time.Since(session.CreatedAt) > time.Hour {
			shouldRemove = true
		}

		// Remove if rekey has been stuck for too long
		if session.IsRekeying && session.PendingKey != nil {
			if time.Since(session.PendingKey.InitiatedAt) > RekeyGracePeriod {
				// Reset rekey state instead of removing session
				session.IsRekeying = false
				session.PendingKey = nil
			}
		}

		session.mu.Unlock()

		if shouldRemove {
			delete(sm.sessions, peerID)
		}
	}
}
