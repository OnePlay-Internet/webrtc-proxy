package grpc

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pion/webrtc/v3"
	"github.com/thinkonmay/thinkremote-rtchub/util/config"
)

func TestClient(t *testing.T) {
	conf := config.GrpcConfig{
		ServerAddress: "grpc.thinkmay.net", // chose domain
		Port:          30000,
	}
	shutdown_channel := make(chan bool)

	client, err := InitGRPCClient(&conf, nil, shutdown_channel)
	if err != nil {
		t.Error(err)
	}
	client.OnSDP(func(i *webrtc.SessionDescription) {
		json, _ := json.Marshal(i)
		fmt.Printf("%s\n", json)
		// shutdown_channel<-true;
	})
	client.OnICE(func(i *webrtc.ICECandidateInit) {
		json, _ := json.Marshal(i)
		fmt.Printf("%s\n", json)
	})
	test := "test"
	var test_num uint16
	test_num = 0

	err = client.SendICE(&webrtc.ICECandidateInit{
		Candidate:     test,
		SDPMid:        &test,
		SDPMLineIndex: &test_num,
	})
	if err != nil {
		t.Error(err)
	}
	err = client.SendSDP(&webrtc.SessionDescription{})
	if err != nil {
		t.Error(err)
	}

	<-shutdown_channel
}
