package broadcaster

import (
	"github.com/pion/rtp"
	"github.com/thinkonmay/thinkremote-rtchub/listener"
)

type Broadcaster interface {
	Configure(listener.Listener) error
	SendRTPPacket(*rtp.Packet)
	OnIDR(func())

	Close()
}
