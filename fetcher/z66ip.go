package fetcher

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/types"
)

//Z66IP fetches proxy server from http://www.66ip.cn
type Z66IP struct{}

//UID returns the unique identifier for this spec.
func (f Z66IP) UID() string {
	return "66ip"
}

//Urls return the server urls that provide the free proxy server lists.
func (f Z66IP) Urls() []string {
	//FIXME seems no available proxy most of the time
	return []string{
		`http://www.66ip.cn/1.html`,
		`http://www.66ip.cn/2.html`,
		`http://www.66ip.cn/3.html`,
		`http://www.66ip.cn/4.html`,
		`http://www.66ip.cn/5.html`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f Z66IP) IsGBK() bool {
	return true
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f Z66IP) UseMasterProxy() bool {
	return false
}

//ContentType returns the target url's content type
func (f Z66IP) ContentType() types.ContentType {
	return types.StaticHTML
}

//ParseJSON parses JSON payload and extracts proxy information
func (f Z66IP) ParseJSON(payload []byte) (ps []*types.ProxyServer) {
	return
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f Z66IP) ListSelector() []string {
	return []string{
		`#main div div:nth-child(1) table tbody tr`,
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f Z66IP) RefreshInterval() int {
	return 10
}

//ScanItem process each item found in the table determined by ListSelector().
func (f Z66IP) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	if i == 0 {
		//skip header
		return
	}
	anon := strings.TrimSpace(s.Find("td:nth-child(4)").Text())
	if strings.Contains(anon, `透明`) {
		return
	}
	host := strings.TrimSpace(s.Find("td:nth-child(1)").Text())
	port := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
	if "0" == port {
		//invalid port
		return
	}
	ps = types.NewProxyServer(f.UID(), host, port, "http")
	return
}
