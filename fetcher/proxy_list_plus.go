package fetcher

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/agux/roprox/types"
)

//ProxyListPlus fetches proxy server from https://list.proxylistplus.com
type ProxyListPlus struct{
	defaultFetcherSpec
}

//UID returns the unique identifier for this spec.
func (f ProxyListPlus) UID() string {
	return "ProxyListPlus"
}

//Urls return the server urls that provide the free proxy server lists.
func (f ProxyListPlus) Urls() []string {
	return []string{
		`https://list.proxylistplus.com/SSL-List-1`,
		`https://list.proxylistplus.com/SSL-List-2`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f ProxyListPlus) IsGBK() bool {
	return false
}

//ProxyMode returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f ProxyListPlus) ProxyMode() types.ProxyMode {
	return types.MasterProxy
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f ProxyListPlus) ListSelector() []string {
	return []string{
		"#page table.bg tbody tr.cells",
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f ProxyListPlus) RefreshInterval() int {
	return 30
}

//ScanItem process each item found in the table determined by ListSelector().
func (f ProxyListPlus) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	anon := strings.TrimSpace(s.Find("td:nth-child(4)").Text())
	if strings.EqualFold(anon, "transparent") {
		return
	}
	host := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
	port := strings.TrimSpace(s.Find("td:nth-child(3)").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http", "")
	return
}
