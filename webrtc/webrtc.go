package webrtc

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/thinkonmay/thinkremote-rtchub/listener"
	"github.com/thinkonmay/thinkremote-rtchub/util/thread"
)

type WebRTCClient struct {
	conn       *webrtc.PeerConnection
	IDRhandler func()
	stop       chan bool
}

type WebRTCConfig struct {
	Ices      []webrtc.ICEServer
	Signaling SignalingClient
	Sender    listener.Listener
}

type SignalingMessage struct {
	Event string
	Data  string
}

type SignalingClient interface {
	Send([]byte) error
	Recv() ([]byte, error)
	Close()
}

func InitWebRtcClient(conf WebRTCConfig) (client *WebRTCClient, err error) {
	client = &WebRTCClient{}
	signaling := conf.Signaling
	if client.conn, err = webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: conf.Ices,
	}); err != nil {
		return
	}

	thread.SafeThread(func() {
		codec := conf.Sender.GetCodec()
		if track, err := webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{MimeType: codec},
			fmt.Sprintf("%d", time.Now().UnixNano()),
			fmt.Sprintf("%d", time.Now().UnixNano())); err != nil {
		} else if err != nil {
		} else if sender, err := client.conn.AddTrack(track); err != nil {
		} else {
			client.readLoopRTP(conf.Sender, track, sender)
		}

		if offer, err := client.conn.CreateOffer(nil); err != nil {
		} else if err = client.conn.SetLocalDescription(offer); err != nil {
		} else {
		}
	})

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
			candidate := webrtc.ICECandidateInit{}
			answer := webrtc.SessionDescription{}
			switch message.Event {
			case "candidate":
				if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
					break
				} else if err := client.conn.AddICECandidate(candidate); err != nil {
					break
				}
			case "answer":
				if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
					break
				} else if err = client.conn.SetRemoteDescription(answer); err != nil {
					break
				}
			default:
			}
		}
	})
	return
}

func (client *WebRTCClient) readLoopRTP(
	listener listener.Listener,
	track *webrtc.TrackLocalStaticRTP,
	sender *webrtc.RTPSender,
) {

	id := track.ID()
	listener.RegisterRTPHandler(id, func(pk *rtp.Packet) {
		if err := track.WriteRTP(pk); err != nil {
			fmt.Printf("failed to send rtp %s", err.Error())
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
				client.IDRhandler()
			}
		}
	})

	thread.SafeThread(func() {
		<-client.stop
		thread.TriggerStop(client.stop)
		listener.DeregisterRTPHandler(id)
	})
}

func (client *WebRTCClient) Close() {
	client.conn.Close()
	client.stop <- true
}
