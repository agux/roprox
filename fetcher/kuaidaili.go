package fetcher

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/types"
)

//KuaiDaiLi fetches proxy server from https://www.kuaidaili.com
type KuaiDaiLi struct{}

//UID returns the unique identifier for this spec.
func (f KuaiDaiLi) UID() string {
	return "KuaiDaiLi"
}

//Urls return the server urls that provide the free proxy server lists.
func (f KuaiDaiLi) Urls() []string {
	return []string{
		`https://www.kuaidaili.com/ops/proxylist/1/`,
		`https://www.kuaidaili.com/ops/proxylist/2/`,
		`https://www.kuaidaili.com/ops/proxylist/3/`,
	}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f KuaiDaiLi) IsGBK() bool {
	return false
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f KuaiDaiLi) UseMasterProxy() bool {
	return false
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f KuaiDaiLi) ListSelector() []string {
	return []string{
		"#freelist table tbody tr",
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f KuaiDaiLi) RefreshInterval() int {
	return 10
}

//ScanItem process each item found in the table determined by ListSelector().
func (f KuaiDaiLi) ScanItem(i int, s *goquery.Selection) (ps *types.ProxyServer) {
	lchk := strings.TrimSpace(s.Find("td:nth-child(8)").Text())
	anon := strings.TrimSpace(s.Find("td:nth-child(3)").Text())
	if strings.HasSuffix(lchk, "小时前") || strings.Contains(anon, `透明`) {
		return
	}
	host := strings.TrimSpace(s.Find("td:nth-child(1)").Text())
	port := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
	ps = types.NewProxyServer(f.UID(), host, port, "http")
	return
}
