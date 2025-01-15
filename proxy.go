package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	wrtc "github.com/pion/webrtc/v4"
	"github.com/thinkonmay/thinkremote-rtchub/broadcaster"
	"github.com/thinkonmay/thinkremote-rtchub/broadcaster/udp"
	"github.com/thinkonmay/thinkremote-rtchub/broadcaster/webrtc"
	"github.com/thinkonmay/thinkremote-rtchub/listener"
	"github.com/thinkonmay/thinkremote-rtchub/util/turn"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

type Session struct {
	broadcaster.Broadcaster `json:"-"`
	RemoteAddr              string `json:"remoteAddr"`
}

type Proxy struct {
	stop chan bool

	sessions         map[string]*Session
	lsmap, listeners map[string]listener.Listener
	mut              *sync.Mutex

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
		stop:      make(chan bool, 2),
		listeners: map[string]listener.Listener{},
		lsmap:     map[string]listener.Listener{},
		sessions:  map[string]*Session{},
		mut:       &sync.Mutex{},
	}

	if turn, err := turn.NewTurnServer(config); err != nil {
		return nil, err
	} else {
		proxy.turn = turn
	}

	mux.HandleFunc("/broadcasters/register", func(w http.ResponseWriter, r *http.Request) {
		turn := turn.TurnRequest{
			Username: uuid.NewString(),
			Password: uuid.NewString(),
		}

		if uids, found := r.Header[http.CanonicalHeaderKey("cred")]; !found || len(uids) == 0 {
			w.WriteHeader(400)
			w.Write([]byte("no credential header available"))
		} else if listener, found := proxy.listeners[uids[0]]; !found {
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
		} else if broadcaster, err := webrtc.InitWebRtcClient(webrtc.WebRTCConfig{
			Ices: []wrtc.ICEServer{{
				URLs: []string{fmt.Sprintf("stun:%s:%d", config.PublicIP, config.Port)},
			}, {
				URLs:           []string{fmt.Sprintf("turn:%s:%d", config.PublicIP, config.Port)},
				Username:       turn.Username,
				Credential:     turn.Password,
				CredentialType: wrtc.ICECredentialTypePassword,
			}},
			Signaling: &WSConn{conn: conn},
		}); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else if err := broadcaster.Configure(listener); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else {
			defer broadcaster.Close()
			defer proxy.turn.Close(turn.Username)
			defer conn.Close()
			id := listener.RegisterRTPHandler(func(p *rtp.Packet) {
				broadcaster.SendRTPPacket(p)
			})
			defer listener.DeregisterRTPHandler(id)

			broadcaster.OnIDR(func() {
				listener.SendControlMsg([]byte{})
			})
			proxy.sessions[id] = &Session{
				Broadcaster: broadcaster,
				RemoteAddr:  r.RemoteAddr,
			}

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

	mux.HandleFunc("/broadcasters/new", func(w http.ResponseWriter, r *http.Request) {
		body := struct{ Addr, Id string }{}
		if dat, err := io.ReadAll(r.Body); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else if err := json.Unmarshal(dat, &body); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else if broadcaster, err := udp.NewUDPSource(body.Addr); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else {
			proxy.mut.Lock()
			defer proxy.mut.Unlock()

			listener := proxy.listeners[body.Id]

			broadcaster.OnIDR(func() {
				listener.SendControlMsg([]byte{})
			})

			id := listener.RegisterRTPHandler(func(p *rtp.Packet) {
				broadcaster.SendRTPPacket(p)
			})

			proxy.lsmap[id] = listener
			proxy.sessions[id] = &Session{
				RemoteAddr:  body.Addr,
				Broadcaster: broadcaster,
			}

			w.WriteHeader(200)
			w.Write([]byte(id))
		}
	})

	mux.HandleFunc("/broadcasters/close", func(w http.ResponseWriter, r *http.Request) {
		proxy.mut.Lock()
		defer proxy.mut.Unlock()
		if data, err := io.ReadAll(r.Body); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else if session, found := proxy.sessions[string(data)]; !found {
			w.WriteHeader(400)
			w.Write([]byte("listener not found"))
		} else if listener, found := proxy.lsmap[string(data)]; !found {
			w.WriteHeader(400)
			w.Write([]byte("listener map not found"))
		} else {
			session.Close()
			listener.DeregisterRTPHandler(string(data))

			delete(proxy.lsmap, string(data))
			delete(proxy.listeners, string(data))
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
			"port":  listener.GetPort(),
			"codec": listener.GetCodec(),
		}); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		} else {
			proxy.mut.Lock()
			defer proxy.mut.Unlock()

			proxy.listeners[uid] = listener
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
		} else if listener, found := proxy.listeners[string(data)]; !found {
			w.WriteHeader(400)
			w.Write([]byte("listener not found"))
		} else {
			listener.Close()
			delete(proxy.listeners, string(data))
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

	for _, prox := range prox.listeners {
		prox.Close()
	}
}
