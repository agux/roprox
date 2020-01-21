package fetcher

import (
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/types"
)

//ProxyListOrg fetches proxy server from https://proxy-list.org
type ProxyListOrg struct{}

//UID returns the unique identifier for this spec.
func (f ProxyListOrg) UID() string {
	return "ProxyListOrg"
}

//Urls return the server urls that provide the free proxy server lists.
func (f ProxyListOrg) Urls() []string {
	return []string{
		`https://proxy-list.org/english/search.php?search=anonymous-and-elite&country=any&type=anonymous-and-elite&port=any&ssl=any`,
		`https://proxy-list.org/english/search.php?search=anonymous-and-elite&country=any&type=anonymous-and-elite&port=any&ssl=any&p=2`,
		`https://proxy-list.org/english/search.php?search=anonymous-and-elite&country=any&type=anonymous-and-elite&port=any&ssl=any&p=3`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f ProxyListOrg) IsGBK() bool {
	return false
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f ProxyListOrg) UseMasterProxy() bool {
	return true
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f ProxyListOrg) ListSelector() []string {
	return []string{
		"#proxy-table div.table-wrap div ul",
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f ProxyListOrg) RefreshInterval() int {
	return 10
}

//ScanItem process each item found in the table determined by ListSelector().
func (f ProxyListOrg) ScanItem(i int, s *goquery.Selection) (ps *types.ProxyServer) {
	anon := strings.TrimSpace(s.Find("li.type strong").Text())
	if strings.EqualFold(anon, "transparent") {
		return
	}

	script := strings.TrimSpace(s.Find("li.proxy script").Text())
	r := regexp.MustCompile(`Proxy\('(.*)'\)`).FindStringSubmatch(script)
	val := ""
	if len(r) > 0 {
		hash := r[len(r)-1]
		hostBytes, err := base64.StdEncoding.DecodeString(hash)
		if err != nil {
			log.Errorf("%s unable to decode base64 host string: %s", f.UID(), hash)
			return
		}
		val = string(hostBytes)
	} else {
		log.Errorf(`%s unable to parse script: %s`, f.UID(), script)
		return
	}

	strs := strings.Split(val, ":")
	if len(strs) != 2 {
		log.Errorf("unable to parse host:port string: %s", val)
		return
	}
	host := strs[0]
	port := strs[1]
	ps = types.NewProxyServer(f.UID(), host, port, "http")
	return
}
