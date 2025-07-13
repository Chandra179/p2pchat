package peer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const CHAT_PROTOCOL = "/private-chat/1.0.0"

func (p *PeerInfo) ChatHandler() {
	p.Host.SetStreamHandler(CHAT_PROTOCOL, func(s network.Stream) {
		defer s.Close()
		var msg string
		decoder := json.NewDecoder(s)
		if err := decoder.Decode(&msg); err != nil {
			fmt.Printf("Failed to receive msg: %v\n", err)
			return
		}
		fmt.Println(msg)
	})
}

func (p *PeerInfo) SendSimple(targetPeerID peer.ID, text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stream, err := p.Host.NewStream(
		network.WithAllowLimitedConn(ctx, "reason"),
		targetPeerID,
		CHAT_PROTOCOL,
	)
	if err != nil {
		fmt.Printf("Failed to send msg: %v\n", err)
		return err
	}
	defer stream.Close()
	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(text); err != nil {
		return err
	}
	return nil
}
