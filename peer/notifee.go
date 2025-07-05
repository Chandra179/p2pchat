package peer

import (
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p/core/network"
	ma "github.com/multiformats/go-multiaddr"
)

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
