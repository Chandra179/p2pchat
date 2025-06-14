package protocol

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/libp2p/go-libp2p/core/network"
)

const ProtocolID = "/chat/1.0.0"

func ChatStreamHandler(s network.Stream) {
	fmt.Println("📥 New chat stream")
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go readLoop(rw)
	go writeLoop(rw)
}

func readLoop(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			return
		}
		fmt.Print("💬 ", str)
	}
}

func writeLoop(rw *bufio.ReadWriter) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("✏️ > ")
		if scanner.Scan() {
			text := scanner.Text()
			rw.WriteString(fmt.Sprintf("%s\n", strings.TrimSpace(text)))
			rw.Flush()
		}
	}
}

// p2pNode.SetStreamHandler(protocol.ProtocolID, protocol.ChatStreamHandler)

// if len(os.Args) > 1 {
// 	// Dial a peer
// 	addr, _ := multiaddr.NewMultiaddr(os.Args[1])
// 	pi, _ := peer.AddrInfoFromP2pAddr(addr)
// 	p2pNode.Host.Connect(ctx, *pi)
// 	s, _ := p2pNode.Host.NewStream(ctx, pi.ID, protocol.ProtocolID)
// 	protocol.ChatStreamHandler(s)
// }
