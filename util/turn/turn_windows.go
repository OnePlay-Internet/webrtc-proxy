package turn

import (
	"fmt"
	"net"

	"github.com/pion/turn/v4"
)

func PacketConfig(ip string, port int) ([]turn.PacketConnConfig, error) {
	udpListener, err := net.ListenPacket("udp4", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, fmt.Errorf("Failed to create TURN server listener: %s", err.Error())
	}

	relayAddressGenerator := &turn.RelayAddressGeneratorStatic{
		RelayAddress: net.ParseIP(ip), // Claim that we are listening on IP passed by user
		Address:      "0.0.0.0",       // But actually be listening on every interface
	}

	return []turn.PacketConnConfig{turn.PacketConnConfig{
		PacketConn:            udpListener,
		RelayAddressGenerator: relayAddressGenerator,
	}}, nil
}
