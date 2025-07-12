package privatechat

import (
	"fmt"
	"log"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/status-im/doubleratchet"
)

// SessionManager handles secure session lifecycle
type SessionManager struct {
	sessions map[peer.ID]doubleratchet.Session
	keyPairs map[peer.ID]*KeyPair
	storage  doubleratchet.SessionStorage
	mu       sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[peer.ID]doubleratchet.Session),
		keyPairs: make(map[peer.ID]*KeyPair),
		storage:  NewInMemorySessionStorage(),
	}
}

// CreateSession creates a new secure session with a peer
func (sm *SessionManager) CreateSession(hostID, peerID peer.ID, sharedSecret [32]byte, theirPublicKey [32]byte, isInitiator bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Create session ID using both peer IDs
	sessionID := fmt.Sprintf("%s-%s", hostID, peerID)
	var session doubleratchet.Session
	var err error

	if isInitiator {
		// We are the initiator - create session with our key pair
		dhKeyPair, errGenKey := generateDHKeyPair()
		if errGenKey != nil {
			return fmt.Errorf("failed to generate DH key pair: %w", err)
		}

		session, err = doubleratchet.New(
			[]byte(sessionID),
			doubleratchet.Key(sharedSecret[:]),
			dhKeyPair,
			sm.storage,
		)
	} else {
		// We are the responder - create session with their public key
		session, err = doubleratchet.NewWithRemoteKey(
			[]byte(sessionID),
			doubleratchet.Key(sharedSecret[:]),
			doubleratchet.Key(theirPublicKey[:]),
			sm.storage,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to create double ratchet session: %w", err)
	}

	sm.sessions[peerID] = session
	log.Printf("Secure session established with peer %s", peerID)
	return nil
}

// GetSession returns a session for a peer (thread-safe)
func (sm *SessionManager) GetSession(peerID peer.ID) (doubleratchet.Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[peerID]
	return session, exists
}

// HasSession checks if a secure session exists with a peer
func (sm *SessionManager) HasSession(peerID peer.ID) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	_, exists := sm.sessions[peerID]
	return exists
}

// CloseSession closes a secure session with a peer
func (sm *SessionManager) CloseSession(peerID peer.ID) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, peerID)
	delete(sm.keyPairs, peerID)
	log.Printf("Closed secure session with peer %s", peerID)
}

// GetKeyPair returns a key pair for a peer (thread-safe)
func (sm *SessionManager) GetKeyPair(peerID peer.ID) (*KeyPair, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	keyPair, exists := sm.keyPairs[peerID]
	return keyPair, exists
}

// SetKeyPair stores a key pair for a peer (thread-safe)
func (sm *SessionManager) SetKeyPair(peerID peer.ID, keyPair *KeyPair) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.keyPairs[peerID] = keyPair
}

// GenerateAndStoreKeyPair generates a new key pair and stores it for a peer
func (sm *SessionManager) GenerateAndStoreKeyPair(peerID peer.ID) (*KeyPair, error) {
	keyPair, err := generateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	sm.SetKeyPair(peerID, keyPair)
	return keyPair, nil
}

// ListActiveSessions returns a list of peer IDs with active sessions
func (sm *SessionManager) ListActiveSessions() []peer.ID {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	peers := make([]peer.ID, 0, len(sm.sessions))
	for peerID := range sm.sessions {
		peers = append(peers, peerID)
	}
	return peers
}
