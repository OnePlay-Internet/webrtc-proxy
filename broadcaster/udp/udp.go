package udp

import (
	"fmt"
	"net"
	"os"

	"github.com/pion/rtp"
	"github.com/thinkonmay/thinkremote-rtchub/broadcaster"
	"github.com/thinkonmay/thinkremote-rtchub/listener"
	"github.com/thinkonmay/thinkremote-rtchub/util/thread"
)

type UDPSource struct {
	*net.UDPConn

	stop    chan bool
	channel chan *rtp.Packet
	idr     func()
}

// Close implements broadcaster.Broadcaster.
func (u *UDPSource) Close() {
	u.UDPConn.Close()
}

// Configure implements broadcaster.Broadcaster.
func (u *UDPSource) Configure(l listener.Listener) error {
	thread.SafeLoop(u.stop, 0, func() {
		select {
		case <-u.stop:
		case pkt := <-u.channel:
			if data, err := pkt.Marshal(); err != nil {
				fmt.Printf("failed to unmarshal rtp packet %s\n", err.Error())
				u.idr()
			} else if _, err := u.UDPConn.Write(data); err != nil {
				fmt.Printf("failed to write rtp packet %s\n", err.Error())
				u.idr()
			}
		}
	})

	return nil
}

// OnIDR implements broadcaster.Broadcaster.
func (u *UDPSource) OnIDR(fun func()) {
	u.idr = fun
}

// SendRTPPacket implements broadcaster.Broadcaster.
func (u *UDPSource) SendRTPPacket(pkt *rtp.Packet) {
	u.channel <- pkt
}

func NewUDPSource(addr string) (broadcaster.Broadcaster, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", os.Args[1])
	if err != nil {
		return nil, err
	}
	UDPConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}

	return &UDPSource{
		UDPConn: UDPConn,
		stop:    make(chan bool),
		channel: make(chan *rtp.Packet),
		idr:     func() {},
	}, nil
}
