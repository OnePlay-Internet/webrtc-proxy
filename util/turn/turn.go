package turn

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
			fmt.Printf("failed to read turn.json file %s", err.Error())
		} else if err := json.Unmarshal(bytes, &_sessions); err != nil {
			fmt.Printf("failed to decode turn.json file %s", err.Error())
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

	mux := http.NewServeMux()
	mux.HandleFunc("/open", func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
		}

		body := TurnRequest{}
		if err := json.Unmarshal(data, &body); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}

		server.mut.Lock()
		defer server.mut.Unlock()

		fmt.Printf("Started a new turn server %v\n", body)
		server.sessions[body.Username] = turn.GenerateAuthKey(body.Username, realm, body.Password)

		w.WriteHeader(200)
		w.Write([]byte("success"))
	})

	mux.HandleFunc("/close", func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}

		body := struct {
			Username string `json:"username"`
		}{}
		if err := json.Unmarshal(data, &body); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}

		server.mut.Lock()
		defer server.mut.Unlock()

		if _, ok := server.sessions[body.Username]; ok {
			delete(server.sessions, body.Username)
			w.WriteHeader(200)
			w.Write([]byte("success"))
		} else {
			w.WriteHeader(404)
			w.Write([]byte(fmt.Sprintf("username %s not found", body.Username)))
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
		server.Mux = mux
		return server, nil
	}
}

type TurnClient struct {
	Addr string
}

func (client *TurnClient) Open(req TurnRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		fmt.Sprintf("http://%s/open", client.Addr),
		"application/json",
		bytes.NewReader(data))
	if err != nil {
		return err
	} else if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("%s", string(body))
	}

	return nil
}

func (client *TurnClient) Close(username string) error {
	data, err := json.Marshal(TurnRequest{
		Username: username,
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(
		fmt.Sprintf("http://%s/close", client.Addr),
		"application/json",
		bytes.NewReader(data))
	if err != nil {
		return err
	} else if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("%s", string(body))
	}

	return nil
}
