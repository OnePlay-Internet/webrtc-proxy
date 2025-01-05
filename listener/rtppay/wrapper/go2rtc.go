package wrapper

import (
	"fmt"

	"github.com/pion/rtp"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/core"
)

type PacketizerWrapper struct {
	Fun       func(uint16, core.HandlerFunc) core.HandlerFunc
	Timestamp uint32
	MTU       uint16
}

func (wr *PacketizerWrapper) Packetize(buff []byte, samples uint32) (result []*rtp.Packet) {
	defer fmt.Printf("packetize %d gain %d rtp packets\n", len(buff),len(result))
	final := func(packet *core.Packet) {
		fmt.Println("hi")
		result = append(result, packet)
	}

	wr.Timestamp += samples
	wr.Fun(wr.MTU, final)(&core.Packet{
		Payload: buff,
	})

	return result
}
