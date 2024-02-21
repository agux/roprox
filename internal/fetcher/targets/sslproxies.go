package targets

import (
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/agux/roprox/internal/types"
)

// SSLProxies fetches proxy server from https://www.sslproxies.org/
type SSLProxies struct {
	defaultFetcherSpec
}

// UID returns the unique identifier for this spec.
func (f SSLProxies) UID() string {
	return "sslproxies"
}

// Urls return the server urls that provide the free proxy server lists.
func (f SSLProxies) Urls() []string {
	return []string{`https://www.sslproxies.org/`}
}

// IsGBK returns wheter the web page is GBK encoded.
func (f SSLProxies) IsGBK() bool {
	return false
}

// ProxyMode returns whether the fetcher needs a master proxy server
// to access the free proxy list provider.
func (f SSLProxies) ProxyMode() types.ProxyMode {
	return types.MasterProxy
}

// ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f SSLProxies) ListSelector() []string {
	return []string{
		"#proxylisttable tbody tr",
	}
}

// RefreshInterval determines how often the list should be refreshed, in minutes.
func (f SSLProxies) RefreshInterval() int {
	return 30
}

// ScanItem process each item found in the table determined by ListSelector().
func (f SSLProxies) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
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
