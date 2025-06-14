package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
)

type PubSubService struct {
	Ctx    context.Context
	Host   host.Host
	PubSub *pubsub.PubSub
	Topics map[string]*pubsub.Topic
}

func NewPubSubService(ctx context.Context, h host.Host) (*PubSubService, error) {
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}

	return &PubSubService{
		Ctx:    ctx,
		Host:   h,
		PubSub: ps,
		Topics: make(map[string]*pubsub.Topic),
	}, nil
}

func (p *PubSubService) JoinTopic(topicName string) (*pubsub.Topic, *pubsub.Subscription, error) {
	topic, err := p.PubSub.Join(topicName)
	if err != nil {
		return nil, nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, nil, err
	}

	p.Topics[topicName] = topic
	return topic, sub, nil
}

func (s *PubSubService) ListenToTopic(sub *pubsub.Subscription) {
	go func() {
		for {
			msg, err := sub.Next(s.Ctx)
			if err != nil {
				log.Println("Error reading from subscription:", err)
				continue
			}

			chatMsg, err := UnmarshalChatMessage(msg.Data)
			if err != nil {
				log.Println("Invalid chat message:", err)
				continue
			}

			// Lookup sender's public key
			pubKey := s.Host.Peerstore().PubKey(msg.ReceivedFrom)
			if pubKey == nil || !chatMsg.Verify(pubKey) {
				log.Println("⚠️  Invalid signature from", chatMsg.Sender)
				continue
			}

			fmt.Printf("[%s][%s] %s\n", time.Unix(chatMsg.Timestamp, 0).Format("15:04:05"), chatMsg.Sender, chatMsg.Message)
		}
	}()
}

func (p *PubSubService) Publish(topicName string, data []byte) error {
	topic, ok := p.Topics[topicName]
	if !ok {
		return fmt.Errorf("topic %s not joined", topicName)
	}
	return topic.Publish(p.Ctx, data)
}

type ChatMessage struct {
	Sender    string `json:"sender"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
	Signature []byte `json:"signature,omitempty"` // signed over the rest of the message
}

func NewChatMessage(sender, message string) *ChatMessage {
	return &ChatMessage{
		Sender:    sender,
		Timestamp: time.Now().Unix(),
		Message:   message,
	}
}

func (m *ChatMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func UnmarshalChatMessage(data []byte) (*ChatMessage, error) {
	var msg ChatMessage
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

func (m *ChatMessage) Sign(priv crypto.PrivKey) error {
	// Copy of message with empty Signature
	copy := *m
	copy.Signature = nil

	data, err := json.Marshal(copy)
	if err != nil {
		return err
	}

	sig, err := priv.Sign(data)
	if err != nil {
		return err
	}

	m.Signature = sig
	return nil
}

func (m *ChatMessage) Verify(pub crypto.PubKey) bool {
	copy := *m
	copy.Signature = nil

	data, err := json.Marshal(copy)
	if err != nil {
		return false
	}

	ok, err := pub.Verify(data, m.Signature)
	return err == nil && ok
}
