package proxy

import (
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	wbrtc "github.com/pion/webrtc/v4"
	"github.com/thinkonmay/thinkremote-rtchub/listener"
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

func InitWebRTCProxy() (*Proxy, error) {
	proxy := &Proxy{
		stop: make(chan bool, 2),
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		uid := ""
		if listener, found := proxy.keymap[uid]; !found {
		} else if conn, err := upgrader.Upgrade(w, r, nil); err != nil {
		} else if webrtcClient, err := webrtc.InitWebRtcClient(webrtc.WebRTCConfig{
			Ices:      []wbrtc.ICEServer{},
			Signaling: &WSConn{conn: conn},
			Sender:    listener,
		}); err != nil {
		} else {
			webrtcClient.IDRhandler = func() {
				listener.SendControlMsg([]byte{})
			}
		}
	})

	http.HandleFunc("/listeners/new", func(w http.ResponseWriter, r *http.Request) {
		uid := uuid.NewString()
		if listener, err := UDPListener("h264"); err != nil {
		} else {
			proxy.mut.Lock()
			defer proxy.mut.Unlock()

			proxy.keymap[uid] = listener
		}
	})

	http.HandleFunc("/listeners/close", func(w http.ResponseWriter, r *http.Request) {
		uid := ""
		proxy.mut.Lock()
		defer proxy.mut.Unlock()
		if listener, found := proxy.keymap[uid]; !found {
		} else {
			listener.Close()
		}
	})

	return proxy, nil
}

func (prox *Proxy) Stop() {
}
