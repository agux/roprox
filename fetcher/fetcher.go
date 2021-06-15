package fetcher

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"github.com/ssgreg/repeat"

	//shorten type reference

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/network"
	"github.com/agux/roprox/types"
	t "github.com/agux/roprox/types"
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

func fetchDynamicHTML(urlIdx int, url string, chpx chan<- *t.ProxyServer, fspec t.FetcherSpec) (c int) {
	df := fspec.(t.DynamicHTMLFetcher)
	psmap := make(map[string]*t.ProxyServer)

	op := func(rc int) (e error) {
		//clear browser cache
		// if e = network.ClearBrowserCache().Do(ctx); e != nil {
		// 	log.Errorf("#%d %s failed to clear browser cache: %+v", rc, url, e)
		// }

		// create parent context
		o, rpx := allocatorOptions(fspec)
		ctx, c := chromedp.NewExecAllocator(
			context.Background(),
			o...)
		defer c()
		ctx, c = chromedp.NewContext(ctx)
		defer c()
		// navigate
		// create context with homepage-specific timeout
		// ctx, c := context.WithTimeout(parent, time.Duration(df.HomePageTimeout())*time.Second)
		// defer c()
		tm := time.AfterFunc(time.Duration(df.HomePageTimeout())*time.Second, c)
		if e = chromedp.Run(ctx, chromedp.Navigate(url)); e != nil {
			updateProxyScore(fspec, rpx, false)
			//TODO maybe it will not timeout when using a bad proxy, and shows chrome error page instead
			e = errors.Wrapf(e, "#%d failed to navigate %s", rc, url)
			log.Error(e)
			return repeat.HintTemporary(e)
		}
		tm.Stop()
		updateProxyScore(fspec, rpx, true)

		//Do the fetching
		var ps []*t.ProxyServer
		ctx, c = context.WithTimeout(ctx, time.Duration(conf.Args.WebDriver.Timeout)*time.Second)
		defer c()
		if ps, e = df.Fetch(ctx, urlIdx, url); e != nil {
			for _, el := range ps {
				psmap[fmt.Sprintf("%s:%s", el.Host, el.Port)] = el
			}
			stop := false
			if repeat.IsStop(e) {
				stop = true
			}
			e = errors.Wrapf(e, "#%d failed to run webdriver for %+s", rc, url)
			log.Error(e)
			if stop {
				return repeat.HintStop(e)
			}
			return repeat.HintTemporary(e)
		}
		for _, el := range ps {
			psmap[fmt.Sprintf("%s:%s", el.Host, el.Port)] = el
		}
		return
	}

	e := repeat.Repeat(
		repeat.FnWithCounter(op),
		repeat.StopOnSuccess(),
		repeat.LimitMaxTries(fspec.Retry()),
		repeat.WithDelay(
			repeat.FullJitterBackoff(500*time.Millisecond).WithMaxDelay(10*time.Second).Set(),
		),
	)
	if e != nil {
		e = errors.Wrapf(e, "max retry exceeded, giving up: %+s", url)
		log.Error(e)
		return
	}

	for _, p := range psmap {
		chpx <- p
		c++
	}

	return
}

func fetchStaticHTML(urlIdx int, url string, chpx chan<- *t.ProxyServer, fspec t.FetcherSpec) (c int) {
	htmlFetcher := fspec.(t.StaticHTMLFetcher)
	gbk := htmlFetcher.IsGBK()

	selectors := htmlFetcher.ListSelector()
	sel := ""
	res, e := network.HTTPGetResponse(url, nil, fspec.ProxyMode() == types.MasterProxy, true)
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
	if h, e := doc.Html(); e == nil {
		log.Tracef("html returned from %s:\n%s", url, h)
	} else {
		log.Errorf("failed to get html content from %s: %+v", url, e)
		return
	}
	c = 0
	//parse free proxy item
	if urlIdx < len(selectors) {
		sel = selectors[urlIdx]
	} else {
		sel = selectors[len(selectors)-1]
	}
	doc.Find(sel).Each(
		func(i int, s *goquery.Selection) {
			ps := htmlFetcher.ScanItem(i, urlIdx, s)
			if ps != nil {
				chpx <- ps
				c++
			}
		})
	return
}

func fetchJSON(urlIdx int, url string, chpx chan<- *t.ProxyServer, fspec t.FetcherSpec) (c int) {
	res, e := network.HTTPGetResponse(url, nil, fspec.ProxyMode() == types.MasterProxy, true)
	if e != nil {
		log.Errorf("failed to get free proxy list from %s, giving up %+v", url, e)
		return
	}
	defer res.Body.Close()
	payload, e := ioutil.ReadAll(res.Body)
	if e != nil {
		log.Errorf("failed to read html body from %s, giving up %+v", url, e)
		return
	}
	log.Tracef("json returned from %s:\n%s", url, string(payload))
	ps := fspec.(t.JSONFetcher).ParseJSON(payload)
	for _, p := range ps {
		chpx <- p
		c++
	}
	return
}

func fetchPlainText(urlIdx int, url string, chpx chan<- *t.ProxyServer, fspec t.FetcherSpec) (c int) {
	res, e := network.HTTPGetResponse(url, nil, fspec.ProxyMode() == types.MasterProxy, true)
	if e != nil {
		log.Errorf("failed to get free proxy list from %s, giving up %+v", url, e)
		return
	}
	defer res.Body.Close()
	payload, e := ioutil.ReadAll(res.Body)
	if e != nil {
		log.Errorf("failed to read html body from %s, giving up %+v", url, e)
		return
	}
	log.Tracef("plain text returned from %s:\n%s", url, string(payload))
	ps := fspec.(t.PlainTextFetcher).ParsePlainText(payload)
	for _, p := range ps {
		chpx <- p
		c++
	}
	return
}

func fetchFor(urlIdx int, url string, chpx chan<- *t.ProxyServer, fspec t.FetcherSpec) {
	log.Debugf("fetching proxy server from %s", url)
	c := 0
	switch fspec.(type) {
	case t.StaticHTMLFetcher:
		c = fetchStaticHTML(urlIdx, url, chpx, fspec)
	case t.DynamicHTMLFetcher:
		c = fetchDynamicHTML(urlIdx, url, chpx, fspec)
	case t.JSONFetcher:
		c = fetchJSON(urlIdx, url, chpx, fspec)
	case t.PlainTextFetcher:
		c = fetchPlainText(urlIdx, url, chpx, fspec)
	default:
		log.Warnf("unsupported fetcher type: %+v", reflect.TypeOf(fspec))
	}
	log.Infof("%d proxies available from %s", c, url)
}

func updateProxyScore(fspec t.FetcherSpec, rpx *t.ProxyServer, suc bool) {
	if rpx != nil {
		switch fspec.ProxyMode() {
		case t.RotateGlobalProxy:
			network.UpdateProxyScoreGlobal(rpx, false)
		case t.RotateProxy:
			network.UpdateProxyScore(rpx, false)
		}
	}
}
