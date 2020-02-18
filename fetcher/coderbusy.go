package fetcher

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/types"
)

//CoderBusy fetches proxy server from https://proxy.coderbusy.com
type CoderBusy struct{
	defaultFetcherSpec
}

//UID returns the unique identifier for this spec.
func (f CoderBusy) UID() string {
	return "CoderBusy"
}

//Urls return the server urls that provide the free proxy server lists.
func (f CoderBusy) Urls() []string {
	return []string{
		`https://proxy.coderbusy.com/classical/anonymous-type/highanonymous.aspx`,
		`https://proxy.coderbusy.com/classical/anonymous-type/highanonymous.aspx?page=2`,
		`https://proxy.coderbusy.com/classical/anonymous-type/highanonymous.aspx?page=3`,
		`https://proxy.coderbusy.com/classical/anonymous-type/anonymous.aspx`,
		`https://proxy.coderbusy.com/classical/anonymous-type/anonymous.aspx?page=2`,
		`https://proxy.coderbusy.com/classical/anonymous-type/anonymous.aspx?page=3`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f CoderBusy) IsGBK() bool {
	return false
}

//ProxyMode returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f CoderBusy) ProxyMode() types.ProxyMode {
	return types.Direct
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f CoderBusy) ListSelector() []string {
	return []string{
		"#site-app div div div.card-body div table tbody tr",
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f CoderBusy) RefreshInterval() int {
	return 30
}

//ScanItem process each item found in the table determined by ListSelector().
func (f CoderBusy) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	host := strings.TrimSpace(s.Find("td:nth-child(1)").Text())
	port := strings.TrimSpace(s.Find("td.port-box").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http", "")
	return
}
