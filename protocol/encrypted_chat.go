package protocol

import (
	"bufio"
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"golang.org/x/crypto/chacha20poly1305"
)

type EncryptedMessage struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Payload []byte `json:"payload"`
}

func deriveP256ECDHSharedKey(priv crypto.PrivKey, pub crypto.PubKey) ([]byte, error) {
	privStd, err := crypto.PrivKeyToStdKey(priv)
	if err != nil {
		return nil, err
	}
	pubStd, err := crypto.PubKeyToStdKey(pub)
	if err != nil {
		return nil, err
	}

	privKey, ok1 := privStd.(*ecdsa.PrivateKey)
	pubKey, ok2 := pubStd.(*ecdsa.PublicKey)
	if !ok1 || !ok2 {
		return nil, errors.New("only ECDSA keys are supported")
	}

	curve := ecdh.P256()

	// Construct ECDH private key
	privKeyECDH, err := curve.NewPrivateKey(privKey.D.Bytes())
	if err != nil {
		return nil, err
	}

	// Construct ECDH public key
	pubKeyECDH, err := curve.NewPublicKey(elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y))
	if err != nil {
		return nil, err
	}

	sharedSecret, err := privKeyECDH.ECDH(pubKeyECDH)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(sharedSecret)
	return hash[:], nil
}

// --- Symmetric AEAD encryption ---
func encrypt(sharedKey, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(sharedKey)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize()) // empty = deterministic for demo; use rand in prod
	return aead.Seal(nonce, nonce, plaintext, nil), nil
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
	return aead.Open(nil, nonce, ciphertext[nonceSize:], nil)
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

	sharedKey, err := deriveP256ECDHSharedKey(priv, peerPub)
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

	sharedKey, err := deriveP256ECDHSharedKey(priv, pub)
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

	data, _ := json.Marshal(msg)
	_, err = stream.Write(data)
	return err
}
