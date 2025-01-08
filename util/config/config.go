package config

type WebRTCConfig struct {
}

type WebsocketConfig struct {
	Port          int
	ServerAddress string
}

type AuthConfig struct {
	Token string `json:"token"`
}
type GrpcConfig struct {
	Audio struct {
		GrpcPort int    `json:"GrpcPort"`
		URL      string `json:"URL"`
	} `json:"Audio"`
	Video struct {
		GrpcPort int    `json:"GrpcPort"`
		URL      string `json:"URL"`
	} `json:"Video"`

	ServerAddress string `json:"HostName"`
}

type ListenerConfig struct {
	ID       string
	StreamID string
	Codec    string
}

type BroadcasterConfig struct {
	Name  string
	Codec string
}
