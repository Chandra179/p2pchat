package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

func SendSimple(p protocol.ID, h host.Host, priv crypto.PrivKey, peerID peer.ID, text string) error {
	stream, err := h.NewStream(
		network.WithAllowLimitedConn(context.Background(), "customprotocol"),
		peerID,
		"/customprotocol",
	)
	if err != nil {
		fmt.Println("err creating stream")
		return err
	}
	defer stream.Close()
	conn := stream.Conn()
	addr := conn.RemoteMultiaddr().String()
	fmt.Println("Stream connected via:", addr)

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode("msg 123"); err != nil {
		return err
	}
	return nil
}
