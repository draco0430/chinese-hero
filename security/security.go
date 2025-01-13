package security

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
)

var (
	RemoteAddrs    = make(map[string]int)
	BannedIPs      = make(map[string]int)
	BannedIPsMutex sync.RWMutex
)

type PlayerStatus struct {
	Launcher bool `json:"launcher"`
}

func CheckPlayer(ip string) bool {
	data := url.Values{
		"ip":  {ip},
		"key": {"BBB"},
	}
	resp, err := http.PostForm("http://138.201.158.26:1338/launcher-available", data) // 80.240.28.136
	if err != nil {
		// WTFFF ==
		fmt.Println(err)
		return false
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return false
	}

	var tmpData PlayerStatus

	err = json.Unmarshal(body, &tmpData)
	if err != nil {
		fmt.Println(err)
		return false
	}

	if !tmpData.Launcher {
		return false
	}

	return true
}
