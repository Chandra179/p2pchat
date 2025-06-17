// protocol/encrypted_chat.go
package protocol

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"p2p/cryptoutils"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// Message types
const (
	MsgTypeKeyExchange         = "key_exchange"
	MsgTypeKeyExchangeResponse = "key_exchange_response"
	MsgTypeRekey               = "rekey"
	MsgTypeRekeyResponse       = "rekey_response"
	MsgTypeEncrypted           = "encrypted"
	MsgTypeAck                 = "ack"
)

// ProtocolMessage is the wrapper for all message types
type ProtocolMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// EncryptedMessage represents a message structure for encrypted communication.
type EncryptedMessage struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Payload   []byte `json:"payload"`
	MessageID string `json:"message_id"`
}

// AckMessage represents acknowledgment
type AckMessage struct {
	Status    string `json:"status"`
	MessageID string `json:"message_id,omitempty"`
}

var (
	privateMsgCache     *lru.Cache
	privateMsgCacheInit error
	sessionManager      *SessionManager
)

func init() {
	privateMsgCache, privateMsgCacheInit = lru.New(1024)
	sessionManager = NewSessionManager()

	// Start session cleanup routine
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			sessionManager.ClearExpiredSessions()
		}
	}()
}

// HandlePrivateMessage handles incoming private messages with session management and automatic rekeying
func HandlePrivateMessage(stream network.Stream, priv crypto.PrivKey) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer().String()
	peerPub := stream.Conn().RemotePublicKey()
	if peerPub == nil {
		fmt.Println("❌ No public key from remote peer")
		return
	}

	var protocolMsg ProtocolMessage
	decoder := json.NewDecoder(bufio.NewReader(stream))
	if err := decoder.Decode(&protocolMsg); err != nil {
		fmt.Println("❌ Failed to decode protocol message:", err)
		return
	}

	switch protocolMsg.Type {
	case MsgTypeKeyExchange:
		handleKeyExchange(stream, protocolMsg.Payload, priv, peerPub, peerID)
	case MsgTypeKeyExchangeResponse:
		handleKeyExchangeResponse(stream, protocolMsg.Payload, peerPub, peerID)
	case MsgTypeRekey:
		handleRekeyRequest(stream, protocolMsg.Payload, priv, peerPub, peerID)
	case MsgTypeRekeyResponse:
		handleRekeyResponse(stream, protocolMsg.Payload, peerPub, peerID)
	case MsgTypeEncrypted:
		handleEncryptedMessage(stream, protocolMsg.Payload, peerID)
	default:
		fmt.Printf("❌ Unknown message type: %s\n", protocolMsg.Type)
	}
}

