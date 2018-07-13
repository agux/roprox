package fetcher

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
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
		"#main table tbody tr:nth-child(2)",
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f CoolProxy) RefreshInterval() int {
	return 10
}

//ScanItem process each item found in the table determined by ListSelector().
func (f CoolProxy) ScanItem(i int, s *goquery.Selection) (ps *types.ProxyServer) {
	if s.Size() < 5 {
		//skip promotion row
		return
	}
	host := strings.TrimSpace(s.Find("td:nth-child(1)").Text())
	port := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http")
	return
}
