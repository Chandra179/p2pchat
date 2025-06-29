package p2p

import (
	"encoding/base64"
	"fmt"
	"p2p/config"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	ma "github.com/multiformats/go-multiaddr"
)

func RunRelay(cfg *config.Config) {
	privKey, err := decodePrivateKey(cfg.RelayID)
	if err != nil {
		fmt.Printf("Failed to decode private key: %v\n", err)
		return
	}

	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", cfg.RelayPort)
	advertiseAddr := fmt.Sprintf("/ip4/%s/tcp/%s", cfg.PublicIP, cfg.RelayPort)

	relay1, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.AddrsFactory(func(addrs []ma.Multiaddr) []ma.Multiaddr {
			adv, _ := ma.NewMultiaddr(advertiseAddr)
			return []ma.Multiaddr{adv}
		}),
		libp2p.EnableRelayService(),
		libp2p.EnableNATService(),
	)
	if err != nil {
		fmt.Printf("Failed to create relay: %v\n", err)
		return
	}
	_, err = relay.New(relay1)
	if err != nil {
		fmt.Printf("Failed to instantiate the relay: %v\n", err)
		return
	}
	relayinfo := peer.AddrInfo{
		ID:    relay1.ID(),
		Addrs: relay1.Addrs(),
	}
	fmt.Println(relayinfo.ID.String())
	fmt.Println(relayinfo.Addrs)
	select {}
}

// GenerateStaticRelayKey generates a new Ed25519 private key and returns it as a base64-encoded string.
func GenerateStaticRelayKey() (string, error) {
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}

	bytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	fmt.Println("Generated static relay key:", base64.StdEncoding.EncodeToString(bytes))
	return base64.StdEncoding.EncodeToString(bytes), nil
}
