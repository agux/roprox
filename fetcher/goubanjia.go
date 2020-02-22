package fetcher

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/agux/roprox/types"
)

//GouBanJia fetches proxy server from http://www.goubanjia.com/
type GouBanJia struct{
	defaultFetcherSpec
}

//UID returns the unique identifier for this spec.
func (f GouBanJia) UID() string {
	return "GouBanJia"
}

//Urls return the server urls that provide the free proxy server lists.
func (f GouBanJia) Urls() []string {
	return []string{`http://www.goubanjia.com/`}
}

//IsGBK returns wheter the web page is GBK encoded.
func (f GouBanJia) IsGBK() bool {
	return false
}

//ProxyMode returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f GouBanJia) ProxyMode() types.ProxyMode {
	return types.MasterProxy
}

//ListSelector returns the jQuery selector for searching the proxy server list/table.
func (f GouBanJia) ListSelector() []string {
	return []string{
		"#services div div.row div div div table tbody tr",
	}
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f GouBanJia) RefreshInterval() int {
	return 30
}

//ScanItem process each item found in the table determined by ListSelector().
func (f GouBanJia) ScanItem(i, urlIdx int, s *goquery.Selection) (ps *types.ProxyServer) {
	anon := strings.TrimSpace(s.Find("td:nth-child(2) a").Text())
	if strings.Contains(anon, "透明") {
		return
	}
	vals := make([]string, 0, 16)
	s.Find("td.ip").Children().Each(
		func(i int, s *goquery.Selection) {
			style, ok := s.Attr("style")
			if ok {
				m, e := regexp.MatchString(`.*display\s*:\s*none;?`, style)
				if e != nil {
					log.Error("failed to regexp match", style)
					return
				}
				if m {
					return
				}
			}
			t := strings.TrimSpace(s.Text())
			if len(t) > 0 {
				vals = append(vals, t)
			}
		})
	if len(vals) == 0 {
		return
	}
	host := strings.Join(vals[:len(vals)-1], "")
	port := vals[len(vals)-1]
	ps = types.NewProxyServer(f.UID(), host, port, "http", "")
	return
}
