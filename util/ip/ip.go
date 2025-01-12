package ip

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gortc.io/stun"
)

func GetPublicIP() (string, error) {
	retry := 30
	for i := 0; i < retry; i++ {
		if result := getPublicIPCurl("https://ipv4.icanhazip.com/"); result != "" {
			return result, nil
		} else if result = getPublicIPSTUN(); result != "" {
			return result, nil
		}

		time.Sleep(time.Second)
	}

	return "", fmt.Errorf("timeout 10s fetching public IP")
}

func getPublicIPCurl(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return ""
	}

	if ip, err := io.ReadAll(resp.Body); err != nil {
		return ""
	} else {
		return strings.ReplaceAll(string(ip), "\n", "")
	}
}

func getPublicIPSTUN() (result string) {
	addr := "stun.l.google.com:19302"

	c, err := stun.Dial("udp4", addr)
	if err != nil {
		return ""
	}
	if err = c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
		if res.Error != nil {
			return
		}
		xorAddr := stun.XORMappedAddress{}
		if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
			return
		}

		result = xorAddr.IP.String()
	}); err != nil {
		return ""
	}
	if err := c.Close(); err != nil {
		return ""
	}

	return result
}
