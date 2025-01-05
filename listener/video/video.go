package video

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/pion/randutil"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	proxy "github.com/thinkonmay/thinkremote-rtchub"
	"github.com/thinkonmay/thinkremote-rtchub/listener"
	"github.com/thinkonmay/thinkremote-rtchub/listener/multiplexer"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/h264"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay/wrapper"
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

func CreatePipeline(queue *proxy.Queue) (listener.Listener,
	error) {

	pipeline := &VideoPipeline{
		closed:   make(chan bool, 2),
		pipeline: nil,
		mut:      &sync.Mutex{},
		codec:    webrtc.MimeTypeH264,

		clockRate: 90000,
		Multiplexer: multiplexer.NewMultiplexer("video", &wrapper.PacketizerWrapper{
			Fun:       h264.RTPPay,
			Timestamp: randutil.NewMathRandomGenerator().Uint32(),
			MTU:       1400,
		}),
	}

	buffer := make([]byte, 1024*1024) //1MB
	local_index := queue.CurrentIndex()
	firsttime := true
	thread.HighPriorityLoop(pipeline.closed, func() {
		for local_index >= queue.CurrentIndex() {
			time.Sleep(time.Microsecond * 100)
		}

		local_index++
		if size, duration := queue.Copy(buffer, local_index); size > len(buffer) {
		} else {
			pipeline.Multiplexer.Send(buffer[:size], uint32(time.Duration(duration).Seconds()*pipeline.clockRate))
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
	p.Multiplexer.RegisterRTPHandler(id, fun)
}

func (p *VideoPipeline) DeregisterRTPHandler(id string) {
	p.Multiplexer.DeregisterRTPHandler(id)
}
