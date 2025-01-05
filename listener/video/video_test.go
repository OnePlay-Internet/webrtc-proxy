package video

import (
	"fmt"
	"testing"
	"time"

	"github.com/pion/rtp"
)

func TestUDPListener(t *testing.T) {
	pipeline, err := CreatePipeline()
	if err != nil {
		panic(err)
	}

	pipeline.RegisterRTPHandler("basdf", func(p *rtp.Packet) {
		fmt.Printf("%v\n", len(p.Payload))
	})

	time.Sleep(time.Minute)
}
