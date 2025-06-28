package p2p

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"p2p/config"
	"strings"

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
	peerHost.SetStreamHandler("/customprotocol", func(s network.Stream) {
		fmt.Println("IT WORKSSSSSSSSSSSSSSSSSSSSSSSS")
		s.Close()
	})
	fmt.Println("Peer ID:", peerHost.ID())
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
	log.Printf("Connected to peer %s via relay %s", targetPeerID, p.RelayID)
	return nil
}

// SendMessage opens a stream to the target peer and sends a message
func (p *PeerInfo) SendMessage(targetPeerID string, message string) error {
	targetID, err := peer.Decode(targetPeerID)
	if err != nil {
		log.Printf("Failed to decode target peer ID: %v", err)
		return err
	}
	// Because we don't have a direct connection to the destination node - we have a relayed connection -
	// the connection is marked as transient. Since the relay limits the amount of data that can be
	// exchanged over the relayed connection, the application needs to explicitly opt-in into using a
	// relayed connection. In general, we should only do this if we have low bandwidth requirements,
	// and we're happy for the connection to be killed when the relayed connection is replaced with a
	// direct (holepunched) connection.
	s, err := p.Host.NewStream(network.WithAllowLimitedConn(context.Background(), "customprotocol"), targetID, "/customprotocol")
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
	fmt.Println("Message sent!")
	return nil
}

func RunPeer(cfg *config.Config) {
	peerInfo, err := InitPeer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize peer: %v", err)
	}
	peerInfo.ConnectRelay()
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("> ")
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Input error: %v\n", err)
				continue
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			cmd, arg := fields[0], fields[1]
			msg := ""
			if cmd == "send" && len(fields) > 2 {
				msg = strings.Join(fields[2:], " ")
			}
			if cmd == "con" && arg != "" {
				err := peerInfo.ConnectPeer(arg)
				if err != nil {
					fmt.Printf("ConnectPeer error: %v\n", err)
				}
			} else if cmd == "send" && arg != "" && msg != "" {
				err := peerInfo.SendMessage(arg, msg)
				if err != nil {
					fmt.Printf("SendMessage error: %v\n", err)
				}
			}
		}
	}()
}
