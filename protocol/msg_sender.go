package protocol

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"p2p/cryptoutils"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var (
	privateMsgCache     *lru.Cache
	privateMsgCacheInit error
)

func init() {
	privateMsgCache, privateMsgCacheInit = lru.New(1024)
	sessionManager = NewSessionManager()
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