func handleKeyExchange(stream network.Stream, payload interface{}, localPriv crypto.PrivKey, remotePub crypto.PubKey, peerID string) {
	// Parse key exchange message
	payloadBytes, _ := json.Marshal(payload)
	var keyExchange SessionKeyExchange
	if err := json.Unmarshal(payloadBytes, &keyExchange); err != nil {
		fmt.Println("❌ Failed to parse key exchange:", err)
		return
	}

	// Create our own session (as responder)
	session, ourKeyExchange, err := sessionManager.EstablishSession(peerID, localPriv, remotePub, false)
	if err != nil {
		fmt.Println("❌ Failed to establish session:", err)
		return
	}

	// Complete the session with remote key
	if err := sessionManager.CompleteSession(session, &keyExchange, remotePub); err != nil {
		fmt.Println("❌ Failed to complete session:", err)
		return
	}

	fmt.Printf("✅ Session established with %s\n", peerID)

	// Send our key exchange response
	response := ProtocolMessage{
		Type:    MsgTypeKeyExchangeResponse,
		Payload: ourKeyExchange,
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(response); err != nil {
		fmt.Println("❌ Failed to send key exchange response:", err)
	}
}

func handleKeyExchangeResponse(_ network.Stream, payload interface{}, remotePub crypto.PubKey, peerID string) {
	// Parse key exchange response
	payloadBytes, _ := json.Marshal(payload)
	var keyExchange SessionKeyExchange
	if err := json.Unmarshal(payloadBytes, &keyExchange); err != nil {
		fmt.Println("❌ Failed to parse key exchange response:", err)
		return
	}

	// Get our pending session
	session, exists := sessionManager.GetSession(peerID)
	if !exists {
		fmt.Println("❌ No pending session found for peer")
		return
	}

	// Complete the session
	if err := sessionManager.CompleteSession(session, &keyExchange, remotePub); err != nil {
		fmt.Println("❌ Failed to complete session:", err)
		return
	}

	fmt.Printf("✅ Session completed with %s\n", peerID)
}

func handleRekeyRequest(stream network.Stream, payload interface{}, localPriv crypto.PrivKey, remotePub crypto.PubKey, peerID string) {
	// Parse rekey request
	payloadBytes, _ := json.Marshal(payload)
	var rekeyMsg SessionKeyExchange
	if err := json.Unmarshal(payloadBytes, &rekeyMsg); err != nil {
		fmt.Println("❌ Failed to parse rekey request:", err)
		return
	}

	fmt.Printf("🔄 Handling rekey request from %s\n", peerID)

	// Handle the rekey request
	rekeyResponse, err := sessionManager.HandleRekeyRequest(peerID, &rekeyMsg, localPriv, remotePub)
	if err != nil {
		fmt.Println("❌ Failed to handle rekey request:", err)
		return
	}

	// Send rekey response
	response := ProtocolMessage{
		Type:    MsgTypeRekeyResponse,
		Payload: rekeyResponse,
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(response); err != nil {
		fmt.Println("❌ Failed to send rekey response:", err)
		return
	}

	fmt.Printf("✅ Rekey completed with %s (responder)\n", peerID)
}

func handleRekeyResponse(_ network.Stream, payload interface{}, remotePub crypto.PubKey, peerID string) {
	// Parse rekey response
	payloadBytes, _ := json.Marshal(payload)
	var rekeyResponse SessionKeyExchange
	if err := json.Unmarshal(payloadBytes, &rekeyResponse); err != nil {
		fmt.Println("❌ Failed to parse rekey response:", err)
		return
	}

	// Complete the rekey
	if err := sessionManager.CompleteRekey(peerID, &rekeyResponse, remotePub); err != nil {
		fmt.Println("❌ Failed to complete rekey:", err)
		return
	}

	fmt.Printf("✅ Rekey completed with %s (initiator)\n", peerID)
}

func handleEncryptedMessage(stream network.Stream, payload interface{}, peerID string) {
	// Parse encrypted message
	payloadBytes, _ := json.Marshal(payload)
	var msg EncryptedMessage
	if err := json.Unmarshal(payloadBytes, &msg); err != nil {
		fmt.Println("❌ Failed to parse encrypted message:", err)
		return
	}

	// Check for duplicates
	if privateMsgCacheInit != nil {
		fmt.Println("❌ LRU cache init failed:", privateMsgCacheInit)
		return
	}
	if msg.MessageID == "" {
		fmt.Println("❌ Missing MessageID, dropping message")
		return
	}
	if _, found := privateMsgCache.Get(msg.MessageID); found {
		return // Duplicate message, silently drop
	}
	privateMsgCache.Add(msg.MessageID, struct{}{})

	// Use session (increments counter and checks for rekey needs)
	session, needsRekey, err := sessionManager.UseSession(peerID)
	if err != nil {
		fmt.Println("❌ Failed to use session:", err)
		return
	}
	if session == nil {
		fmt.Println("❌ No active session for peer, cannot decrypt")
		return
	}

	// Decrypt message
	session.mu.RLock()
	sharedKey := session.SharedKey
	session.mu.RUnlock()

	plaintext, err := cryptoutils.X25519ChaChaDecrypt(sharedKey, msg.Payload)
	if err != nil {
		fmt.Println("❌ Decrypt failed:", err)
		return
	}

	fmt.Printf("💬 [Private] %s: %s\n", msg.From, string(plaintext))

	// Send ACK
	ack := ProtocolMessage{
		Type: MsgTypeAck,
		Payload: AckMessage{
			Status:    "ok",
			MessageID: msg.MessageID,
		},
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(ack); err != nil {
		fmt.Println("❌ Failed to send ACK:", err)
	}

	// Trigger rekey if needed (asynchronously to avoid blocking message handling)
	if needsRekey {
		go func() {
			fmt.Printf("🔄 Triggering automatic rekey for %s\n", peerID)
			// Note: This would need access to the host and private key
			// In practice, you'd pass these or have a callback mechanism
		}()
	}
}

// SendPrivateMessage sends an encrypted message with session management and automatic rekeying
func SendPrivateMessage(p protocol.ID, h host.Host, priv crypto.PrivKey, peerID peer.ID, text string) error {
	peerIDStr := peerID.String()

	// Check if we have an active session and if rekey is needed
	session, needsRekey, err := sessionManager.UseSession(peerIDStr)
	if err != nil {
		return fmt.Errorf("failed to use session: %w", err)
	}

	if session == nil {
		// Need to establish session first
		fmt.Printf("🔄 Establishing session with %s...\n", peerIDStr)
		if err := establishSessionWithPeer(p, h, priv, peerID); err != nil {
			return fmt.Errorf("failed to establish session: %w", err)
		}

		// Get the newly established session
		session, _, err = sessionManager.UseSession(peerIDStr)
		if err != nil {
			return fmt.Errorf("failed to get established session: %w", err)
		}
		if session == nil {
			return fmt.Errorf("session establishment failed")
		}
	}

	// Perform rekey if needed
	if needsRekey {
		fmt.Printf("🔄 Performing automatic rekey for %s...\n", peerIDStr)
		if err := performRekey(p, h, priv, peerID); err != nil {
			fmt.Printf("⚠️ Rekey failed, continuing with current session: %v\n", err)
		}

		// Get updated session after rekey
		session, _, err = sessionManager.UseSession(peerIDStr)
		if err != nil {
			return fmt.Errorf("failed to get session after rekey: %w", err)
		}
		if session == nil {
			return fmt.Errorf("session lost after rekey")
		}
	}

	// Encrypt and send message
	return sendEncryptedMessage(p, h, session, peerID, text)
}

func performRekey(p protocol.ID, h host.Host, priv crypto.PrivKey, peerID peer.ID) error {
	operation := func() error {
		stream, err := h.NewStream(context.TODO(), peerID, p)
		if err != nil {
			return backoff.Permanent(err)
		}
		defer stream.Close()

		// Initiate rekey
		rekeyMsg, err := sessionManager.InitiateRekey(peerID.String(), priv)
		if err != nil {
			return backoff.Permanent(err)
		}

		// Send rekey request
		msg := ProtocolMessage{
			Type:    MsgTypeRekey,
			Payload: rekeyMsg,
		}

		encoder := json.NewEncoder(stream)
		if err := encoder.Encode(msg); err != nil {
			return backoff.Permanent(err)
		}

		// Wait for rekey response
		var response ProtocolMessage
		decoder := json.NewDecoder(bufio.NewReader(stream))
		if err := decoder.Decode(&response); err != nil {
			return err
		}

		if response.Type != MsgTypeRekeyResponse {
			return backoff.Permanent(fmt.Errorf("unexpected response type: %s", response.Type))
		}

		// Parse and complete rekey
		payloadBytes, _ := json.Marshal(response.Payload)
		var rekeyResponse SessionKeyExchange
		if err := json.Unmarshal(payloadBytes, &rekeyResponse); err != nil {
			return backoff.Permanent(err)
		}

		pub := h.Peerstore().PubKey(peerID)
		if pub == nil {
			return backoff.Permanent(fmt.Errorf("no pubkey for peer %s", peerID))
		}

		if err := sessionManager.CompleteRekey(peerID.String(), &rekeyResponse, pub); err != nil {
			return backoff.Permanent(err)
		}

		return nil
	}

	bo := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)
	return backoff.Retry(operation, bo)
}

func establishSessionWithPeer(p protocol.ID, h host.Host, priv crypto.PrivKey, peerID peer.ID) error {
	operation := func() error {
		stream, err := h.NewStream(context.TODO(), peerID, p)
		if err != nil {
			return backoff.Permanent(err)
		}
		defer stream.Close()

		pub := h.Peerstore().PubKey(peerID)
		if pub == nil {
			return backoff.Permanent(fmt.Errorf("no pubkey for peer %s", peerID))
		}

		// Create session (as initiator)
		session, keyExchange, err := sessionManager.EstablishSession(peerID.String(), priv, pub, true)
		if err != nil {
			return backoff.Permanent(err)
		}

		// Send key exchange
		msg := ProtocolMessage{
			Type:    MsgTypeKeyExchange,
			Payload: keyExchange,
		}

		encoder := json.NewEncoder(stream)
		if err := encoder.Encode(msg); err != nil {
			return backoff.Permanent(err)
		}

		// Wait for response
		var response ProtocolMessage
		decoder := json.NewDecoder(bufio.NewReader(stream))
		if err := decoder.Decode(&response); err != nil {
			return err
		}

		if response.Type != MsgTypeKeyExchangeResponse {
			return backoff.Permanent(fmt.Errorf("unexpected response type: %s", response.Type))
		}

		// Parse and complete session
		payloadBytes, _ := json.Marshal(response.Payload)
		var remoteKeyExchange SessionKeyExchange
		if err := json.Unmarshal(payloadBytes, &remoteKeyExchange); err != nil {
			return backoff.Permanent(err)
		}

		if err := sessionManager.CompleteSession(session, &remoteKeyExchange, pub); err != nil {
			return backoff.Permanent(err)
		}

		fmt.Printf("✅ Session established with %s\n", peerID.String())
		return nil
	}

	bo := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)
	return backoff.Retry(operation, bo)
}

func sendEncryptedMessage(p protocol.ID, h host.Host, session *SessionKey, peerID peer.ID, text string) error {
	operation := func() error {
		stream, err := h.NewStream(context.TODO(), peerID, p)
		if err != nil {
			return backoff.Permanent(err)
		}
		defer stream.Close()

		// Get current shared key (thread-safe)
		session.mu.RLock()
		sharedKey := session.SharedKey
		session.mu.RUnlock()

		// Encrypt message
		ciphertext, err := cryptoutils.X25519ChaChaEncrypt(sharedKey, []byte(text))
		if err != nil {
			return backoff.Permanent(err)
		}

		// Create encrypted message
		encMsg := EncryptedMessage{
			From:      h.ID().String(),
			To:        peerID.String(),
			Payload:   ciphertext,
			MessageID: uuid.NewString(),
		}

		msg := ProtocolMessage{
			Type:    MsgTypeEncrypted,
			Payload: encMsg,
		}

		// Send message
		encoder := json.NewEncoder(stream)
		if err := encoder.Encode(msg); err != nil {
			return backoff.Permanent(err)
		}

		// Wait for ACK
		ackCh := make(chan bool, 1)
		errCh := make(chan error, 1)

		go func() {
			var response ProtocolMessage
			decoder := json.NewDecoder(bufio.NewReader(stream))
			if err := decoder.Decode(&response); err != nil {
				errCh <- err
				return
			}

			if response.Type == MsgTypeAck {
				ackCh <- true
			} else {
				errCh <- fmt.Errorf("unexpected response type: %s", response.Type)
			}
		}()

		select {
		case <-ackCh:
			fmt.Println("✅ Message delivered")
			return nil
		case err := <-errCh:
			return err
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timeout waiting for ACK")
		}
	}

	bo := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)
	return backoff.Retry(operation, bo)
}

// GetSessionInfo returns information about active sessions (for debugging/monitoring)
func GetSessionInfo() map[string]interface{} {
	sessionManager.mu.RLock()
	defer sessionManager.mu.RUnlock()

	info := make(map[string]interface{})
	for peerID, session := range sessionManager.sessions {
		session.mu.RLock()
		sessionInfo := map[string]interface{}{
			"created_at":     session.CreatedAt,
			"last_used":      session.LastUsed,
			"message_count":  session.MessageCount,
			"rekey_sequence": session.RekeySequence,
			"is_rekeying":    session.IsRekeying,
			"age_minutes":    time.Since(session.CreatedAt).Minutes(),
			"idle_minutes":   time.Since(session.LastUsed).Minutes(),
		}
		session.mu.RUnlock()
		info[peerID] = sessionInfo
	}

	return info
}
