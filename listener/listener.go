package listener

import (
	"github.com/pion/rtp"
)

type Listener interface {
	GetCodec() string
	GetPort() int

	RegisterRTPHandler(string, func(*rtp.Packet))
	DeregisterRTPHandler(string)
	SendControlMsg([]byte)

	Close()
}
