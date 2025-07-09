package peer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

func (p *PeerInfo) ChatHandler() {
	p.Host.SetStreamHandler("/customprotocol", func(s network.Stream) {
		log.Println("Awesome! We're now communicating via the relay!")
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

func (p *PeerInfo) SendSimple(peerID peer.ID, text string) error {
	stream, err := p.Host.NewStream(
		network.WithAllowLimitedConn(context.Background(), "reason"),
		peerID,
		"/customprotocol",
	)
	if err != nil {
		fmt.Println("err creating stream")
		return err
	}
	defer stream.Close()
	encoder := json.NewEncoder(stream)
	if err := encoder.Encode("msg 123"); err != nil {
		return err
	}
	return nil
}
