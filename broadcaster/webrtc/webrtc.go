package webrtc

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/thinkonmay/thinkremote-rtchub/broadcaster"
	"github.com/thinkonmay/thinkremote-rtchub/listener"
	"github.com/thinkonmay/thinkremote-rtchub/util/thread"
)

type WebRTCClient struct {
	conn *webrtc.PeerConnection
	idr  func()
	stop chan bool

	chann chan *rtp.Packet
}

// Configure implements broadcaster.Broadcaster.
func (client *WebRTCClient) Configure(list listener.Listener) error {
	codec := list.GetCodec()
	if track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: codec},
		fmt.Sprintf("%d", time.Now().UnixNano()),
		fmt.Sprintf("%d", time.Now().UnixNano())); err != nil {
	} else if sender, err := client.conn.AddTrack(track); err != nil {
	} else if offer, err := client.conn.CreateOffer(nil); err != nil {
	} else if err = client.conn.SetLocalDescription(offer); err != nil {
	} else {
		thread.SafeThread(func() {
			client.readLoopRTP(track, sender)
		})
	}

	return nil
}

// SendRTPPacket implements broadcaster.Broadcaster.
func (client *WebRTCClient) SendRTPPacket(pkt *rtp.Packet) {
	client.chann <- pkt
}

type WebRTCConfig struct {
	Ices      []webrtc.ICEServer
	Signaling SignalingClient
}

type SignalingMessage struct {
	Event string
	Data  interface{}
}

type SignalingClient interface {
	Send([]byte) error
	Recv() ([]byte, error)
	Close()
}

func InitWebRtcClient(conf WebRTCConfig) (broadcaster.Broadcaster, error) {
	var err error
	client := &WebRTCClient{}
	signaling := conf.Signaling
	if client.conn, err = webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: conf.Ices,
	}); err != nil {
		return nil, err
	}

	client.conn.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
	})
	client.conn.OnICEGatheringStateChange(func(is webrtc.ICEGatheringState) {
	})

	client.conn.OnICECandidate(func(ice *webrtc.ICECandidate) {
		if ice == nil {
			return
		} else if data, err := json.Marshal(ice.ToJSON()); err != nil {
		} else if data, err = json.Marshal(SignalingMessage{
			Event: "candidate",
			Data:  string(data),
		}); err != nil {
		} else if err := signaling.Send(data); err != nil {
		} else {

		}
	})
	thread.SafeLoop(client.stop, time.Second, func() {
		message := SignalingMessage{}
		if raw, err := signaling.Recv(); err != nil {
		} else if err = json.Unmarshal(raw, &message); err != nil {
		} else {
			switch message.Event {
			case "candidate":
				if candidate, ok := message.Data.(webrtc.ICECandidateInit); !ok {
					// todo
				} else if err := client.conn.AddICECandidate(candidate); err != nil {
					break
				}
			case "answer":
				if answer, ok := message.Data.(webrtc.SessionDescription); !ok {
					// todo
				} else if err = client.conn.SetRemoteDescription(answer); err != nil {
					break
				}
			default:
			}
		}
	})

	return client, nil
}

func (client *WebRTCClient) readLoopRTP(
	track *webrtc.TrackLocalStaticRTP,
	sender *webrtc.RTPSender,
) {
	thread.HighPriorityLoop(client.stop, func() {
		select {
		case pkt := <-client.chann:
			if err := track.WriteRTP(pkt); err != nil {
				fmt.Printf("failed to send rtp %s", err.Error())
			}
		case <-client.stop:
			break
		}
	})

	thread.SafeLoop(client.stop, 0, func() {
		if packets, _, err := sender.ReadRTCP(); err != nil {
			fmt.Printf("failed to receive rtcp %s", err.Error())
			time.Sleep(time.Second)
		} else {
			IDR := false
			for _, pkt := range packets {
				switch pkt.(type) {
				case *rtcp.FullIntraRequest:
					IDR = true
				case *rtcp.PictureLossIndication:
					IDR = true
				case *rtcp.TransportLayerNack:
				case *rtcp.ReceiverReport:
				case *rtcp.SenderReport:
				case *rtcp.ExtendedReport:
				}
			}

			if IDR {
				client.idr()
			}
		}
	})

	thread.SafeThread(func() {
		<-client.stop
		thread.TriggerStop(client.stop)
	})
}

func (client *WebRTCClient) Close() {
	client.conn.Close()
	client.stop <- true
}

func (client *WebRTCClient) OnIDR(fun func()) {
	client.idr = fun
}