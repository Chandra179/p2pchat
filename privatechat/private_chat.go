package privatechat

import (
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// PrivateChat coordinates all secure messaging components
type PrivateChat struct {
	sessionManager *SessionManager
	streamHandler  *MessageHandler
	messageSender  *MessageSender
	host           host.Host
}

// NewPrivateChat creates a new secure chat coordinator
func NewPrivateChat(host host.Host) *PrivateChat {
	sessionManager := NewSessionManager()
	messageSender := NewMessageSender(host, sessionManager)
	streamHandler := NewMessageHandler(sessionManager, messageSender, host.ID())

	return &PrivateChat{
		sessionManager: sessionManager,
		streamHandler:  streamHandler,
		messageSender:  messageSender,
		host:           host,
	}
}

// Init initializes the secure chat protocol
func (sc *PrivateChat) init() {
	sc.host.SetStreamHandler("/secure-chat/1.0.0", sc.streamHandler.HandleStream)
}

// InitiateChat starts a secure chat session with a peer
func (sc *PrivateChat) InitiateChat(peerID peer.ID) error {
	return sc.messageSender.InitiateKeyExchange(peerID)
}

// SendMessage sends an encrypted message to a peer
func (sc *PrivateChat) SendMessage(peerID peer.ID, message string) error {
	return sc.messageSender.SendEncryptedMessage(peerID, message)
}

// SendMessageWithRetry sends a message with retry logic
func (sc *PrivateChat) SendMessageWithRetry(peerID peer.ID, message string, maxRetries int) error {
	return sc.messageSender.SendMessageWithRetry(peerID, message, maxRetries)
}

// BroadcastMessage sends a message to all peers with active sessions
func (sc *PrivateChat) BroadcastMessage(message string) error {
	return sc.messageSender.BroadcastMessage(message)
}

// HasSession checks if a secure session exists with a peer
func (sc *PrivateChat) HasSession(peerID peer.ID) bool {
	return sc.sessionManager.HasSession(peerID)
}

// CloseSession closes a secure session with a peer
func (sc *PrivateChat) CloseSession(peerID peer.ID) {
	sc.sessionManager.CloseSession(peerID)
}

// ListActiveSessions returns a list of peer IDs with active sessions
func (sc *PrivateChat) ListActiveSessions() []peer.ID {
	return sc.sessionManager.ListActiveSessions()
}

// GetSessionManager returns the session manager (for advanced usage)
func (sc *PrivateChat) GetSessionManager() *SessionManager {
	return sc.sessionManager
}

// GetMessageSender returns the message sender (for advanced usage)
func (sc *PrivateChat) GetMessageSender() *MessageSender {
	return sc.messageSender
}

// GetStreamHandler returns the stream handler (for advanced usage)
func (sc *PrivateChat) GetStreamHandler() *MessageHandler {
	return sc.streamHandler
}
