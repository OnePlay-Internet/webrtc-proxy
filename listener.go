package proxy

import (
	"fmt"
	"net"
	"time"

	"github.com/pion/randutil"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/thinkonmay/thinkremote-rtchub/listener"
	"github.com/thinkonmay/thinkremote-rtchub/listener/multiplexer"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/bits"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/h264"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/h265"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/opus"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/wrapper"
	"github.com/thinkonmay/thinkremote-rtchub/util/thread"
)

type udpListener struct {
	closed chan bool

	port        int
	clockRate   float64
	codec       string
	multiplexer *multiplexer.Multiplexer

	pc             net.PacketConn
	controlChannel chan []byte
	addr           net.Addr
}

func UDPListener(codec string) (listener.Listener, error) {
	packetizer := wrapper.PacketizerWrapper{
		Fun:       h264.RTPPay,
		Timestamp: randutil.NewMathRandomGenerator().Uint32(),
		MTU:       1400,
	}

	pipeline := &udpListener{
		codec:       webrtc.MimeTypeH264,
		closed:      make(chan bool, 2),
		clockRate:   90000,
		multiplexer: multiplexer.NewMultiplexer(&packetizer),
	}

	switch codec {
	case "h264":
		packetizer.Fun = h264.RTPPay
		pipeline.codec = webrtc.MimeTypeH264
	case "h265":
		packetizer.Fun = h265.SafariPay
		pipeline.codec = webrtc.MimeTypeH265
	case "opus":
		pipeline.multiplexer = multiplexer.NewMultiplexer(opus.NewOpusPayloader())
		pipeline.codec = webrtc.MimeTypeOpus
	}

	if port, err := getFreePort(); err != nil {
		return nil, err
	} else if pc, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port)); err != nil {
		return nil, err
	} else {
		pipeline.port = port
		pipeline.pc = pc
	}

	_firsttime := true
	firsttime  := func() {
		if _firsttime {
			fmt.Println("capturing udp")
			_firsttime = false
		}
	}

	buf := make([]byte, 1024*1024*10)
	thread.HighPriorityLoop(pipeline.closed, func() {
		if n, addr, err := pipeline.pc.ReadFrom(buf); err != nil {
		} else {
			reader := bits.NewReader(buf[:n])
			sample := reader.ReadUint32()
			pipeline.multiplexer.Send(reader.Left(), sample)
			pipeline.addr = addr
			firsttime()
		}
	})

	ticker := time.NewTicker(time.Second)
	thread.HighPriorityLoop(pipeline.closed, func() {
		select {
		case <-ticker.C:
		case dat := <-pipeline.controlChannel:
			if pipeline.addr != nil {
			} else if _, err := pipeline.pc.WriteTo(dat, pipeline.addr); err != nil {
			}
		}
	})

	return pipeline, nil
}

func (p *udpListener) GetCodec() string {
	return p.codec
}
func (p *udpListener) GetPort() int {
	return p.port
}

func (p *udpListener) Close() {
	thread.TriggerStop(p.closed)
}

func (p *udpListener) RegisterRTPHandler(id string, fun func(pkt *rtp.Packet)) {
	p.multiplexer.RegisterRTPHandler(id, fun)
}

func (p *udpListener) DeregisterRTPHandler(id string) {
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

func (p *udpListener) SendControlMsg(data []byte) {
	p.controlChannel <- data
}
