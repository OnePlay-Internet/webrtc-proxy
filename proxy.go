package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	wrtc "github.com/pion/webrtc/v4"
	"github.com/thinkonmay/thinkremote-rtchub/listener"
	"github.com/thinkonmay/thinkremote-rtchub/util/turn"
	"github.com/thinkonmay/thinkremote-rtchub/webrtc"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

type Proxy struct {
	stop chan bool

	keymap map[string]listener.Listener
	mut    *sync.Mutex

	Mux  *http.ServeMux
	turn *turn.TurnServer
}

type WSConn struct {
	conn *websocket.Conn
}

func (conn *WSConn) Send([]byte) error {
	return nil
}
func (conn *WSConn) Recv() ([]byte, error) {
	return []byte{}, nil
}
func (conn *WSConn) Close() {
}

func InitWebRTCProxy(
	mux *http.ServeMux,
	config turn.TurnServerConfig,
) (*Proxy, error) {
	proxy := &Proxy{
		stop: make(chan bool, 2),
	}

	if turn, err := turn.NewTurnServer(config); err != nil {
		return nil, err
	} else {
		proxy.turn = turn
	}

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		turn := turn.TurnRequest{
			Username: uuid.NewString(),
			Password: uuid.NewString(),
		}

		if uids, found := r.Header["credential"]; !found || len(uids) == 0 {
		} else if listener, found := proxy.keymap[uids[0]]; !found {
		} else if conn, err := upgrader.Upgrade(w, r, nil); err != nil {
		} else if err = conn.WriteJSON(webrtc.SignalingMessage{
			Event: "open",
			Data:  turn,
		}); err != nil {
		} else if err := proxy.turn.Open(turn); err != nil {
		} else if webrtcClient, err := webrtc.InitWebRtcClient(webrtc.WebRTCConfig{
			Ices: []wrtc.ICEServer{{
				URLs: []string{fmt.Sprintf("stun:%s:%d", config.PublicIP, config.Port)},
			}, {
				URLs:           []string{fmt.Sprintf("turn:%s:%d", config.PublicIP, config.Port)},
				Username:       turn.Username,
				Credential:     turn.Password,
				CredentialType: wrtc.ICECredentialTypePassword,
			}},
			Signaling: &WSConn{conn: conn},
			Sender:    listener,
		}); err != nil {
		} else {
			defer webrtcClient.Close()
			defer proxy.turn.Close(turn.Username)
			defer conn.Close()

			webrtcClient.HandleIDR(func() {
				listener.SendControlMsg([]byte{})
			})

			typ, dat, err := 0, []byte{}, (error)(nil)
			for err == nil {
				if typ, dat, err = conn.ReadMessage(); err != nil {
				} else if typ != websocket.BinaryMessage {
				} else {
					listener.SendControlMsg(dat)
				}
			}

			conn.WriteJSON(webrtc.SignalingMessage{
				Event: "close",
				Data:  err.Error(),
			})
		}
	})

	mux.HandleFunc("/listeners/new", func(w http.ResponseWriter, r *http.Request) {
		body := struct{ Codec string }{}
		uid := uuid.NewString()
		if dat, err := io.ReadAll(r.Body); err != nil {
		} else if err := json.Unmarshal(dat, &body); err != nil {
		} else if listener, err := UDPListener(body.Codec); err != nil {
		} else {
			proxy.mut.Lock()
			defer proxy.mut.Unlock()

			proxy.keymap[uid] = listener
			w.Write([]byte(uid))
		}
	})

	mux.HandleFunc("/listeners/close", func(w http.ResponseWriter, r *http.Request) {
		proxy.mut.Lock()
		defer proxy.mut.Unlock()

		if data, err := io.ReadAll(r.Body); err != nil {
		} else if listener, found := proxy.keymap[string(data)]; !found {
		} else {
			listener.Close()
		}
	})

	return proxy, nil
}

func (prox *Proxy) Stop() {

}
