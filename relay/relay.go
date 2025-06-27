package relay

import (
	"fmt"
	"p2p/config"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	ma "github.com/multiformats/go-multiaddr"
)

func RunRelay(cfg *config.Config) {
	if cfg.PublicIP == "" || cfg.RelayPort == "" {
		fmt.Println("PUBLIC_IP or RELAY_TCP_PORT not set in config")
		return
	}
	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", cfg.RelayPort)
	advertiseAddr := fmt.Sprintf("/ip4/%s/tcp/%s", cfg.PublicIP, cfg.RelayPort)

	relay1, err := libp2p.New(
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.AddrsFactory(func(addrs []ma.Multiaddr) []ma.Multiaddr {
			adv, _ := ma.NewMultiaddr(advertiseAddr)
			return []ma.Multiaddr{adv}
		}),
	)
	if err != nil {
		fmt.Printf("Failed to create relay1: %v\n", err)
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
