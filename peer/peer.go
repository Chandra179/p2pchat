package peer

import (
	"context"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	ma "github.com/multiformats/go-multiaddr"
)

type PeerInfo struct {
	RoutedHost rhost.RoutedHost
	PeerID     peer.ID
	Host       host.Host
}

func InitPeerHost(peerPrivKey crypto.PrivKey) (*PeerInfo, error) {
	// TODO: configure ip and port for listen address
	h, err := libp2p.New(
		libp2p.Identity(peerPrivKey),
		libp2p.EnableHolePunching(),
		libp2p.NATPortMap(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		log.Printf("Failed to create node: %v", err)
		return nil, err
	}

	fmt.Println("Peer ID:", h.ID())
	return &PeerInfo{Host: h}, nil
}

func (p *PeerInfo) ConnectAndReserveRelay(relayID peer.ID) {
	relayAddr, err := ma.NewMultiaddr("/ip4/35.208.121.167/tcp/9000")
	if err != nil {
		log.Printf("Failed to parse relay multiaddr: %v", err)
		return
	}
	relayinfo := peer.AddrInfo{
		ID:    relayID,
		Addrs: []ma.Multiaddr{relayAddr},
	}
	if err := p.Host.Connect(context.Background(), relayinfo); err != nil {
		log.Printf("Failed too connect to relay: %v", err)
		return
	}
	_, err = client.Reserve(context.Background(), p.Host, relayinfo)
	if err != nil {
		log.Printf("Failed to reserved relay %v", err)
		return
	}
	fmt.Println("success connect to relay")
}

func (p *PeerInfo) Ping(id peer.ID, addr string) {
	maddr, err := ma.NewMultiaddr(addr)
	if err != nil {
		log.Printf("Invalid multiaddr: %v", err)
		return
	}
	fmt.Println(p.Host.Network().CanDial(id, maddr))
}

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

func (p *PeerInfo) ChatHandler() {
	p.Host.SetStreamHandler("/customprotocol", func(s network.Stream) {
		log.Println("Awesome! We're now communicating via the relay!")
		s.Close()
	})
}
