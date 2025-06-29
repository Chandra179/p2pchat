package peer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"p2p/config"
	"p2p/utils"
	"strings"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	autonat "github.com/libp2p/go-libp2p/p2p/host/autonat"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	ma "github.com/multiformats/go-multiaddr"
)

type PeerInfo struct {
	RoutedHost   rhost.RoutedHost
	RelayID      peer.ID
	PeerID       peer.ID
	RelayAddr    ma.Multiaddr
	TargetPeerID peer.ID
	Host         host.Host
}

type ConnLogger struct{}

func (cl *ConnLogger) Listen(net network.Network, addr ma.Multiaddr)      {}
func (cl *ConnLogger) ListenClose(net network.Network, addr ma.Multiaddr) {}
func (cl *ConnLogger) Connected(net network.Network, conn network.Conn) {
	remoteAddr := conn.RemoteMultiaddr().String()
	localAddr := conn.LocalMultiaddr().String()

	if strings.Contains(remoteAddr, "p2p-circuit") {
		fmt.Printf("[Notifiee] 🔁 Connected via RELAY: %s <-> %s\n", localAddr, remoteAddr)
	} else {
		fmt.Printf("[Notifiee] 📡 Connected via DIRECT (Hole Punched): %s <-> %s\n", localAddr, remoteAddr)
	}
}
func (cl *ConnLogger) Disconnected(net network.Network, conn network.Conn) {
	fmt.Printf("[Notifiee] Disconnected: %s <-> %s\n", conn.LocalMultiaddr(), conn.RemoteMultiaddr())
}
func (cl *ConnLogger) OpenedStream(net network.Network, stream network.Stream) {
	fmt.Printf("[Notifiee] OpenedStream: %s -> %s\n", stream.Conn().LocalMultiaddr(), stream.Conn().RemoteMultiaddr())
}
func (cl *ConnLogger) ClosedStream(net network.Network, stream network.Stream) {
	fmt.Printf("[Notifiee] ClosedStream: %s -> %s\n", stream.Conn().LocalMultiaddr(), stream.Conn().RemoteMultiaddr())
}

func initPeer(cfg *config.Config) (*PeerInfo, error) {
	// priv key is for relay ID
	privKeyRelay, err := utils.DecodePrivateKey(cfg.RelayID)
	if err != nil {
		fmt.Printf("Failed to decode private key: %v\n", err)
		return nil, err
	}
	relayID, err := peer.IDFromPrivateKey(privKeyRelay)
	if err != nil {
		log.Printf("Failed to derive relay ID from private key: %v", err)
		return nil, err
	}
	privKeyPeer, err := utils.DecodePrivateKey(cfg.PeerID)
	if err != nil {
		fmt.Printf("Failed to decode private key: %v\n", err)
		return nil, err
	}
	relayAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", cfg.PublicIP, cfg.RelayPort))
	if err != nil {
		log.Printf("Failed to parse multiaddr: %v", err)
		return nil, err
	}
	peerHost, err := libp2p.New(
		libp2p.Identity(privKeyPeer),
		libp2p.NoListenAddrs,
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
	)
	if err != nil {
		log.Printf("Failed to create node: %v", err)
		return nil, err
	}

	// Register Notifiee for live connection transitions
	peerHost.Network().Notify(&ConnLogger{})
	peerNat, err := autonat.New(peerHost)
	if err != nil {
		fmt.Println("AutoNAT failed:", err)
	}
	fmt.Println("AutoNAT status:", peerNat.Status())
	peerHost.SetStreamHandler("/customprotocol", func(s network.Stream) {
		buf := make([]byte, 1024)
		n, err := s.Read(buf)
		if err != nil {
			fmt.Println("Error reading from stream:", err)
		} else {
			fmt.Printf("Received message: %s\n", string(buf[:n]))
		}
		s.Close()
	})
	fmt.Println("Peer ID:", peerHost.ID())
	return &PeerInfo{
		RelayID:   relayID,
		RelayAddr: relayAddr,
		Host:      peerHost,
	}, nil
}

func (p *PeerInfo) connectRelay() {
	relayinfo := peer.AddrInfo{
		ID:    p.RelayID,
		Addrs: []ma.Multiaddr{p.RelayAddr},
	}
	if err := p.Host.Connect(context.Background(), relayinfo); err != nil {
		log.Printf("Failed to connect peer to relay: %v", err)
		return
	}
	_, err := client.Reserve(context.Background(), p.Host, relayinfo)
	if err != nil {
		log.Printf("failed to receive a relay reservation from relay. %v", err)
		return
	}
}

func (p *PeerInfo) ConnectPeer(targetPeerID string) error {
	targetRelayaddr, err := ma.NewMultiaddr("/p2p/" + p.RelayID.String() + "/p2p-circuit/p2p/" + targetPeerID)
	if err != nil {
		log.Println(err)
		return err
	}
	targetID, err := peer.Decode(targetPeerID)
	if err != nil {
		log.Printf("Failed to decode target peer ID: %v", err)
		return err
	}
	targetPeer := peer.AddrInfo{
		ID:    targetID,
		Addrs: []ma.Multiaddr{targetRelayaddr},
	}
	if err := p.Host.Connect(context.Background(), targetPeer); err != nil {
		log.Printf("Unexpected error here. Failed to connect peer to target peer: %v", err)
		return err
	}
	p.TargetPeerID = targetID
	return nil
}

func (p *PeerInfo) SendMessage(message string) error {
	// Because we don't have a direct connection to the destination node - we have a relayed connection -
	// the connection is marked as transient. Since the relay limits the amount of data that can be
	// exchanged over the relayed connection, the application needs to explicitly opt-in into using a
	// relayed connection. In general, we should only do this if we have low bandwidth requirements,
	// and we're happy for the connection to be killed when the relayed connection is replaced with a
	// direct (holepunched) connection.
	s, err := p.Host.NewStream(network.WithAllowLimitedConn(context.Background(), "customprotocol"), p.TargetPeerID, "/customprotocol")
	if err != nil {
		return errors.New("Whoops, this should have worked...: " + err.Error())
	}

	s.Read(make([]byte, 1)) // block until the handler closes the stream
	defer s.Close()

	_, err = s.Write([]byte(message))
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		return err
	}
	return nil
}

func RunPeer(cfg *config.Config) (*PeerInfo, error) {
	peerInfo, err := initPeer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize peer: %v", err)
	}
	peerInfo.connectRelay()
	return peerInfo, nil
}
