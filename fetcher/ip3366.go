package fetcher

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/types"
)

//IP3366 fetches proxy server from http://www.ip3366.net
type IP3366 struct{
	defaultFetcherSpec
}

//UID returns the unique identifier for this spec.
func (f IP3366) UID() string {
	return "IP3366"
}

//Urls return the server urls that provide the free proxy server lists.
func (f IP3366) Urls() []string {
	return []string{
		`http://www.ip3366.net/free/?stype=1`,
		`http://www.ip3366.net/free/?stype=1&page=2`,
		`http://www.ip3366.net/free/?stype=1&page=3`,
		`http://www.ip3366.net/free/?stype=1&page=4`,
		`http://www.ip3366.net/free/?stype=1&page=5`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f IP3366) IsGBK() bool {
	return true
}

//ProxyMode returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f IP3366) ProxyMode() types.ProxyMode {
	return types.Direct
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f IP3366) ListSelector() []string {
	return []string{
		`#list table tbody tr`,
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f IP3366) RefreshInterval() int {
	return 30
}

//ScanItem process each item found in the table determined by ListSelector().
func (f IP3366) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	anon := strings.TrimSpace(s.Find("td:nth-child(3)").Text())
	if strings.Contains(anon, `透明`) {
		return
	}
	host := strings.TrimSpace(s.Find("td:nth-child(1)").Text())
	port := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http", "")
	return
}
