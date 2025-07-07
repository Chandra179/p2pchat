package peer

import (
	"context"
	"fmt"
	"log"
	"p2p/config"
	"p2p/cryptoutils"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"
)

type PeerInfo struct {
	RoutedHost rhost.RoutedHost
	PeerID     peer.ID
	Host       host.Host
	PrivKey    crypto.PrivKey
}

func InitPeerHost(cfg *config.Config) (*PeerInfo, error) {
	privKeyPeer, err := cryptoutils.DecodeBase64Key(cfg.PeerID)
	if err != nil {
		fmt.Printf("Failed to decode private key: %v\n", err)
		return nil, err
	}
	// listenAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", cfg.PeerPort))
	// if err != nil {
	// 	return nil, fmt.Errorf("invalid listen address: %w", err)
	// }
	peerHost, err := libp2p.New(
		libp2p.Identity(privKeyPeer),
		libp2p.EnableHolePunching(),
		// libp2p.NATPortMap(),
		// libp2p.ListenAddrs(listenAddr),
		// libp2p.EnableRelay(),
	)
	if err != nil {
		log.Printf("Failed to create node: %v", err)
		return nil, err
	}
	// TODO: might need seperate this into a new function cause ProtocolID is set when peer send a message to other peer
	// the protocolID is identifier for a stream like (chat channel name)
	peerHost.SetStreamHandler("/customprotocol", func(s network.Stream) {
		log.Println("Awesome! We're now communicating via the relay!")

		// End the example
		s.Close()
		// chat.HandlePrivateMessage(s, privKeyPeer)
	})

	// peerHost.Network().Notify(&ConnLogger{})
	fmt.Println("Peer ID:", peerHost.ID())
	return &PeerInfo{Host: peerHost, PrivKey: privKeyPeer}, nil
}

// Hosts that want to have messages relayed on their behalf need to reserve a slot
// with the circuit relay service host
func (p *PeerInfo) ConnectAndReserveRelay(relayID string) {
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
	// _, err = client.Reserve(context.Background(), p.Host, relayinfo)
	// if err != nil {
	// 	log.Printf("unreachable2 failed to receive a relay reservation from relay1. %v", err)
	// 	return
	// }
	fmt.Println("success connect to relay")
}
