package wrapper

import (
	"github.com/pion/rtp"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/core"
)

type PacketizerWrapper struct {
	Fun       func(uint16, core.HandlerFunc) core.HandlerFunc
	Timestamp uint32
	MTU       uint16
}

func (wr *PacketizerWrapper) Packetize(buff []byte, samples uint32) []*rtp.Packet {
	result := []*rtp.Packet{}
	final := func(packet *core.Packet) {
		result = append(result, packet)
	}

	wr.Timestamp += samples
	wr.Fun(wr.MTU, final)(&core.Packet{
		Payload: buff,
	})

	return result
}
