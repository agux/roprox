package targets

import (
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/agux/roprox/internal/types"
)

// FreeProxyCZ fetches proxy server from http://free-proxy.cz
type FreeProxyCZ struct {
	defaultFetcherSpec
}

// UID returns the unique identifier for this spec.
func (f FreeProxyCZ) UID() string {
	return "FreeProxyCZ"
}

// Urls return the server urls that provide the free proxy server lists.
func (f FreeProxyCZ) Urls() []string {
	return []string{
		`http://free-proxy.cz/en/`,
		`http://free-proxy.cz/en/proxylist/main/2`,
		`http://free-proxy.cz/en/proxylist/main/3`,
		`http://free-proxy.cz/en/proxylist/main/4`,
		`http://free-proxy.cz/en/proxylist/main/5`,
		`http://free-proxy.cz/en/proxylist/country/CN/all/ping/all`,
		`http://free-proxy.cz/en/proxylist/country/CN/all/ping/all/2`,
		`http://free-proxy.cz/en/proxylist/country/CN/all/ping/all/3`,
		`http://free-proxy.cz/en/proxylist/country/CN/all/ping/all/4`,
		`http://free-proxy.cz/en/proxylist/country/CN/all/ping/all/5`,
	}
}

// IsGBK returns wheter the web page is GBK encoded.
func (f FreeProxyCZ) IsGBK() bool {
	return false
}

// ProxyMode returns whether the fetcher needs a master proxy server
// to access the free proxy list provider.
func (f FreeProxyCZ) ProxyMode() types.ProxyMode {
	return types.MasterProxy
}

// ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f FreeProxyCZ) ListSelector() []string {
	return []string{
		"#proxy_list tbody tr",
	}
}

// RefreshInterval determines how often the list should be refreshed, in minutes.
func (f FreeProxyCZ) RefreshInterval() int {
	return 30
}

// ScanItem process each item found in the table determined by ListSelector().
func (f FreeProxyCZ) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	if s.Find("td").Length() < 5 {
		//skip promotion row
		return
	}
	anon := strings.TrimSpace(s.Find("td:nth-child(7) small").Text())
	if strings.EqualFold(anon, "transparent") {
		return
	}
	script := strings.TrimSpace(s.Find("td:nth-child(1) script").Text())
	r := regexp.MustCompile(`Base64\.decode\("(.*)"\)`).FindStringSubmatch(script)
	host := ""
	if len(r) > 0 {
		hash := r[len(r)-1]
		hostBytes, err := base64.StdEncoding.DecodeString(hash)
		if err != nil {
			log.Errorf("%s unable to decode base64 host string: %s", f.UID(), hash)
			return
		}
		host = string(hostBytes)
	} else {
		log.Errorf(`%s unable to parse script: %s`, f.UID(), script)
		return
	}
	port := strings.TrimSpace(s.Find("td:nth-child(2) span").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http", "")
	return
}
