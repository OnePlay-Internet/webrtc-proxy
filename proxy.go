package proxy

import (
	"fmt"
	"time"

	webrtclib "github.com/pion/webrtc/v3"
	"github.com/thinkonmay/thinkremote-rtchub/listener"
	"github.com/thinkonmay/thinkremote-rtchub/signalling"
	"github.com/thinkonmay/thinkremote-rtchub/util/config"
	"github.com/thinkonmay/thinkremote-rtchub/webrtc"
)

type Proxy struct {
	listeners []listener.Listener

	chan_conf *config.DataChannelConfig

	signallingClient signalling.Signalling
	webrtcClient     *webrtc.WebRTCClient

	Shutdown chan bool
}

func InitWebRTCProxy(grpc_conf signalling.Signalling,
					webrtc_conf *config.WebRTCConfig,
					chan_conf *config.DataChannelConfig,
					lis []listener.Listener,
					onTrack webrtc.OnTrackFunc,
					) (proxy *Proxy, err error) {
	fmt.Printf("started proxy\n")
	proxy = &Proxy{
		Shutdown:  make(chan bool),
		chan_conf: chan_conf,
		signallingClient: grpc_conf,
		listeners: lis,
	}

	go proxy.handleTimeout()
	if proxy.webrtcClient, err = webrtc.InitWebRtcClient(onTrack, *webrtc_conf); err != nil { 
		return 
	}

	go func() { for {
			state := proxy.webrtcClient.GatherStateChange()
			switch state {
			case webrtclib.ICEGathererStateGathering:
			case webrtclib.ICEGathererStateComplete:
			case webrtclib.ICEGathererStateClosed:
			}
		}
	}()
	go func() { for {
			state := proxy.webrtcClient.ConnectionStateChange()
			switch state {
			case webrtclib.ICEConnectionStateConnected:
				proxy.webrtcClient.Listen(proxy.listeners)
			case webrtclib.ICEConnectionStateClosed:
			case webrtclib.ICEConnectionStateFailed:
			case webrtclib.ICEConnectionStateDisconnected:
				proxy.Stop()
			}
		}
	}()

	go func() { for { proxy.signallingClient.SendICE(proxy.webrtcClient.OnLocalICE()) } }()
	go func() { for { proxy.signallingClient.SendSDP(proxy.webrtcClient.OnLocalSDP()) } }()
	proxy.signallingClient.OnICE(func(i *webrtclib.ICECandidateInit) { proxy.webrtcClient.OnIncomingICE(i) })
	proxy.signallingClient.OnSDP(func(i *webrtclib.SessionDescription) { proxy.webrtcClient.OnIncominSDP(i) })
	return
}

func (proxy *Proxy) handleTimeout() {
	start := make(chan bool, 2)
	go func() {
		proxy.signallingClient.WaitForEnd()
		fmt.Println("application ended exchanging signaling message")
		start <-true
	}()
	go func() {
		proxy.signallingClient.WaitForStart()
		fmt.Println("application start exchanging signaling message")
		proxy.webrtcClient.RegisterDataChannel(proxy.chan_conf)
		time.Sleep(30 * time.Second)
		start <-false 
	}()

	success := <-start
	if !success {
		fmt.Println("application exchange signaling timeout, closing")
		proxy.Stop()
	}
}

func (prox *Proxy) Stop() {
	prox.webrtcClient.Close()
	prox.signallingClient.Stop()
	prox.Shutdown <- true
}
