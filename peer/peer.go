package peer

import (
	"context"
	"fmt"
	"log"
	"p2p/privatechat"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	ma "github.com/multiformats/go-multiaddr"
)

type PeerInfo struct {
	RoutedHost  rhost.RoutedHost
	PeerID      peer.ID
	Host        host.Host
	PeerStore   *PeerStore
	PrivateChat privatechat.PrivateChat
}

func InitPeerHost(peerPrivKey crypto.PrivKey) (*PeerInfo, error) {
	ps, err := NewPeerStore()
	if err != nil {
		log.Fatal(err)
	}

	// TODO: configure ip and port for listen address
	h, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.Identity(peerPrivKey),
		libp2p.EnableHolePunching(),
		libp2p.NATPortMap(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		log.Printf("Failed to create node: %v", err)
		return nil, err
	}
	pc := privatechat.NewPrivateChat(h)
	p := PeerInfo{Host: h, PeerStore: ps, PrivateChat: *pc}
	p.PrivateChat.Init()
	fmt.Println("Local protocols:", h.Mux().Protocols())
	fmt.Println("Peer ID:", h.ID())

	return &p, nil
}

func (p *PeerInfo) ConnectAndReserveRelay(relayID peer.ID, relayIP string, relayPort string) {
	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", relayIP, relayPort))
	if err != nil {
		log.Printf("Failed to parse relay multiaddr: %v", err)
		return
	}
	addrInfo := peer.AddrInfo{
		ID:    relayID,
		Addrs: []ma.Multiaddr{addr},
	}
	if err := p.Host.Connect(context.Background(), addrInfo); err != nil {
		log.Printf("Failed too connect to relay: %v", err)
		return
	}
	_, err = client.Reserve(context.Background(), p.Host, addrInfo)
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
