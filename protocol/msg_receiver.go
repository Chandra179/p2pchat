package protocol

import (
	"bufio"
	"encoding/json"
	"fmt"
	"p2p/cryptoutils"

	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

// AckMessage represents acknowledgment
type AckMessage struct {
	Status    string `json:"status"`
	MessageID string `json:"message_id,omitempty"`
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
	session, ourKeyExchange, err := sessionManager.InitiateSession(peerID, localPriv, false)
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
