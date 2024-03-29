package targets

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/agux/roprox/internal/types"
)

// Xroxy fetches proxy server from https://www.xroxy.com
type Xroxy struct {
	defaultFetcherSpec
}

// UID returns the unique identifier for this spec.
func (f Xroxy) UID() string {
	return "xroxy"
}

// Urls return the server urls that provide the free proxy server lists.
func (f Xroxy) Urls() []string {
	return []string{
		`https://www.xroxy.com/proxylist.php?port=&type=Not_transparent&ssl=&country=&latency=&reliability=#table`,
		`https://www.xroxy.com/proxylist.php?port=&type=Not_transparent&ssl=&country=&latency=&reliability=&sort=reliability&desc=true&pnum=1#table`,
		`https://www.xroxy.com/proxylist.php?port=&type=Not_transparent&ssl=&country=&latency=&reliability=&sort=reliability&desc=true&pnum=2#table`,
		`https://www.xroxy.com/proxylist.php?port=&type=Not_transparent&ssl=&country=&latency=&reliability=&sort=reliability&desc=true&pnum=3#table`,
		`https://www.xroxy.com/proxylist.php?port=&type=Not_transparent&ssl=&country=&latency=&reliability=&sort=reliability&desc=true&pnum=4#table`,
		`https://www.xroxy.com/proxylist.php?port=&type=Socks5&ssl=&country=&latency=&reliability=#table`,
		`https://www.xroxy.com/proxylist.php?port=&type=Socks5&ssl=&country=&latency=&reliability=&sort=reliability&desc=true&pnum=1#table`,
		`https://www.xroxy.com/proxylist.php?port=&type=Socks5&ssl=&country=&latency=&reliability=&sort=reliability&desc=true&pnum=2#table`,
	}
}

// IsGBK returns wheter the web page is GBK encoded.
func (f Xroxy) IsGBK() bool {
	return false
}

// ProxyMode returns whether the fetcher needs a master proxy server
// to access the free proxy list provider.
func (f Xroxy) ProxyMode() types.ProxyMode {
	return types.MasterProxy
}

// ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f Xroxy) ListSelector() []string {
	return []string{
		"#content table:nth-child(8) tbody tr",
	}
}

// RefreshInterval determines how often the list should be refreshed, in minutes.
func (f Xroxy) RefreshInterval() int {
	return 30
}

// ScanItem process each item found in the table determined by ListSelector().
func (f Xroxy) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	if s.Find("td").Length() < 5 {
		//skip promotion row
		return
	}
	stype := strings.TrimSpace(s.Find("td:nth-child(3) a").Text())
	pstype := "http"
	if strings.EqualFold(stype, "transparent") {
		return
	} else if strings.EqualFold(stype, "socks5") {
		pstype = "socks5"
	}
	host := strings.TrimSpace(s.Find("td:nth-child(1) a").Text())
	if len(host) <= 0 {
		return
	}
	port := strings.TrimSpace(s.Find("td:nth-child(2) a").Text())
	loc := strings.TrimSpace(s.Find("td:nth-child(5) a").Text())
	if len(host) <= 0 || len(port) <= 0 {
		return
	}
	ps = types.NewProxyServer(f.UID(), host, port, pstype, loc)
	return
}
