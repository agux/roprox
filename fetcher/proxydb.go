package fetcher

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/types"
)

//ProxyDB fetches proxy server from http://proxydb.net/
type ProxyDB struct{}

//UID returns the unique identifier for this spec.
func (f ProxyDB) UID() string {
	return "ProxyDB"
}

//Urls return the server urls that provide the free proxy server lists.
func (f ProxyDB) Urls() []string {
	return []string{
		`http://proxydb.net/`,
		`http://proxydb.net/?offset=15`,
		`http://proxydb.net/?offset=30`,
		`http://proxydb.net/?offset=45`,
		`http://proxydb.net/?offset=60`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f ProxyDB) IsGBK() bool {
	return false
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f ProxyDB) UseMasterProxy() bool {
	return false
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f ProxyDB) ListSelector() []string {
	return []string{
		"body div div.table-responsive table tbody tr",
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f ProxyDB) RefreshInterval() int {
	return 10
}

//ScanItem process each item found in the table determined by ListSelector().
func (f ProxyDB) ScanItem(i int, s *goquery.Selection) (ps *types.ProxyServer) {
	anon := strings.TrimSpace(s.Find("td:nth-child(6) span").Text())
	if strings.EqualFold("Transparent", anon) {
		return
	}
	str := strings.TrimSpace(s.Find("td:nth-child(1) a").Text())
	vals := strings.Split(str, ":")
	if len(vals) != 2 {
		return
	}
	host := strings.TrimSpace(vals[0])
	port := strings.TrimSpace(vals[1])
	ps = types.NewProxyServer(f.UID(), host, port, "http")
	return
}
