package fetcher

import (
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/agux/roprox/types"
)

//FreeProxyList fetches proxy server from https://free-proxy-list.net/
type FreeProxyList struct{
	defaultFetcherSpec
}

//UID returns the unique identifier for this spec.
func (f FreeProxyList) UID() string {
	return "FreeProxyList"
}

//Urls return the server urls that provide the free proxy server lists.
func (f FreeProxyList) Urls() []string {
	return []string{
		`https://free-proxy-list.net/`,
		`https://free-proxy-list.net/anonymous-proxy.html`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f FreeProxyList) IsGBK() bool {
	return false
}

//ProxyMode returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f FreeProxyList) ProxyMode() types.ProxyMode {
	return types.MasterProxy
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f FreeProxyList) ListSelector() []string {
	return []string{
		"#proxylisttable tbody tr",
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f FreeProxyList) RefreshInterval() int {
	return 30
}

//ScanItem process each item found in the table determined by ListSelector().
func (f FreeProxyList) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	lchk := strings.TrimSpace(s.Find("td:nth-child(8)").Text())
	if strings.HasSuffix(lchk, "minutes ago") {
		m := lchk[:strings.Index(lchk, " ")]
		if i, e := strconv.ParseInt(m, 10, 64); e == nil {
			if int(i) > 50 {
				return
			}
		} else {
			log.Errorf("failed to parse proxy last check string: %s, %+v", m, e)
			return
		}
	}
	anon := strings.TrimSpace(s.Find("td:nth-child(5)").Text())
	if strings.EqualFold(anon, `transparent`) {
		return
	}
	host := strings.TrimSpace(s.Find("td:nth-child(1)").Text())
	port := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http", "")
	return
}
