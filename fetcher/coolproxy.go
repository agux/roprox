package fetcher

import (
	"encoding/json"
	"strconv"

	"github.com/carusyte/roprox/types"
)

//CoolProxy fetches proxy server from https://www.cool-proxy.net/
type CoolProxy struct{}

//UID returns the unique identifier for this spec.
func (f CoolProxy) UID() string {
	return "coolproxy"
}

//Urls return the server urls that provide the free proxy server lists.
func (f CoolProxy) Urls() []string {
	return []string{
		`https://cool-proxy.net/proxies.json`,
	}
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f CoolProxy) UseMasterProxy() bool {
	return true
}

//ParseJSON parses JSON payload and extracts proxy information
func (f CoolProxy) ParseJSON(payload []byte) (ps []*types.ProxyServer) {
	var list []interface{}
	if e := json.Unmarshal(payload, &list); e != nil {
		log.Warnf("%s failed to parse json payload, %+v:\n%s", f.UID(), e, string(payload))
		return
	}
	for i, li := range list {
		var m map[string]interface{}
		var ok bool
		var v interface{}
		var fval float64
		var strVal string
		if m, ok = li.(map[string]interface{}); !ok {
			log.Warnf("%s unable to parse json element at %d: %+v", f.UID(), i, li)
			return
		}
		if v, ok = m["anonymous"]; !ok {
			log.Warnf("%s unable to parse anonymous info at %d: %+v", f.UID(), i, li)
			continue
		}
		if fval, ok = v.(float64); !ok {
			log.Warnf("%s unable to parse anonymous info at %d: %+v", f.UID(), i, li)
			continue
		}
		if fval != 1 {
			//bypassing non-anonymous proxy
			continue
		}
		if v, ok = m["ip"]; !ok {
			log.Warnf("%s unable to parse ip at %d: %+v", f.UID(), i, li)
			continue
		}
		if strVal, ok = v.(string); !ok {
			log.Warnf("%s unable to parse ip at %d: %+v", f.UID(), i, li)
			continue
		}
		host := strVal
		if v, ok = m["port"]; !ok {
			log.Warnf("%s unable to parse port at %d: %+v", f.UID(), i, li)
			continue
		}
		if fval, ok = v.(float64); !ok {
			log.Warnf("%s unable to parse port at %d: %+v", f.UID(), i, li)
			continue
		}
		port := strconv.Itoa(int(fval))
		ps = append(ps, types.NewProxyServer(f.UID(), host, port, "http", ""))
	}
	return
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f CoolProxy) RefreshInterval() int {
	return 30
}
