package pubsub

import (
	"context"
	"fmt"
	"log"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
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

func (p *PubSubService) ListenToTopic(sub *pubsub.Subscription) {
	go func() {
		for {
			msg, err := sub.Next(p.Ctx)
			if err != nil {
				log.Println("Subscription error:", err)
				return
			}
			if msg.ReceivedFrom == p.Host.ID() {
				continue
			}
			fmt.Printf("\U0001f4e8 PubSub msg from %s: %s\n", msg.ReceivedFrom, string(msg.Data))
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
