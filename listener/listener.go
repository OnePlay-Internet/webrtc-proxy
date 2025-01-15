package listener

import (
	"github.com/pion/rtp"
)

type Listener interface {
	GetCodec() string
	GetPort() int

	RegisterRTPHandler(func(*rtp.Packet)) string
	DeregisterRTPHandler(string)
	SendControlMsg([]byte)

	Close()
}
