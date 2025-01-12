package turn

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pion/turn/v4"
	"github.com/thinkonmay/thinkremote-rtchub/util/thread"
)

const (
	realm     = "thinkmay.net"
	threadNum = 16
)

type TurnServerConfig struct {
	Path     string `json:"path"`
	Port     int    `json:"port"`
	PublicIP string `json:"publicip"`
}
type TurnServer struct {
	sessions map[string][]byte
	mut      *sync.Mutex

	Mux  *http.ServeMux
	turn *turn.Server

	sync bool
}
type TurnRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewTurnServer(config TurnServerConfig) (*TurnServer, error) {
	server := &TurnServer{
		sessions: map[string][]byte{},
		mut:      &sync.Mutex{},
		sync:     true,
	}

	thread.SafeThread(func() {
		_sessions := map[string]string{}
		if bytes, err := os.ReadFile(config.Path); err != nil {
			fmt.Printf("failed to read turn.json file %s\n", err.Error())
		} else if err := json.Unmarshal(bytes, &_sessions); err != nil {
			fmt.Printf("failed to decode turn.json file %s\n", err.Error())
		} else {
			for key, ss := range _sessions {
				if out, err := base64.RawStdEncoding.DecodeString(ss); err != nil {
					fmt.Printf("failed to decode turn cred %s\n", err.Error())
				} else {
					fmt.Printf("Started a new turn server %v\n", key)
					server.sessions[key] = out
				}
			}
		}

		for server.sync {
			time.Sleep(time.Second)

			server.mut.Lock()
			_sessions = map[string]string{}
			for k, ss := range server.sessions {
				_sessions[k] = base64.RawStdEncoding.EncodeToString(ss)
			}
			server.mut.Unlock()

			bytes, err := json.MarshalIndent(_sessions, "", "	")
			if err == nil {
				os.WriteFile(config.Path, bytes, 0777)
			}
		}
	})

	if packetConnConfigs, err := PacketConfig(config.PublicIP, config.Port); err != nil {
		return nil, err
	} else if server.turn, err = turn.NewServer(turn.ServerConfig{
		Realm:             realm,
		PacketConnConfigs: packetConnConfigs,
		AuthHandler: func(username string, realm string, srcAddr net.Addr) ([]byte, bool) {
			server.mut.Lock()
			defer server.mut.Unlock()

			if key, ok := server.sessions[username]; ok {
				return key, true
			}
			return nil, false
		},
	}); err != nil {
		return nil, err
	} else {
		return server, nil
	}
}

func (server *TurnServer) Open(body TurnRequest) error {
	server.mut.Lock()
	defer server.mut.Unlock()

	fmt.Printf("Started a new turn server %v\n", body)
	server.sessions[body.Username] = turn.GenerateAuthKey(body.Username, realm, body.Password)
	return nil
}

func (server *TurnServer) Close(username string) error {
	server.mut.Lock()
	defer server.mut.Unlock()

	if _, ok := server.sessions[username]; ok {
		delete(server.sessions, username)
		return nil
	} else {
		return fmt.Errorf("turn session not found %s", username)
	}
}
