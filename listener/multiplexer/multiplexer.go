package multiplexer

import (
	"sync"

	"github.com/pion/rtp"
	"github.com/thinkonmay/thinkremote-rtchub/listener/rtppay"
	"github.com/thinkonmay/thinkremote-rtchub/util/thread"
)

const (
	queue_size = 1024
)

type sample struct {
	data    []byte
	samples uint32
	id      int
}
type Multiplexer struct {
	packetizer rtppay.Packetizer

	mutex   *sync.Mutex
	queue   chan *sample
	handler map[string]*Handler
}

type Handler struct {
	handler func(*rtp.Packet)
	buffer  chan *rtp.Packet
	stop    chan bool
}

func NewMultiplexer(packetizer rtppay.Packetizer) *Multiplexer {
	ret := &Multiplexer{
		mutex:      &sync.Mutex{},
		queue:      make(chan *sample, queue_size),
		handler:    map[string]*Handler{},
		packetizer: packetizer,
	}

	return ret
}

func (ret *Multiplexer) Send(Buff []byte, Samples uint32) {
	packets := ret.packetizer.Packetize(Buff, Samples)
	ret.mutex.Lock()
	defer ret.mutex.Unlock()
	for _, handler := range ret.handler {
		for _, p := range packets {
			handler.buffer <- p
		}
	}
}

func (p *Multiplexer) Close() {
	keys := make([]string, 0, len(p.handler))
	for k := range p.handler {
		keys = append(keys, k)
	}

	for _, v := range keys {
		p.DeregisterRTPHandler(v)
	}
}

func (p *Multiplexer) RegisterRTPHandler(id string, fun func(pkt *rtp.Packet)) {
	handler := &Handler{
		handler: fun,
		buffer:  make(chan *rtp.Packet, queue_size),
		stop:    make(chan bool, 2),
	}

	thread.HighPriorityLoop(handler.stop, func() {
		select {
		case buffer := <-handler.buffer:
			handler.handler(buffer)
		case <-handler.stop:
			thread.TriggerStop(handler.stop)
		}
	})

	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.handler[id] = handler
}

func (p *Multiplexer) DeregisterRTPHandler(id string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if handler, found := p.handler[id]; found {
		thread.TriggerStop(handler.stop)
		delete(p.handler, id)
	}
}
