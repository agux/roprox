package targets

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/agux/roprox/internal/types"
)

// CNProxy fetches proxy server from http://cn-proxy.com/
type CNProxy struct {
	defaultFetcherSpec
}

// UID returns the unique identifier for this spec.
func (f CNProxy) UID() string {
	return "CNProxy"
}

// Urls return the server urls that provide the free proxy server lists.
func (f CNProxy) Urls() []string {
	return []string{
		`http://cn-proxy.com/`,
		// `http://cn-proxy.com/archives/218`,
	}
}

// IsGBK returns wheter the web page is GBK encoded.
func (f CNProxy) IsGBK() bool {
	return false
}

// ProxyMode returns whether the fetcher needs a master proxy server
// to access the free proxy list provider.
func (f CNProxy) ProxyMode() types.ProxyMode {
	return types.MasterProxy
}

// ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f CNProxy) ListSelector() []string {
	return []string{
		`#w1 table tbody tr`,
		// `#post-218 div.col-mid div.entry-content table:nth-child(8) tbody tr`,
	}
}

// RefreshInterval determines how often the list should be refreshed, in minutes.
func (f CNProxy) RefreshInterval() int {
	return 30
}

// ScanItem process each item found in the table determined by ListSelector().
func (f CNProxy) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	// anon := strings.TrimSpace(s.Find("td:nth-child(3)").Text())
	// if strings.Contains(anon, "透明") {
	// 	return
	// }
	host := strings.TrimSpace(s.Find("td:nth-child(1)").Text())
	port := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
	loc := strings.TrimSpace(s.Find("td:nth-child(3)").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http", loc)
	return
}
