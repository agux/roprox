package fetcher

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/types"
)

//GatherProxy fetches proxy server from http://www.gatherproxy.com
type GatherProxy struct{}

//TODO: need web driver to parse dynamic content

//UID returns the unique identifier for this spec.
func (f GatherProxy) UID() string {
	return "GatherProxy"
}

//Urls return the server urls that provide the free proxy server lists.
func (f GatherProxy) Urls() []string {
	return []string{
		`http://www.gatherproxy.com/proxylist/anonymity/?t=Elite`,
		`http://www.gatherproxy.com/proxylist/anonymity/?t=Anonymous`,
		`http://www.gatherproxy.com/proxylist/country/?c=China`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f GatherProxy) IsGBK() bool {
	return false
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f GatherProxy) UseMasterProxy() bool {
	return true
}

//ContentType returns the target url's content type
func (f GatherProxy) ContentType() types.ContentType{
	return types.StaticHTML
}
//ParseJSON parses JSON payload and extracts proxy information
func (f GatherProxy) ParseJSON(payload []byte) (ps []*types.ProxyServer){
	return
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f GatherProxy) ListSelector() []string {
	return []string{
		// "#tblproxy tbody tr",
		"#tblproxy tbody script",
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f GatherProxy) RefreshInterval() int {
	return 10
}

//ScanItem process each item found in the table determined by ListSelector().
func (f GatherProxy) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	js := strings.TrimSpace(s.Text())
	log.Tracef("found script: %s", js)
	rx :=
		`"PROXY_IP":"([0-9\.]*)",.*` +
			`"PROXY_PORT":"([0-9ABCDEF]*)",.*`
	r := regexp.MustCompile(rx).FindStringSubmatch(js)
	if len(r) < 3 {
		log.Warnf("unable to parse js for %s: %s", f.UID(), js)
		return
	}
	host := strings.TrimSpace(r[1])
	port, e := strconv.ParseInt(strings.TrimSpace(r[2]), 16, 64)
	if e != nil {
		log.Warnf("%s unable to parse proxy port from hex: %s", f.UID(), r[2])
	}
	ps = types.NewProxyServer(f.UID(), host, strconv.Itoa(int(port)), "http")
	return
}
