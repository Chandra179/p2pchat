package p2p

import (
	"context"
	"fmt"
	"log"
	"p2p/config"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	ma "github.com/multiformats/go-multiaddr"
)

type PeerInfo struct {
	RelayID   peer.ID
	RelayAddr ma.Multiaddr
	Host      host.Host
}

func InitPeer(cfg *config.Config) (*PeerInfo, error) {
	// priv key is for relay ID
	privKey, err := decodePrivateKey(cfg.RelayID)
	if err != nil {
		fmt.Printf("Failed to decode private key: %v\n", err)
		return nil, err
	}
	relayID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		log.Printf("Failed to derive relay ID from private key: %v", err)
		return nil, err
	}
	relayAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", cfg.RelayIP, cfg.RelayPort))
	if err != nil {
		log.Printf("Failed to parse multiaddr: %v", err)
		return nil, err
	}
	peerHost, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.EnableRelay(),
	)
	if err != nil {
		log.Printf("Failed to create node: %v", err)
		return nil, err
	}
	return &PeerInfo{
		RelayID:   relayID,
		RelayAddr: relayAddr,
		Host:      peerHost,
	}, nil
}

func (p *PeerInfo) ConnectRelay() {
	relayinfo := peer.AddrInfo{
		ID:    p.RelayID,
		Addrs: []ma.Multiaddr{p.RelayAddr},
	}
	if err := p.Host.Connect(context.Background(), relayinfo); err != nil {
		log.Printf("Failed to connect peer to relay: %v", err)
		return
	}
	p.Host.SetStreamHandler("/customprotocol", func(s network.Stream) {
		log.Println("Awesome! We're now communicating via the relay!")
		s.Close()
	})
	_, err := client.Reserve(context.Background(), p.Host, relayinfo)
	if err != nil {
		log.Printf("failed to receive a relay reservation from relay. %v", err)
		return
	}

	select {}
}

func (p *PeerInfo) ConnectPeer(targetPeerID string) {
	targetRelayaddr, err := ma.NewMultiaddr("/p2p/" + p.RelayID.String() + "/p2p-circuit/p2p/" + targetPeerID)
	if err != nil {
		log.Println(err)
		return
	}
	targetPeer := peer.AddrInfo{
		ID:    peer.ID(targetPeerID),
		Addrs: []ma.Multiaddr{targetRelayaddr},
	}
	if err := p.Host.Connect(context.Background(), targetPeer); err != nil {
		log.Printf("Unexpected error here. Failed to connect unreachable1 and unreachable2: %v", err)
		return
	}
	log.Printf("Connected to peer %s via relay %s", targetPeerID, p.RelayID.String())
}

func RunPeer(cfg *config.Config) {
	peerInfo, err := InitPeer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize peer: %v", err)
	}
	peerInfo.ConnectRelay()
	go func() {
		for {
			var cmd, arg string
			fmt.Print("> ")
			_, err := fmt.Scanln(&cmd, &arg)
			if err != nil {
				continue
			}
			if cmd == "con" && arg != "" {
				peerInfo.ConnectPeer(arg)
			}
		}
	}()
}
