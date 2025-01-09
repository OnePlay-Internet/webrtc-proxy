package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	proxy "github.com/thinkonmay/thinkremote-rtchub"
	"github.com/thinkonmay/thinkremote-rtchub/util/thread"
	"github.com/thinkonmay/thinkremote-rtchub/util/turn"
)

func main() {
	port := 8080
	defaultClients := []struct{ Codec string }{}
	mux := http.NewServeMux()
	config := turn.TurnServerConfig{
		Path:     "./turn.json",
		Port:     3478,
		PublicIP: "",
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
				fmt.Sprintf("http://127.0.0.1:%d", port),
				"application/json",
				bytes.NewReader(data)); err != nil {
			} else if body, err := io.ReadAll(resp.Body); err != nil {
				panic(err)
			} else if resp.StatusCode != 200 {
				panic(body)
			}
		}

		<-stop
	}
}
