package main

import (
	"fmt"
	"os"

	proxy "github.com/OnePlay-Internet/webrtc-proxy"
	"github.com/OnePlay-Internet/webrtc-proxy/listener"
	"github.com/OnePlay-Internet/webrtc-proxy/util/config"
	"github.com/OnePlay-Internet/webrtc-proxy/util/tool"
	"github.com/pion/webrtc/v3"
)

func main() {
	var token string

	Turn :=		"turn:workstation.thinkmay.net:3478";
	TurnUser 	 :=		"oneplay";
	TurnPassword :=		"oneplay";

	dev := tool.GetDevice()
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--token" {
			token = args[i+1]
		} else if arg == "--device" {
			fmt.Printf("%s\n",dev.Monitors[0].Adapter)
		} else if arg == "--help" {
			return
		}
	}

	if token == "" {
		return
	}


	grpc := config.GrpcConfig{
		Port:          30000,
		ServerAddress: "54.169.49.176",
		Token:         token,
	}

	rtc := config.WebRTCConfig{
		Ices: []webrtc.ICEServer{{
			URLs: []string{
				"stun:stun.l.google.com:19302",
			},
		}, {
			URLs: []string{
				"stun:workstation.thinkmay.net:3478",
			},
		}, {
				URLs: []string { Turn },
				Username: TurnUser,
				Credential: TurnPassword,
				CredentialType: webrtc.ICECredentialTypePassword,
			},
		},
	}

	br := []*config.BroadcasterConfig{{
		Name: "audio",
		Codec: webrtc.MimeTypeH264,
	}}

	chans := config.DataChannelConfig{
		Confs: map[string]*config.DataChannel {
			"hid": {
				Send:    make(chan string),
				Recv:    make(chan string),
				Channel: nil,
			},
		},
	}

	Lists := make([]listener.Listener, 0)

	var err error
	var prox *proxy.Proxy;
	if prox,err = proxy.InitWebRTCProxy(nil, &grpc, &rtc, br, &chans, Lists,tool.GetDevice(),
		func(monitor tool.Monitor, soundcard tool.Soundcard) error {
			for _, listener := range Lists {
				conf := listener.GetConfig()
				if conf.StreamID == "video" {
					err := listener.SetSource(&monitor)
					if err != nil {
						return err
					}
				} else if conf.StreamID == "audio" {
					err := listener.SetSource(&soundcard)
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	); err != nil {
		fmt.Printf("failed to init webrtc proxy: %s\n",err.Error())
		return
	}

	<-prox.Shutdown
}
