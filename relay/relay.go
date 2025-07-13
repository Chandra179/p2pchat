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
	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", cfg.RelayPort)
	advertiseAddr := fmt.Sprintf("/ip4/%s/tcp/%s", cfg.RelayIP, cfg.RelayPort)

	r, err := libp2p.New(
		libp2p.Identity(cfg.RelayPrivKey),
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.AddrsFactory(func(addrs []ma.Multiaddr) []ma.Multiaddr {
			adv, _ := ma.NewMultiaddr(advertiseAddr)
			return []ma.Multiaddr{adv}
		}),
		libp2p.EnableRelayService(),
	)
	if err != nil {
		fmt.Printf("Failed to create relay: %v\n", err)
		return
	}
	_, err = relay.New(r)
	if err != nil {
		fmt.Printf("Failed to instantiate the relay: %v\n", err)
		return
	}
	relayinfo := peer.AddrInfo{
		ID:    r.ID(),
		Addrs: r.Addrs(),
	}
	fmt.Println(relayinfo.ID.String())
	fmt.Println(relayinfo.Addrs)
	select {}
}
