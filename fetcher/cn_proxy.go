package fetcher

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/types"
)

//CNProxy fetches proxy server from http://cn-proxy.com/
type CNProxy struct{}

//UID returns the unique identifier for this spec.
func (f CNProxy) UID() string {
	return "CNProxy"
}

//Urls return the server urls that provide the free proxy server lists.
func (f CNProxy) Urls() []string {
	return []string{
		`http://cn-proxy.com/`,
		`http://cn-proxy.com/archives/218`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f CNProxy) IsGBK() bool {
	return false
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f CNProxy) UseMasterProxy() bool {
	return true
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f CNProxy) ListSelector() []string {
	return []string{
		`#post-4 div div:nth-child(17) table tbody tr`,
		`#tablekit-table-1,#tablekit-table-52 tbody tr`,
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f CNProxy) RefreshInterval() int {
	return 10
}

//ScanItem process each item found in the table determined by ListSelector().
func (f CNProxy) ScanItem(i int, s *goquery.Selection) (ps *types.ProxyServer) {
	anon := strings.TrimSpace(s.Find("td:nth-child(3)").Text())
	if strings.Contains(anon, "透明") {
		return
	}
	host := strings.TrimSpace(s.Find("td:nth-child(1)").Text())
	port := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http")
	return
}
