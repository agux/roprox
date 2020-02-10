package fetcher

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/types"
)

//SpysOne fetches proxy server from http://www.66ip.cn
type SpysOne struct{}

//UID returns the unique identifier for this spec.
func (f SpysOne) UID() string {
	//TODO this website requires dynamic html parsing
	return "SpysOne"
}

//Urls return the server urls that provide the free proxy server lists.
func (f SpysOne) Urls() []string {
	return []string{
		`http://spys.one/en/anonymous-proxy-list/`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f SpysOne) IsGBK() bool {
	return false
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f SpysOne) UseMasterProxy() bool {
	return true
}

//ContentType returns the target url's content type
func (f SpysOne) ContentType() types.ContentType {
	return types.StaticHTML
}

//ParseJSON parses JSON payload and extracts proxy information
func (f SpysOne) ParseJSON(payload []byte) (ps []*types.ProxyServer) {
	panic("not json proxy site")
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f SpysOne) ListSelector() []string {
	return []string{
		`body table:nth-child(3) tbody tr:nth-child(5) td table tbody tr[class^='spy1x']`,
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f SpysOne) RefreshInterval() int {
	return 10
}

//ScanItem process each item found in the table determined by ListSelector().
func (f SpysOne) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	if i == 0 {
		//skip header
		return
	}

	proxyInfo := strings.TrimSpace(s.Find("td:nth-child(1) font").Text())

	log.Debugf("%s extracted proxy info: %s", f.UID(), proxyInfo)

	// ptn := `^(?P<ip>.*)document\.write\("[^"]*"(?P<port_code>.*\)\))`
	ptn := `^(?P<ip>.*)document\.write\("[^"]*"(?P<port_code>[0-9a-zA-Z\^\+\(\)]*\))\)`
	exp := regexp.MustCompile(ptn)
	match := exp.FindStringSubmatch(proxyInfo)
	log.Debugf("%s found matched regex: %+v", f.UID(), match)
	r := make(map[string]string)
	for i, name := range exp.SubexpNames() {
		if i != 0 && name != "" {
			r[name] = match[i]
		}
	}
	log.Debugf("%s extracted map: %+v", f.UID(), r)
	return
	// return types.NewProxyServer(p.UID(), r["ip"], r["port"], strings.ToLower(r["type"])))
}
