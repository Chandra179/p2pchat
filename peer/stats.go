package peer

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
)

func (p *PeerInfo) Stat(id peer.ID) {
	conn := p.Host.Network().ConnsToPeer(id)
	for _, c := range conn {
		fmt.Println(c.IsClosed())
		fmt.Println(c.Scope().Stat().NumConnsInbound)
		fmt.Println(c.Scope().Stat().NumConnsOutbound)
		fmt.Println(c.Scope().Stat().NumStreamsInbound)
		fmt.Println(c.Scope().Stat().NumStreamsOutbound)
		fmt.Println(c.Stat().Direction)
		fmt.Println(c.Stat().Opened)
		fmt.Println(c.Stat().NumStreams)
		fmt.Println(c.Stat().Limited)
		fmt.Println(c.ConnState().Security)
		fmt.Println(c.ConnState().StreamMultiplexer)
		fmt.Println(c.ConnState().Transport)
	}
}
