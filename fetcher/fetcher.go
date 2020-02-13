package fetcher

import (
	"context"
	"io"
	"io/ioutil"
	"reflect"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"github.com/ssgreg/repeat"

	//shorten type reference

	"github.com/carusyte/roprox/conf"
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

func fetchDynamicHTML(urlIdx int, url string, chpx chan<- *t.ProxyServer, fspec t.FetcherSpec) (c int) {
	useMasterProxy := fspec.UseMasterProxy()
	// o := chromedp.DefaultExecAllocatorOptions[:]
	var o []chromedp.ExecAllocatorOption
	if useMasterProxy {
		o = append(o, chromedp.ProxyServer("socks5://localhost:1080"))
	}
	if conf.Args.WebDriver.NoImage {
		o = append(o, chromedp.Flag("blink-settings", "imagesEnabled=false"))
	}
	for _, opt := range chromedp.DefaultExecAllocatorOptions {
		if reflect.ValueOf(chromedp.Headless).Pointer() == reflect.ValueOf(opt).Pointer() &&
			!conf.Args.WebDriver.Headless {
			log.Debug("ignored headless mode")
			continue
		}
		o = append(o, opt)
	}

	var ps []*t.ProxyServer

	op := func(rc int) (e error) {
		var (
			ctx        context.Context
			c1, c2, c3 context.CancelFunc
		)
		defer func() {
			if c1 != nil {
				c1()
			}
			if c2 != nil {
				c2()
			}
			if c3 != nil {
				c3()
			}
		}()
		// create context
		ctx, c1 = context.WithTimeout(context.Background(), time.Duration(conf.Args.WebDriver.Timeout)*time.Second)
		ctx, c2 = chromedp.NewExecAllocator(ctx, o...)
		ctx, c3 = chromedp.NewContext(ctx)

		// navigate
		if e = chromedp.Run(ctx, chromedp.Navigate(url)); e != nil {
			e = errors.Wrapf(e, "#%d failed to run webdriver for %+s", rc, url)
			log.Error(e)
			return repeat.HintTemporary(e)
		}
		if ps, e = fspec.(t.DynamicHTMLFetcher).Fetch(ctx, urlIdx, url); e != nil {
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
		return
	}

	e := repeat.Repeat(
		repeat.FnWithCounter(op),
		repeat.StopOnSuccess(),
		repeat.LimitMaxTries(conf.Args.WebDriver.MaxRetry),
		repeat.WithDelay(
			repeat.FullJitterBackoff(500*time.Millisecond).WithMaxDelay(10*time.Second).Set(),
		),
	)
	if e != nil {
		e = errors.Wrapf(e, "max retry exceeded, giving up: %+s", url)
		log.Error(e)
		return
	}

	for _, p := range ps {
		chpx <- p
		c++
	}

	return
}

func fetchStaticHTML(urlIdx int, url string, chpx chan<- *t.ProxyServer, fspec t.FetcherSpec) (c int) {
	htmlFetcher := fspec.(t.StaticHTMLFetcher)
	gbk := htmlFetcher.IsGBK()
	useMasterProxy := fspec.UseMasterProxy()

	selectors := htmlFetcher.ListSelector()
	sel := ""
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
	useMasterProxy := fspec.UseMasterProxy()
	res, e := util.HTTPGetResponse(url, nil, useMasterProxy, true)
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
	default:
		log.Warnf("unsupported fetcher type: %+v", reflect.TypeOf(fspec))
	}
	log.Infof("%d proxies available from %s", c, url)
}
