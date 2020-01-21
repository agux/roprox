package fetcher

import (
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/types"
	"github.com/carusyte/roprox/util"
)

//FIXME: get 0 proxy
//CoolProxy fetches proxy server from https://www.cool-proxy.net/
type CoolProxy struct{}

//UID returns the unique identifier for this spec.
func (f CoolProxy) UID() string {
	return "coolproxy"
}

//Urls return the server urls that provide the free proxy server lists.
func (f CoolProxy) Urls() []string {
	return []string{
		`https://www.cool-proxy.net/proxies/http_proxy_list/country_code:/port:/anonymous:1/page:1`,
		`https://www.cool-proxy.net/proxies/http_proxy_list/country_code:/port:/anonymous:1/page:2`,
		`https://www.cool-proxy.net/proxies/http_proxy_list/country_code:/port:/anonymous:1/page:3`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f CoolProxy) IsGBK() bool {
	return false
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f CoolProxy) UseMasterProxy() bool {
	return true
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f CoolProxy) ListSelector() []string {
	return []string{
		"#main table tbody tr",
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f CoolProxy) RefreshInterval() int {
	return 10
}

//ScanItem process each item found in the table determined by ListSelector().
func (f CoolProxy) ScanItem(i int, s *goquery.Selection) (ps *types.ProxyServer) {
	if i < 1 {
		//skip header
		return
	}
	if s.Find("td").Length() < 5 {
		//skip promotion row
		return
	}
	//remove script node
	script := strings.TrimSpace(s.Find("td:nth-child(1) script").Text())
	r := regexp.MustCompile(`str_rot13\("(.*)"\)`).FindStringSubmatch(script)
	host := ""
	if len(r) > 0 {
		hash := util.Rot13(r[len(r)-1])
		hostBytes, err := base64.StdEncoding.DecodeString(hash)
		if err != nil {
			log.Errorf("%s unable to decode base64 host string: %s", f.UID(), hash)
			return
		}
		host = string(hostBytes)
	} else {
		log.Errorf(`%s unable to parse script: %s`, f.UID(), script)
		return
	}
	port := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http")
	return
}
