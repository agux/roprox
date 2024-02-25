package targets

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/logging"
	"github.com/agux/roprox/internal/network"
	"github.com/agux/roprox/internal/types"
	t "github.com/agux/roprox/internal/types"
	"github.com/agux/roprox/internal/ua"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/pkg/errors"
	"github.com/ssgreg/repeat"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var log = logging.Logger

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

func FetchFor(urlIdx int, url string, chpx chan<- *t.ProxyServer, fspec t.FetcherSpec) {
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

func allocatorOptions(fspec types.FetcherSpec) (o []chromedp.ExecAllocatorOption, rpx *types.ProxyServer) {
	proxyMode := fspec.ProxyMode()
	df := fspec.(types.DynamicHTMLFetcher)
	switch proxyMode {
	case types.MasterProxy:
		p := conf.Args.Network.MasterProxyAddr
		log.Debugf("using proxy: %s", p)
		o = append(o, chromedp.ProxyServer(p))
	case types.RotateProxy:
		var e error
		if rpx, e = network.PickProxy(); e != nil {
			log.Fatalf("%s unable to pick rotate proxy: %+v", fspec.UID(), e)
			return
		}
		p := fmt.Sprintf("%s://%s:%s", rpx.Type, rpx.Host, rpx.Port)
		log.Debugf("using proxy: %s", p)
		o = append(o, chromedp.ProxyServer(p))
	}
	if types.Direct != proxyMode {
		if ua, e := ua.PickUserAgent(); e != nil {
			log.Fatalf("failed to pick user agents from the pool: %+v", e)
		} else {
			o = append(o, chromedp.UserAgent(ua))
		}
	}
	if conf.Args.WebDriver.NoImage {
		o = append(o, chromedp.Flag("blink-settings", "imagesEnabled=false"))
	}
	// o = append(o, chromedp.NoFirstRun, chromedp.NoDefaultBrowserCheck)

	for _, opt := range chromedp.DefaultExecAllocatorOptions {
		if reflect.ValueOf(chromedp.Headless).Pointer() == reflect.ValueOf(opt).Pointer() {
			if df.Headless() {
				log.Debug("headless mode is enabled")
			} else {
				log.Debug("ignored headless mode")
				continue
			}
		}
		o = append(o, opt)
	}
	return
}

func updateProxyScore(fspec t.FetcherSpec, rpx *t.ProxyServer, suc bool) {
	if rpx != nil {
		network.UpdateProxyScore(rpx, false)
	}
}

type defaultFetcherSpec struct {
}

func (f defaultFetcherSpec) Retry() int {
	return 0
}

type defaultDynamicHTMLFetcher struct {
}

// HomePageTimeout specifies how many seconds to wait before home page navigation is timed out
func (f defaultDynamicHTMLFetcher) HomePageTimeout() int {
	return 20
}

// waitPageLoaded blocks until a target receives a Page.loadEventFired.
func waitPageLoaded(ctx context.Context) error {
	// TODO: this function is inherently racy, as we don't run ListenTarget
	// until after the navigate action is fired. For example, adding
	// time.Sleep(time.Second) at the top of this body makes most tests hang
	// forever, as they miss the load event.
	//
	// However, setting up the listener before firing the navigate action is
	// also racy, as we might get a load event from a previous navigate.
	//
	// For now, the second race seems much more common in real scenarios, so
	// keep the first approach. Is there a better way to deal with this?
	ch := make(chan struct{})
	lctx, cancel := context.WithCancel(ctx)
	chromedp.ListenTarget(lctx, func(ev interface{}) {
		if _, ok := ev.(*page.EventLoadEventFired); ok {
			cancel()
			close(ch)
		}
	})
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func jsGetText(sel string) (js string) {
	const funcJS = `function getText(sel) {
				var text = [];
				var elements = document.body.querySelectorAll(sel);

				for(var i = 0; i < elements.length; i++) {
					var current = elements[i];
					//if(current.children.length === 0 && current.textContent.replace(/ |\n/g,'') !== '') {
					// Check the element has no children && that it is not empty
					//	text.push(current.textContent + ',');
					//}
					text.push(current.innerText)
				}
				return text
			 };`

	invokeFuncJS := `var a = getText('` + sel + `'); a;`
	js = strings.Join([]string{funcJS, invokeFuncJS}, " ")
	log.Tracef("javascript: %s", js)
	return js
}

func scrollToBottom(ctx context.Context) (e error) {
	var bottom bool
	for i := 1; true; i++ {
		if e = chromedp.Run(ctx,
			chromedp.KeyEvent(kb.End),
		); e != nil {
			return errors.Wrapf(e, "failed to send kb.End key #%d", i)
		}

		log.Debugf("End key sent #%d", i)

		if e = chromedp.Run(ctx,
			chromedp.Evaluate(jsPageBottom(), &bottom),
		); e != nil {
			return errors.Wrapf(e, "failed to check page bottom #%d", i)
		}

		if bottom {
			//found footer
			break
		}

		time.Sleep(time.Millisecond * 500)
	}
	return
}

func jsPageBottom() (js string) {
	js = `
		function isPageBottom() {
			var b;
			try {
				b = (window.innerHeight + window.scrollY) >= document.body.offsetHeight;
			} catch(ex1) {
				try {
					b = (window.innerHeight + window.pageYOffset) >= document.body.offsetHeight - 2;
				} catch(ex2) {
					try {
						b = (window.innerHeight + window.scrollY) >= document.body.scrollHeight;
					} catch(ex3) {
						var scrollTop = (document.documentElement && document.documentElement.scrollTop) || document.body.scrollTop;
						var scrollHeight = (document.documentElement && document.documentElement.scrollHeight) || document.body.scrollHeight;
						b = (scrollTop + window.innerHeight) >= scrollHeight;
					}
				}
			}
			return b;
		};
		var a = isPageBottom(); a;
	`
	log.Tracef("javascript: %s", js)
	return
}
