package network

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/types"
)

var timeout = time.Duration(conf.Args.Probe.Timeout) * time.Second
var maxTimeout = timeout

func httpbin(ps *types.ProxyServer) (yourIp string, e error) {
	url := "https://httpbin.org/ip"
	var res *http.Response
	if res, e = HTTPGet(url, nil, ps, timeout, maxTimeout); e != nil {
		return
	}
	// parse the returned res as JSON in the following format:
	// {
	//		"origin": "14.153.78.250"
	// }
	// for example, the return value `yourIp` shall be `14.153.78.250` in this case.
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		e = err
		return
	}

	var data struct {
		Origin string `json:"origin"`
	}
	if e = json.Unmarshal(body, &data); e != nil {
		return
	}
	yourIp = data.Origin
	return
}

func rebounceIpAsText(url string, ps *types.ProxyServer) (yourIp string, e error) {
	var res *http.Response
	if res, e = HTTPGet(url, nil, ps, timeout, maxTimeout); e != nil {
		return
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		e = err
		return
	}
	bodyStr := string(body)
	// extract IP address
	ipRegex := regexp.MustCompile(`(?:[0-9]{1,3}\.){3}[0-9]{1,3}`)
	ips := ipRegex.FindAllString(bodyStr, -1)
	if len(ips) == 0 {
		// no IP address found. Return as is
		return bodyStr, nil
	}
	yourIp = ips[0]
	return
}

func icanhazip(ps *types.ProxyServer) (yourIp string, e error) {
	return rebounceIpAsText("https://icanhazip.com/", ps)
}

func ipify(ps *types.ProxyServer) (yourIp string, e error) {
	return rebounceIpAsText("https://api.ipify.org", ps)
}

func ifconfigme(ps *types.ProxyServer) (yourIp string, e error) {
	return rebounceIpAsText("https://ifconfig.me/ip", ps)
}

func ipinfo(ps *types.ProxyServer) (yourIp string, e error) {
	return rebounceIpAsText("ipinfo.io/ip", ps)
}

func seeip(ps *types.ProxyServer) (yourIp string, e error) {
	return rebounceIpAsText("https://api.seeip.org", ps)
}

func myexternalip(ps *types.ProxyServer) (yourIp string, e error) {
	return rebounceIpAsText("http://myexternalip.com/raw", ps)
}
