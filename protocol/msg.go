package protocol

import (
	"time"
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

var (
	sessionManager *SessionManager
)

func init() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			sessionManager.ClearExpiredSessions()
		}
	}()
}
