package protocol

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"filippo.io/edwards25519"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

type EncryptedMessage struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Payload []byte `json:"payload"`
}

// --- ACK struct ---
type AckMessage struct {
	Status string `json:"status"`
}

func deriveSharedKey(priv crypto.PrivKey, pub crypto.PubKey) ([]byte, error) {
	sPriv, err := privToX25519(priv)
	if err != nil {
		return nil, err
	}
	rPub, err := pubToX25519(pub)
	if err != nil {
		return nil, err
	}

	shared, err := curve25519.X25519(sPriv[:], rPub[:])
	if err != nil {
		return nil, err
	}

	return shared, nil
}

func encrypt(sharedKey, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(sharedKey)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil // prepend nonce to ciphertext
}

func decrypt(sharedKey, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(sharedKey)
	if err != nil {
		return nil, err
	}
	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	return aead.Open(nil, nonce, ciphertext, nil)
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

	peerPub := stream.Conn().RemotePublicKey()
	if peerPub == nil {
		fmt.Println("no public key from remote")
		return
	}

	sharedKey, err := deriveSharedKey(priv, peerPub)
	if err != nil {
		fmt.Println("shared key failed:", err)
		return
	}

	plaintext, err := decrypt(sharedKey, msg.Payload)
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

// --- Send message ---
func SendPrivateMessage(p protocol.ID, h host.Host, priv crypto.PrivKey, peerID peer.ID, text string) error {
	stream, err := h.NewStream(context.TODO(), peerID, p)
	if err != nil {
		return err
	}
	defer stream.Close()

	pub := h.Peerstore().PubKey(peerID)
	if pub == nil {
		return fmt.Errorf("no pubkey for peer %s", peerID)
	}

	sharedKey, err := deriveSharedKey(priv, pub)
	if err != nil {
		return err
	}

	ciphertext, err := encrypt(sharedKey, []byte(text))
	if err != nil {
		return err
	}

	msg := EncryptedMessage{
		From:    h.ID().String(),
		To:      peerID.String(),
		Payload: ciphertext,
	}

	encoder := json.NewEncoder(stream)
	err = encoder.Encode(msg)
	if err != nil {
		return err
	}

	// Wait for ACK with timeout
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
			return fmt.Errorf("received non-ok ACK: %s", ack.Status)
		}
		fmt.Println("[ACK] Received ACK from receiver")
		return nil
	case err := <-errCh:
		return fmt.Errorf("failed to receive ACK: %w", err)
	case <-time.After(3 * time.Second):
		return fmt.Errorf("timeout waiting for ACK")
	}
}

func privToX25519(priv crypto.PrivKey) ([32]byte, error) {
	var xpriv [32]byte

	// Extract raw Ed25519 private key
	raw, err := priv.Raw()
	if err != nil {
		return xpriv, err
	}
	if len(raw) != 64 {
		return xpriv, errors.New("invalid ed25519 private key length")
	}

	// First 32 bytes of raw is the private seed
	h := sha512.Sum512(raw[:32])
	h[0] &= 248
	h[31] &= 127
	h[31] |= 64
	copy(xpriv[:], h[:32]) // X25519 private scalar

	return xpriv, nil
}

func pubToX25519(pub crypto.PubKey) ([32]byte, error) {
	var xpub [32]byte

	raw, err := pub.Raw()
	if err != nil {
		return xpub, err
	}
	if len(raw) != ed25519.PublicKeySize {
		return xpub, errors.New("invalid ed25519 pubkey length")
	}

	var A edwards25519.Point
	if _, err := A.SetBytes(raw); err != nil {
		return xpub, err
	}
	A.MultByCofactor(&A)
	copy(xpub[:], A.BytesMontgomery())

	return xpub, nil
}
