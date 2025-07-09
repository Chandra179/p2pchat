package peer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

func (p *PeerInfo) ChatHandler() {
	p.Host.SetStreamHandler("/customprotocol", func(s network.Stream) {
		defer s.Close()
		var msg string
		decoder := json.NewDecoder(s)
		if err := decoder.Decode(&msg); err != nil {
			log.Printf("Error decoding message: %v", err)
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
		"/customprotocol",
	)
	if err != nil {
		fmt.Println("err creating stream")
		return err
	}
	defer stream.Close()
	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(text); err != nil {
		return err
	}
	return nil
}
