package peer

import (
	"fmt"
	"log"
	"p2p/config"
	"p2p/utils"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
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
	privKeyPeer, err := utils.DecodePrivateKey(cfg.PeerID)
	if err != nil {
		fmt.Printf("Failed to decode private key: %v\n", err)
		return nil, err
	}
	listenAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", cfg.PeerPort))
	if err != nil {
		return nil, fmt.Errorf("invalid listen address: %w", err)
	}
	peerHost, err := libp2p.New(
		libp2p.Identity(privKeyPeer),
		libp2p.NoListenAddrs,
		libp2p.EnableHolePunching(),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
		libp2p.ListenAddrs(listenAddr),
		// libp2p.EnableAutoRelayWithStaticRelays(),
	)
	if err != nil {
		log.Printf("Failed to create node: %v", err)
		return nil, err
	}
	// peerHost.Network().Notify(&ConnLogger{})
	fmt.Println("Peer ID:", peerHost.ID())
	return &PeerInfo{Host: peerHost, PrivKey: privKeyPeer}, nil
}
