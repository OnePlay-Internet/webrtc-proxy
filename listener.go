package proxy

import (
	"fmt"
	"net"

	"github.com/pion/randutil"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/thinkonmay/thinkremote-rtchub/listener"
	"github.com/thinkonmay/thinkremote-rtchub/listener/multiplexer"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/bits"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/h264"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/wrapper"
	"github.com/thinkonmay/thinkremote-rtchub/util/thread"
)

type VideoPipeline struct {
	closed chan bool

	clockRate   float64
	codec       string
	multiplexer *multiplexer.Multiplexer
}

func UDPListener(codec string) (listener.Listener, error) {
	packetizer := wrapper.PacketizerWrapper{
		Fun: h264.RTPPay,

		Timestamp: randutil.NewMathRandomGenerator().Uint32(),
		MTU:       1400,
	}

	port, err := getFreePort()
	if err != nil {
		return nil, err
	}

	pipeline := &VideoPipeline{
		codec: webrtc.MimeTypeH264,

		closed:      make(chan bool, 2),
		clockRate:   90000,
		multiplexer: multiplexer.NewMultiplexer("video", &packetizer),
	}

	pc, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 1024*1024*10)
	firsttime := true
	thread.HighPriorityLoop(pipeline.closed, func() {
		if n, _, err := pc.ReadFrom(buf); err != nil {
			reader := bits.NewReader(buf[:n])
			sample := reader.ReadUint32()
			pipeline.multiplexer.Send(reader.Left(), sample)
		}

		if firsttime {
			fmt.Println("capturing video")
			firsttime = false
		}

	})
	return pipeline, nil
}

func (p *VideoPipeline) GetCodec() string {
	return p.codec
}

func (p *VideoPipeline) Close() {
	thread.TriggerStop(p.closed)
}

func (p *VideoPipeline) RegisterRTPHandler(id string, fun func(pkt *rtp.Packet)) {
	p.multiplexer.RegisterRTPHandler(id, fun)
}

func (p *VideoPipeline) DeregisterRTPHandler(id string) {
	p.multiplexer.DeregisterRTPHandler(id)
}

func getFreePort() (port int, err error) {
	if a, err := net.ResolveUDPAddr("udp", ":0"); err == nil {
		if l, err := net.ListenUDP("udp", a); err == nil {
			defer l.Close()
			return l.LocalAddr().(*net.UDPAddr).Port, nil
		}
	}
	return
}

func (p *VideoPipeline) SendControlMsg([]byte) {
	return
}
