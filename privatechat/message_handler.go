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
		fmt.Printf("Error decoding secure message: %v\n", err)
		return
	}

	peerID := s.Conn().RemotePeer()
	fmt.Printf("Received %s message from peer %s\n", msg.Type, peerID)

	switch msg.Type {
	case "key_exchange":
		if err := sh.handleKeyExchange(peerID, msg); err != nil {
			fmt.Printf("Failed to handle key exchange: %v\n", err)
		}
	case "encrypted_message":
		if err := sh.handleEncryptedMessage(peerID, msg); err != nil {
			fmt.Printf("Failed to handle encrypted message: %v\n", err)
		}
	default:
		fmt.Printf("Unknown message type: %s\n", msg.Type)
	}
}

// handleKeyExchange processes key exchange messages
func (sh *MessageHandler) handleKeyExchange(peerID peer.ID, msg PrivateMessage) error {
	// Skip if we already have a session
	if sh.sessionManager.HasSession(peerID) {
		log.Printf("Session already exists with peer %s, ignoring key exchange", peerID)
		return nil
	}

	// Generate our key pair if we don't have one for this peer
	if _, exists := sh.sessionManager.GetKeyPair(peerID); !exists {
		if _, err := sh.sessionManager.GenerateAndStoreKeyPair(peerID); err != nil {
			return fmt.Errorf("failed to generate key pair: %w", err)
		}
	}

	// Validate incoming public key
	if len(msg.PublicKey) != 32 {
		return fmt.Errorf("invalid public key length: %d", len(msg.PublicKey))
	}

	// Compute shared secret
	var theirPublicKey [32]byte
	copy(theirPublicKey[:], msg.PublicKey)

	ourKeyPair, _ := sh.sessionManager.GetKeyPair(peerID)
	sharedSecret, err := computeSharedSecret(ourKeyPair.PrivateKey, theirPublicKey)
	if err != nil {
		return fmt.Errorf("failed to compute shared secret: %w", err)
	}

	// Determine initiator/responder roles based on peer ID comparison
	// This ensures both peers agree on who is the initiator
	isInitiator := sh.hostID < peerID

	// Create the session
	if err := sh.sessionManager.CreateSession(sh.hostID, peerID, sharedSecret, theirPublicKey, isInitiator); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// If we received a key_exchange_init and don't have a session, send our response
	if string(msg.Payload) == "key_exchange_init" {
		if err := sh.messageSender.SendKeyExchange(peerID, "key_exchange_response"); err != nil {
			log.Printf("Failed to send key exchange response: %v", err)
			return err
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

	// Handle the decrypted message (you can customize this)
	sh.onMessageReceived(peerID, string(plaintext), msg.Timestamp)
	return nil
}

// onMessageReceived handles a successfully decrypted message
func (sh *MessageHandler) onMessageReceived(peerID peer.ID, message string, timestamp int64) {
	fmt.Printf("[%s] Secure message from %s: %s\n",
		formatTimestamp(timestamp), peerID, message)
}

// formatTimestamp formats a Unix timestamp for display
func formatTimestamp(timestamp int64) string {
	// You can customize this format as needed
	return fmt.Sprintf("%d", timestamp)
}
