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

type Session struct {
	RemoteAddr string `json:"remoteAddr"`
}

type Proxy struct {
	stop chan bool

	sessions []*Session
	keymap   map[string]listener.Listener
	mut      *sync.Mutex

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
		stop:     make(chan bool, 2),
		keymap:   map[string]listener.Listener{},
		mut:      &sync.Mutex{},
		sessions: []*Session{},
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
			w.WriteHeader(400)
			w.Write([]byte("no credential header available"))
		} else if listener, found := proxy.keymap[uids[0]]; !found {
			w.WriteHeader(400)
			w.Write([]byte("invalid key"))
		} else if conn, err := upgrader.Upgrade(w, r, nil); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else if err = conn.WriteJSON(webrtc.SignalingMessage{
			Event: "open",
			Data:  turn,
		}); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else if err := proxy.turn.Open(turn); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
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
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else {
			defer webrtcClient.Close()
			defer proxy.turn.Close(turn.Username)
			defer conn.Close()

			proxy.sessions = append(proxy.sessions, &Session{
				RemoteAddr: r.RemoteAddr,
			})

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
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else if err := json.Unmarshal(dat, &body); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else if listener, err := UDPListener(body.Codec); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else if data, err := json.Marshal(map[string]interface{}{
			"port": listener.GetPort(),
			"codec": listener.GetCodec(),
		}); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else {
			proxy.mut.Lock()
			defer proxy.mut.Unlock()

			proxy.keymap[uid] = listener
			w.WriteHeader(200)
			w.Write(data)
		}
	})

	mux.HandleFunc("/listeners/close", func(w http.ResponseWriter, r *http.Request) {
		proxy.mut.Lock()
		defer proxy.mut.Unlock()
		if data, err := io.ReadAll(r.Body); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else if listener, found := proxy.keymap[string(data)]; !found {
			w.WriteHeader(400)
			w.Write([]byte("listener not found"))
		} else {
			listener.Close()
			delete(proxy.keymap, string(data))
		}
	})

	mux.HandleFunc("/sessions", func(w http.ResponseWriter, r *http.Request) {
		if data, err := json.Marshal(proxy.sessions); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(200)
			w.Write(data)
		}
	})

	return proxy, nil
}

func (prox *Proxy) Stop() {
	prox.mut.Lock()
	defer prox.mut.Unlock()

	for _, prox := range prox.keymap {
		prox.Close()
	}
}
