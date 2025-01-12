package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	proxy "github.com/thinkonmay/thinkremote-rtchub"
	"github.com/thinkonmay/thinkremote-rtchub/util/ip"
	"github.com/thinkonmay/thinkremote-rtchub/util/thread"
	"github.com/thinkonmay/thinkremote-rtchub/util/turn"
)

var (
	defaultClients = []struct{ Codec string }{{
		Codec: "h264",
	}}
)

func main() {
	port := 8080
	mux := http.NewServeMux()
	publicip, err := "", fmt.Errorf("")
	for err != nil {
		publicip, err = ip.GetPublicIP()
	}
	config := turn.TurnServerConfig{
		Path:     "./turn.json",
		Port:     3478,
		PublicIP: publicip,
	}

	if net, err := net.Listen("tcp", fmt.Sprintf(":%d", port)); err != nil {
		panic(err)
	} else if proxy, err := proxy.InitWebRTCProxy(mux, config); err != nil {
		panic(err)
	} else {
		defer proxy.Stop()

		stop := make(chan error)
		thread.SafeThread(func() {
			stop <- http.Serve(net, mux)
		})

		for _, client := range defaultClients {
			if data, err := json.Marshal(client); err != nil {
				panic(err)
			} else if resp, err := http.Post(
				fmt.Sprintf("http://127.0.0.1:%d/listeners/new", port),
				"application/json",
				bytes.NewReader(data)); err != nil {
			} else if body, err := io.ReadAll(resp.Body); err != nil {
				panic(string(body))
			} else if resp.StatusCode != 200 {
				panic(string(body))
			} else {
				fmt.Printf("created new listener %v %s\n", client,string(body))
			}
		}

		<-stop
	}
}
