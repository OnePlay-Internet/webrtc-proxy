package listener

import (
	"github.com/pion/rtp"
)

type Listener interface {
	GetCodec() string

	RegisterRTPHandler(string, func(*rtp.Packet))
	DeregisterRTPHandler(string)
	SendControlMsg([]byte)

	Close()
}
