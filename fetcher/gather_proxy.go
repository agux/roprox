package fetcher

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/types"
)

//GatherProxy fetches proxy server from http://www.gatherproxy.com
type GatherProxy struct{}

//UID returns the unique identifier for this spec.
func (f GatherProxy) UID() string {
	return "GatherProxy"
}

//Urls return the server urls that provide the free proxy server lists.
func (f GatherProxy) Urls() []string {
	return []string{
		`http://www.gatherproxy.com/proxylist/anonymity/?t=Elite`,
		`http://www.gatherproxy.com/proxylist/anonymity/?t=Anonymous`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f GatherProxy) IsGBK() bool {
	return false
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f GatherProxy) UseMasterProxy() bool {
	return true
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f GatherProxy) ListSelector() []string {
	return []string{
		"#tblproxy tbody tr",
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f GatherProxy) RefreshInterval() int {
	return 10
}

//ScanItem process each item found in the table determined by ListSelector().
func (f GatherProxy) ScanItem(i int, s *goquery.Selection) (ps *types.ProxyServer) {
	if i < 2 {
		//skip headers
		return
	}
	anon := strings.TrimSpace(s.Find("td:nth-child(4)").Text())
	if strings.EqualFold(anon, "transparent") {
		return
	}
	host := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
	port := strings.TrimSpace(s.Find("td:nth-child(3) a").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http")
	return
}
