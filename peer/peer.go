package peer

import (
	"context"
	"fmt"
	"log"
	"p2p/chat"
	"p2p/config"
	"p2p/cryptoutils"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/multiformats/go-multiaddr"
)

type HostInfo struct {
	RoutedHost rhost.RoutedHost
	PeerID     peer.ID
	Host       host.Host
	Identity   crypto.PrivKey
}

func InitPeerHost(cfg *config.Config) (*HostInfo, error) {
	privKeyPeer, err := cryptoutils.DecodeBase64Key(cfg.PeerID)
	if err != nil {
		fmt.Printf("Failed to decode private key: %v\n", err)
		return nil, err
	}
	// TODO: configure ip and port for listen address
	peerHost, err := libp2p.New(
		libp2p.Identity(privKeyPeer),
		libp2p.EnableHolePunching(),
		libp2p.NATPortMap(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		log.Printf("Failed to create node: %v", err)
		return nil, err
	}
	peerHost.SetStreamHandler("/customprotocol", func(s network.Stream) {
		log.Println("Awesome! We're now communicating via the relay!")
		chat.HandlePrivateMessage(s, privKeyPeer)
		s.Close()
	})

	fmt.Println("Peer ID:", peerHost.ID())
	return &HostInfo{Host: peerHost, Identity: privKeyPeer}, nil
}

func (p *HostInfo) ConnectAndReserveRelay(relayID string) {
	relayAddr, err := multiaddr.NewMultiaddr("/ip4/35.208.121.167/tcp/9000")
	if err != nil {
		log.Printf("Failed to parse relay multiaddr: %v", err)
		return
	}
	key, err := cryptoutils.DecodeBase64Key(relayID)
	if err != nil {
		fmt.Printf("Failed to decode private key: %v\n", err)
		return
	}
	id, err := peer.IDFromPrivateKey(key)
	if err != nil {
		fmt.Printf("Failed to extract peer id from private key: %v\n", err)
		return
	}
	relayinfo := peer.AddrInfo{
		ID:    id,
		Addrs: []multiaddr.Multiaddr{relayAddr},
	}
	if err := p.Host.Connect(context.Background(), relayinfo); err != nil {
		log.Printf("Failed to connect unreachable1 and relay1: %v", err)
		return
	}
	_, err = client.Reserve(context.Background(), p.Host, relayinfo)
	if err != nil {
		log.Printf("unreachable2 failed to receive a relay reservation from relay1. %v", err)
		return
	}
	fmt.Println("success connect to relay")
}
