package fetcher

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/types"
)

//HinkyDink fetches proxy server from http://www.mrhinkydink.com
type HinkyDink struct{}

//UID returns the unique identifier for this spec.
func (f HinkyDink) UID() string {
	return "HinkyDink"
}

//Urls return the server urls that provide the free proxy server lists.
func (f HinkyDink) Urls() []string {
	return []string{
		`http://www.mrhinkydink.com/proxies.htm`,
		`http://www.mrhinkydink.com/proxies2.htm`,
		`http://www.mrhinkydink.com/proxies3.htm`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f HinkyDink) IsGBK() bool {
	return false
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f HinkyDink) UseMasterProxy() bool {
	return false
}

//ContentType returns the target url's content type
func (f HinkyDink) ContentType() types.ContentType{
	return types.StaticHTML
}
//ParseJSON parses JSON payload and extracts proxy information
func (f HinkyDink) ParseJSON(payload []byte) (ps []*types.ProxyServer){
	return
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f HinkyDink) ListSelector() []string {
	return []string{
		`body table:nth-child(2) tbody tr:nth-child(2) td:nth-child(3) table tbody tr td table tbody tr[bgcolor="#88ff88"],tr[bgcolor="#ffff88"]`,
		`body table:nth-child(2) tbody tr:nth-child(2) td:nth-child(3) table tbody tr td b table tbody tr[bgcolor="#88ff88"],tr[bgcolor="#ffff88"]`,
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f HinkyDink) RefreshInterval() int {
	return 10
}

//ScanItem process each item found in the table determined by ListSelector().
func (f HinkyDink) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	anon := strings.TrimSpace(s.Find("td:nth-child(3)").Text())
	if strings.EqualFold(anon, `transparent`) {
		return
	}
	host := strings.TrimSpace(s.Find("td:nth-child(1)").Text())
	host = strings.TrimRight(host, `*`)
	port := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http")
	return
}
