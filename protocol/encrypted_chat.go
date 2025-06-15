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

// EncryptedMessage represents a message structure for encrypted communication.
type EncryptedMessage struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Payload   []byte `json:"payload"`
	MessageID string `json:"message_id"`
}

// --- ACK struct ---
type AckMessage struct {
	Status string `json:"status"`
}

var (
	privateMsgCache     *lru.Cache
	privateMsgCacheInit error
)

func init() {
	privateMsgCache, privateMsgCacheInit = lru.New(1024) // 1024 recent message IDs
}

// --- Stream handler ---
func HandlePrivateMessage(stream network.Stream, priv crypto.PrivKey) {
	defer stream.Close()

	var msg EncryptedMessage
	err := json.NewDecoder(bufio.NewReader(stream)).Decode(&msg)
	if err != nil {
		fmt.Println("failed to decode:", err)
		return
	}

	if privateMsgCacheInit != nil {
		fmt.Println("LRU cache init failed:", privateMsgCacheInit)
		return
	}
	if msg.MessageID == "" {
		fmt.Println("missing MessageID, dropping message")
		return
	}
	if _, found := privateMsgCache.Get(msg.MessageID); found {
		// Duplicate message, silently drop
		return
	}
	privateMsgCache.Add(msg.MessageID, struct{}{})

	peerPub := stream.Conn().RemotePublicKey()
	if peerPub == nil {
		fmt.Println("no public key from remote")
		return
	}

	sharedKey, err := cryptoutils.X25519DeriveSharedKey(priv, peerPub)
	if err != nil {
		fmt.Println("shared key failed:", err)
		return
	}

	plaintext, err := cryptoutils.X25519ChaChaDecrypt(sharedKey, msg.Payload)
	if err != nil {
		fmt.Println("decrypt failed:", err)
		return
	}

	fmt.Printf("[Private] %s: %s\n", msg.From, string(plaintext))

	// Send ACK back to sender
	ack := AckMessage{Status: "ok"}
	encoder := json.NewEncoder(stream)
	err = encoder.Encode(ack)
	if err != nil {
		fmt.Println("failed to send ACK:", err)
	}
}

func SendPrivateMessage(p protocol.ID, h host.Host, priv crypto.PrivKey, peerID peer.ID, text string) error {
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

		sharedKey, err := cryptoutils.X25519DeriveSharedKey(priv, pub)
		if err != nil {
			return backoff.Permanent(err)
		}

		ciphertext, err := cryptoutils.X25519ChaChaEncrypt(sharedKey, []byte(text))
		if err != nil {
			return backoff.Permanent(err)
		}

		msg := EncryptedMessage{
			From:      h.ID().String(),
			To:        peerID.String(),
			Payload:   ciphertext,
			MessageID: uuid.NewString(),
		}

		encoder := json.NewEncoder(stream)
		if err := encoder.Encode(msg); err != nil {
			return backoff.Permanent(err)
		}

		ackCh := make(chan AckMessage, 1)
		errCh := make(chan error, 1)
		go func() {
			var ack AckMessage
			dec := json.NewDecoder(bufio.NewReader(stream))
			if err := dec.Decode(&ack); err != nil {
				errCh <- err
				return
			}
			ackCh <- ack
		}()

		select {
		case ack := <-ackCh:
			if ack.Status != "ok" {
				return backoff.Permanent(fmt.Errorf("received non-ok ACK: %s", ack.Status))
			}
			fmt.Println("[ACK] Received ACK from receiver")
			return nil
		case err := <-errCh:
			return err
		case <-time.After(3 * time.Second):
			return fmt.Errorf("timeout waiting for ACK")
		}
	}

	bo := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)
	err := backoff.Retry(operation, bo)
	if err != nil {
		return fmt.Errorf("SendPrivateMessage failed after retries: %w", err)
	}
	return nil
}
