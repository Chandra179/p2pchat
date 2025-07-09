package peer

import (
	"log"

	"github.com/libp2p/go-libp2p/core/network"
)

func (p *PeerInfo) ChatHandler() {
	p.Host.SetStreamHandler("/customprotocol", func(s network.Stream) {
		log.Println("Awesome! We're now communicating via the relay!")
		s.Close()
	})
}
