package fetcher

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/agux/roprox/types"
)

//Data5u fetches proxy server from http://www.data5u.com
type Data5u struct{
	defaultFetcherSpec
}

//UID returns the unique identifier for this spec.
func (f Data5u) UID() string {
	return "Data5u"
}

//Urls return the server urls that provide the free proxy server lists.
func (f Data5u) Urls() []string {
	return []string{
		`http://www.data5u.com`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f Data5u) IsGBK() bool {
	return false
}

//ProxyMode returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f Data5u) ProxyMode() types.ProxyMode {
	return types.Direct
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f Data5u) ListSelector() []string {
	return []string{
		`ul.l2`,
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f Data5u) RefreshInterval() int {
	return 30
}

//ScanItem process each item found in the table determined by ListSelector().
func (f Data5u) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	if i == 0 {
		//skip header
		return
	}
	anon := strings.TrimSpace(s.Find("span:nth-child(3) li").Text())
	if strings.Contains(anon, `透明`) || len(anon) == 0 {
		return
	}
	host := strings.TrimSpace(s.Find("span:nth-child(1) li").Text())
	port := strings.TrimSpace(s.Find("span:nth-child(2) li").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http", "")
	return
}
