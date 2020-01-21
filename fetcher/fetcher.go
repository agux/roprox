package fetcher

import (
	"io"

	"github.com/PuerkitoBio/goquery"

	//shorten type reference
	t "github.com/carusyte/roprox/types"
	"github.com/carusyte/roprox/util"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

//Fetch proxy server information using the specified fetcher specification,
//and output to the channel.
func Fetch(chpx chan<- *t.ProxyServer, fspec t.FetcherSpec) {
	urls := fspec.Urls()
	for i, url := range urls {
		fetchFor(i, url, chpx, fspec)
	}
}

func fetchFor(i int, url string, chpx chan<- *t.ProxyServer, fspec t.FetcherSpec) {
	gbk := fspec.IsGBK()
	useMasterProxy := fspec.UseMasterProxy()
	selectors := fspec.ListSelector()
	sel := ""

	log.Infof("fetching proxy server from %s", url)
	res, e := util.HTTPGetResponse(url, nil, useMasterProxy, true)
	if e != nil {
		log.Errorf("failed to get free proxy list from %s, giving up %+v", url, e)
		return
	}
	defer res.Body.Close()
	var body io.Reader
	body = res.Body
	// parse body using goquery
	if gbk {
		// Convert the designated charset HTML to utf-8 encoded HTML.
		body = transform.NewReader(body, simplifiedchinese.GBK.NewDecoder())
	}
	doc, e := goquery.NewDocumentFromReader(body)
	if e != nil {
		log.Errorf("failed to read response body from %s: %+v", url, e)
		return
	}
	count := 0
	//parse free proxy item
	if i < len(selectors) {
		sel = selectors[i]
	} else {
		sel = selectors[len(selectors)-1]
	}

	doc.Find(sel).Each(
		func(i int, s *goquery.Selection) {
			ps := fspec.ScanItem(i, s)
			if ps != nil {
				chpx <- ps
				count++
			}
		})
	log.Infof("%d proxies available from %s", count, url)
}
