package privatechat

import (
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// SecureChat coordinates all secure messaging components
type SecureChat struct {
	sessionManager *SessionManager
	streamHandler  *MessageHandler
	messageSender  *MessageSender
	host           host.Host
}

// NewSecureChat creates a new secure chat coordinator
func NewSecureChat(host host.Host) *SecureChat {
	sessionManager := NewSessionManager()
	messageSender := NewMessageSender(host, sessionManager)
	streamHandler := NewMessageHandler(sessionManager, messageSender, host.ID())

	return &SecureChat{
		sessionManager: sessionManager,
		streamHandler:  streamHandler,
		messageSender:  messageSender,
		host:           host,
	}
}

// Init initializes the secure chat protocol
func (sc *SecureChat) Init() {
	sc.host.SetStreamHandler("/secure-chat/1.0.0", sc.streamHandler.HandleStream)
}

// InitiateChat starts a secure chat session with a peer
func (sc *SecureChat) InitiateChat(peerID peer.ID) error {
	return sc.messageSender.InitiateKeyExchange(peerID)
}

// SendMessage sends an encrypted message to a peer
func (sc *SecureChat) SendMessage(peerID peer.ID, message string) error {
	return sc.messageSender.SendEncryptedMessage(peerID, message)
}

// SendMessageWithRetry sends a message with retry logic
func (sc *SecureChat) SendMessageWithRetry(peerID peer.ID, message string, maxRetries int) error {
	return sc.messageSender.SendMessageWithRetry(peerID, message, maxRetries)
}

// BroadcastMessage sends a message to all peers with active sessions
func (sc *SecureChat) BroadcastMessage(message string) error {
	return sc.messageSender.BroadcastMessage(message)
}

// HasSession checks if a secure session exists with a peer
func (sc *SecureChat) HasSession(peerID peer.ID) bool {
	return sc.sessionManager.HasSession(peerID)
}

// CloseSession closes a secure session with a peer
func (sc *SecureChat) CloseSession(peerID peer.ID) {
	sc.sessionManager.CloseSession(peerID)
}

// ListActiveSessions returns a list of peer IDs with active sessions
func (sc *SecureChat) ListActiveSessions() []peer.ID {
	return sc.sessionManager.ListActiveSessions()
}

// GetSessionManager returns the session manager (for advanced usage)
func (sc *SecureChat) GetSessionManager() *SessionManager {
	return sc.sessionManager
}

// GetMessageSender returns the message sender (for advanced usage)
func (sc *SecureChat) GetMessageSender() *MessageSender {
	return sc.messageSender
}

// GetStreamHandler returns the stream handler (for advanced usage)
func (sc *SecureChat) GetStreamHandler() *MessageHandler {
	return sc.streamHandler
}
