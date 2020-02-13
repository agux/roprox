package fetcher

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/carusyte/roprox/types"
)

//ProxyFish fetches proxy info from proxyfish.com
type ProxyFish struct {
}

//UID returns the unique identifier for this spec.
func (p ProxyFish) UID() string {
	return "ProxyFish"
}

//Urls return the server urls that provide the free proxy server lists.
func (p ProxyFish) Urls() []string {
	return []string{
		`https://www.proxyfish.com/proxylist/server_processing.php`,
	}
}

//ParseJSON parses JSON payload and extracts proxy information
func (p ProxyFish) ParseJSON(payload []byte) (ps []*types.ProxyServer) {
	js := &proxyFishJSON{}
	if e := json.Unmarshal(payload, js); e != nil {
		log.Warnf("%s failed to parse json: %+v", p.UID(), e)
		return
	}
	dat, e := base64.StdEncoding.DecodeString(js.Data)
	if e != nil {
		log.Warnf("%s failed to decode base64: %+v", p.UID(), e)
		return
	}
	str := string(dat)
	log.Debugf("%s returned data: %s", p.UID(), str)
	// i.e.
	//[["21 minutes ago","124.156.187.59","80","<span class=\"flag-icon
	//flag-icon-cn\"><\/span> China","<div class=\"progress\"><div class=\"progress-bar
	//progress-bar-success\" role=\"progressbar\" aria-valuenow=\"100\" aria-valuemin=\"0\"
	//aria-valuemax=\"100\" style=\"width:100%\"><\/div><\/div>","SOCKS4","Elite"], ...]
	ptn := `\[((?P<time>[^,]*),(?P<ip>[^,]*),(?P<port>[^,]*),(?P<html1>[^,]*),` +
		`(?P<html2>[^,]*),(?P<type>[^,]*),(?P<grade>[^,]*)\])*[,\]]`
	exp := regexp.MustCompile(ptn)
	matches := exp.FindAllStringSubmatch(str, js.RecordsFiltered)
	log.Debugf("%s found matched regex: %+v", p.UID(), matches)
	for _, m := range matches {
		r := make(map[string]string)
		for i, name := range exp.SubexpNames() {
			if i != 0 && name != "" {
				r[name] = strings.Trim(m[i], `"`)
			}
		}
		if t, ok := r["type"]; !ok {
			log.Warnf("%s, unable to extract 'type': %+v", p.UID(), r)
			continue
		} else {
			if !strings.EqualFold("HTTP", t) && !strings.EqualFold("SOCKS5", t) {
				continue
			}
		}
		if g, ok := r["grade"]; !ok {
			log.Warnf("%s, unable to extract 'grade': %+v", p.UID(), r)
			continue
		} else {
			if strings.EqualFold("Transparent", g) {
				continue
			}
		}
		ps = append(ps, types.NewProxyServer(p.UID(), r["ip"], r["port"], strings.ToLower(r["type"]), ""))
	}
	return
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (p ProxyFish) UseMasterProxy() bool {
	return true
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (p ProxyFish) RefreshInterval() int {
	return 10
}

type proxyFishJSON struct {
	Draw            int
	RecordsTotal    int
	RecordsFiltered int
	Data            string
}
