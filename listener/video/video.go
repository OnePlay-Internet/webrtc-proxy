package video

import (
	"fmt"
	"net"
	"sync"
	"time"
	"unsafe"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/thinkonmay/thinkremote-rtchub/listener"
	"github.com/thinkonmay/thinkremote-rtchub/listener/multiplexer"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/h264"
	"github.com/thinkonmay/thinkremote-rtchub/util/thread"
)

type VideoPipelineC unsafe.Pointer
type VideoPipeline struct {
	closed   chan bool
	pipeline unsafe.Pointer
	mut      *sync.Mutex

	clockRate float64

	codec       string
	Multiplexer *multiplexer.Multiplexer
}

func CreatePipeline() (listener.Listener,
	error) {

	pipeline := &VideoPipeline{
		closed:   make(chan bool, 2),
		pipeline: nil,
		mut:      &sync.Mutex{},
		codec:    webrtc.MimeTypeH264,

		clockRate:   90000,
		Multiplexer: multiplexer.NewMultiplexer("video", h264.NewH264Payloader()),
	}

	pc, err := net.ListenPacket("udp", ":63400")
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 1024*1024*10)
	firsttime := true

	start := time.Now().UnixNano()
	thread.HighPriorityLoop(pipeline.closed, func() {
		n, _, err := pc.ReadFrom(buf)
		if err != nil {
			panic(err)
		}

		pipeline.Multiplexer.Send(buf[:n], uint32(time.Duration(time.Now().UnixNano()-start).Seconds()*pipeline.clockRate))
		start = time.Now().UnixNano()
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
	p.Multiplexer.RegisterRTPHandler(id, fun)
}

func (p *VideoPipeline) DeregisterRTPHandler(id string) {
	p.Multiplexer.DeregisterRTPHandler(id)
}
