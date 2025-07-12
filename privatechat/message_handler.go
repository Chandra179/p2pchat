package privatechat

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/status-im/doubleratchet"
)

// MessageHandler handles incoming secure chat streams
type MessageHandler struct {
	sessionManager *SessionManager
	messageSender  *MessageSender
	hostID         peer.ID
}

// NewMessageHandler creates a new stream handler
func NewMessageHandler(sessionManager *SessionManager, messageSender *MessageSender, hostID peer.ID) *MessageHandler {
	return &MessageHandler{
		sessionManager: sessionManager,
		messageSender:  messageSender,
		hostID:         hostID,
	}
}

// HandleStream handles incoming secure chat streams
func (sh *MessageHandler) HandleStream(s network.Stream) {
	defer s.Close()

	var msg PrivateMessage
	decoder := json.NewDecoder(s)
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("Error decoding secure message: %v\n", err)
		return
	}

	peerID := s.Conn().RemotePeer()
	log.Printf("Received '%s' message from peer %s\n", msg.Type, peerID)

	switch msg.Type {
	case "key_exchange":
		if err := sh.handleKeyExchange(peerID, msg); err != nil {
			log.Printf("Failed to handle key exchange with %s: %v\n", peerID, err)
		}
	case "encrypted_message":
		if err := sh.handleEncryptedMessage(peerID, msg); err != nil {
			log.Printf("Failed to handle encrypted message from %s: %v\n", peerID, err)
		}
	default:
		log.Printf("Unknown message type: %s\n", msg.Type)
	}
}

// handleKeyExchange processes key exchange messages, establishing a double ratchet session.
func (sh *MessageHandler) handleKeyExchange(peerID peer.ID, msg PrivateMessage) error {
	// If a session already exists, we don't need to do anything.
	if sh.sessionManager.HasSession(peerID) {
		log.Printf("Session already exists with peer %s, ignoring key exchange.", peerID)
		return nil
	}

	// Determine roles: The peer with the smaller ID is always "Bob".
	isBob := sh.hostID < peerID
	isAlice := !isBob

	// --- Step 1: Ensure long-term keys and compute shared secret ---

	ourKeyPair, exists := sh.sessionManager.GetKeyPair(peerID)
	if !exists {
		var err error
		ourKeyPair, err = sh.sessionManager.GenerateAndStoreKeyPair(peerID)
		if err != nil {
			return fmt.Errorf("failed to generate our long-term key pair: %w", err)
		}
	}

	if len(msg.PublicKey) != 32 {
		return fmt.Errorf("invalid long-term public key length: %d", len(msg.PublicKey))
	}
	var theirLongTermPublicKey [32]byte
	copy(theirLongTermPublicKey[:], msg.PublicKey)

	sharedSecret, err := computeSharedSecret(ourKeyPair.PrivateKey, theirLongTermPublicKey)
	if err != nil {
		return fmt.Errorf("failed to compute shared secret: %w", err)
	}

	// --- Step 2: Role-dependent session creation ---

	if isBob {
		// We are "Bob". We are responsible for creating the session and the ephemeral key.
		// We only act on an 'init' message.
		if string(msg.Payload) != "key_exchange_init" {
			log.Printf("As Bob, received unexpected key exchange type '%s'. Ignoring.", msg.Payload)
			return nil
		}

		// Create the session. This generates our new ephemeral DH key pair.
		ourEphemeralKeyPair, err := sh.sessionManager.CreateSession(sh.hostID, peerID, sharedSecret, [32]byte{})
		if err != nil {
			return fmt.Errorf("failed to create session as Bob: %w", err)
		}
		if ourEphemeralKeyPair == nil {
			return nil // Session already existed, caught internally.
		}

		// We MUST send our new ephemeral public key back in a response.
		log.Printf("As Bob, sending response with our ephemeral public key to %s", peerID)
		return sh.messageSender.SendKeyExchange(peerID, "key_exchange_response", ourEphemeralKeyPair.PublicKey())

	} else if isAlice {
		// We are "Alice". We only act on a 'response' message from Bob.
		if string(msg.Payload) != "key_exchange_response" {
			log.Printf("As Alice, ignoring key exchange type '%s', waiting for 'response'.", msg.Payload)
			return nil
		}

		// The response from Bob MUST contain his ephemeral public key.
		if len(msg.EphemeralPublicKey) != 32 {
			return fmt.Errorf("invalid ephemeral public key in response: length is %d", len(msg.EphemeralPublicKey))
		}
		var theirEphemeralPublicKey [32]byte
		copy(theirEphemeralPublicKey[:], msg.EphemeralPublicKey)

		// Now we have everything needed to create our session.
		log.Printf("As Alice, creating session using Bob's ephemeral public key.")
		_, err := sh.sessionManager.CreateSession(sh.hostID, peerID, sharedSecret, theirEphemeralPublicKey)
		if err != nil {
			return fmt.Errorf("failed to create session as Alice: %w", err)
		}
	}

	return nil
}

// handleEncryptedMessage processes encrypted messages
func (sh *MessageHandler) handleEncryptedMessage(peerID peer.ID, msg PrivateMessage) error {
	session, exists := sh.sessionManager.GetSession(peerID)
	if !exists {
		return fmt.Errorf("no session found for peer %s", peerID)
	}

	// Unmarshal the doubleratchet message
	var drMessage doubleratchet.Message
	if err := json.Unmarshal(msg.Payload, &drMessage); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Decrypt message using double ratchet
	plaintext, err := session.RatchetDecrypt(drMessage, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt message: %w", err)
	}

	// Handle the decrypted message
	sh.onMessageReceived(peerID, string(plaintext), msg.Timestamp)
	return nil
}

// onMessageReceived handles a successfully decrypted message
func (sh *MessageHandler) onMessageReceived(peerID peer.ID, message string, timestamp int64) {
	fmt.Printf("[%d] Secure message from %s: %s\n",
		timestamp, peerID, message)
}
