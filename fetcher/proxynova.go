package fetcher

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/agux/roprox/types"
)

//ProxyNova fetches proxy server from https://www.proxynova.com/
type ProxyNova struct {
	defaultFetcherSpec
}

//UID returns the unique identifier for this spec.
func (f ProxyNova) UID() string {
	return "ProxyNova"
}

//Urls return the server urls that provide the free proxy server lists.
func (f ProxyNova) Urls() []string {
	return []string{
		`https://www.proxynova.com/proxy-server-list/elite-proxies/`,
		`https://www.proxynova.com/proxy-server-list/anonymous-proxies/`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f ProxyNova) IsGBK() bool {
	return false
}

//ProxyMode returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f ProxyNova) ProxyMode() types.ProxyMode {
	return types.MasterProxy
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f ProxyNova) ListSelector() []string {
	return []string{
		`#tbl_proxy_list > tbody:nth-child(2) > tr[data-proxy-id]`,
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f ProxyNova) RefreshInterval() int {
	return 60
}

//ScanItem process each item found in the table determined by ListSelector().
func (f ProxyNova) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	// host := strings.TrimSpace(s.Find("td:nth-child(1) > abbr").AttrOr("title", ""))
	hostStr := strings.TrimSpace(s.Find("td:nth-child(1) > abbr").Text())
	exp := regexp.MustCompile(`(?P<IP>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)
	host := exp.FindString(hostStr)
	if host == "" {
		log.Warnf("[%s] unable to extract IP from text: %s", f.Urls()[urlIdx], hostStr)
		return
	}
	port := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
	loc := strings.TrimSpace(s.Find("td:nth-child(6) > a").Text())
	loc = strings.ReplaceAll(loc, "\n", "")
	exp = regexp.MustCompile(`\s{2,}`)
	loc = exp.ReplaceAllString(loc, " ")
	ps = types.NewProxyServer(f.UID(), host, port, "http", loc)
	return
}
