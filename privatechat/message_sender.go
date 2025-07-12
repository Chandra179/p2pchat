package privatechat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// MessageSender handles sending secure messages
type MessageSender struct {
	host           host.Host
	sessionManager *SessionManager
}

// NewMessageSender creates a new message sender
func NewMessageSender(host host.Host, sessionManager *SessionManager) *MessageSender {
	return &MessageSender{
		host:           host,
		sessionManager: sessionManager,
	}
}

// InitiateKeyExchange starts a secure chat session with a peer
func (ms *MessageSender) InitiateKeyExchange(peerID peer.ID) error {
	if _, err := ms.sessionManager.GenerateAndStoreKeyPair(peerID); err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}
	return ms.SendKeyExchange(peerID, "key_exchange_init")
}

// SendKeyExchange sends a key exchange message
func (ms *MessageSender) SendKeyExchange(peerID peer.ID, exchangeType string) error {
	keyPair, exists := ms.sessionManager.GetKeyPair(peerID)
	if !exists {
		return fmt.Errorf("no key pair found for peer %s", peerID)
	}

	msg := PrivateMessage{
		Type:      "key_exchange",
		PublicKey: keyPair.PublicKey[:],
		Payload:   []byte(exchangeType),
		Timestamp: time.Now().Unix(),
	}

	return ms.sendMessage(peerID, msg)
}

// SendEncryptedMessage sends an encrypted message to a peer
func (ms *MessageSender) SendEncryptedMessage(peerID peer.ID, message string) error {
	session, exists := ms.sessionManager.GetSession(peerID)
	if !exists {
		return fmt.Errorf("no secure session with peer %s. Initialize chat first", peerID)
	}
	drMessage, err := session.RatchetEncrypt([]byte(message), nil)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}
	payload, err := json.Marshal(drMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	msg := PrivateMessage{
		Type:      "encrypted_message",
		Payload:   payload,
		Timestamp: time.Now().Unix(),
	}

	return ms.sendMessage(peerID, msg)
}

// sendMessage sends a secure message to a peer over libp2p stream
func (ms *MessageSender) sendMessage(peerID peer.ID, msg PrivateMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := ms.host.NewStream(
		network.WithAllowLimitedConn(ctx, "private-chat"),
		peerID,
		"/secure-chat/1.0.0",
	)
	if err != nil {
		return fmt.Errorf("failed to create stream to peer %s: %w", peerID, err)
	}
	defer stream.Close()

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	return nil
}

// BroadcastMessage sends a message to all peers with active sessions
func (ms *MessageSender) BroadcastMessage(message string) error {
	activePeers := ms.sessionManager.ListActiveSessions()
	if len(activePeers) == 0 {
		return fmt.Errorf("no active sessions to broadcast to")
	}

	var errors []error
	for _, peerID := range activePeers {
		if err := ms.SendEncryptedMessage(peerID, message); err != nil {
			errors = append(errors, fmt.Errorf("failed to send to %s: %w", peerID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("broadcast failed for some peers: %v", errors)
	}

	return nil
}

// SendMessageWithRetry sends a message with retry logic
func (ms *MessageSender) SendMessageWithRetry(peerID peer.ID, message string, maxRetries int) error {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		if err := ms.SendEncryptedMessage(peerID, message); err == nil {
			return nil // Success
		} else {
			lastErr = err
			if i < maxRetries {
				// Wait before retry (exponential backoff)
				time.Sleep(time.Duration(1<<i) * time.Second)
			}
		}
	}

	return fmt.Errorf("failed to send message after %d retries: %w", maxRetries, lastErr)
}
