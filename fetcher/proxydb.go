package fetcher

import (
	"context"
	"math"
	"net"
	"strings"
	"time"

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/types"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"github.com/ssgreg/repeat"
)

//ProxyDB fetches proxy server from http://proxydb.net/
type ProxyDB struct {
	defaultFetcherSpec
	defaultDynamicHTMLFetcher
}

//FIXME: protected by g-recaptcha

//UID returns the unique identifier for this spec.
func (f ProxyDB) UID() string {
	return "ProxyDB"
}

//Urls return the server urls that provide the free proxy server lists.
func (f ProxyDB) Urls() []string {
	return []string{
		`http://proxydb.net`,
	}
}

//ProxyMode returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f ProxyDB) ProxyMode() types.ProxyMode {
	return types.MasterProxy
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f ProxyDB) RefreshInterval() int {
	return 45
}

//Headless for web driver
func (f ProxyDB) Headless() bool {
	return conf.Args.WebDriver.Headless
}

//Fetch the proxy info.
func (f ProxyDB) Fetch(ctx context.Context, urlIdx int, url string) (ps []*types.ProxyServer, e error) {
	const MaxPage = 20
	var addr, ts, anon, locs []string

	//choose filters and refresh
	if e = chromedp.Run(ctx,
		chromedp.WaitReady(`body > div.container-fluid > form.mb-2`),
		chromedp.WaitReady(`body > div:nth-child(10)`),
		chromedp.Click(`#protocol_http`),
		chromedp.Click(`#protocol_https`),
		chromedp.Click(`#protocol_socks5`),
		chromedp.Click(`#anonlvl_2`),
		chromedp.Click(`#anonlvl_3`),
		chromedp.Click(`#anonlvl_4`),
		chromedp.Click(`body > div.container-fluid > form.mb-2 > button`),
	); e != nil {
		e = errors.Wrap(e, "failed to select filters")
		return
	}

	for i := 0; i < MaxPage; i++ {
		log.Debugf("%s extracting page #%d", f.UID(), i+1)
		if addr, ts, anon, locs, e = f.extract(ctx); e != nil {
			e = errors.Wrapf(e, "failed to extract proxy server at page #%d", i+1)
			return
		}
		log.Debugf("%s parsing page #%d", f.UID(), i+1)
		if list, err := f.parse(addr, ts, anon, locs); err != nil {
			log.Errorf("failed to parse proxy info at page #%d: %+v", i+1, err)
			//try better luck at next page
		} else {
			ps = append(ps, list...)
		}
		var ok bool
		log.Debugf("%s nexting #%d", f.UID(), i+1)
		if ok, e = f.nextPage(ctx); e != nil {
			e = errors.Wrapf(e, "failed to flip page at #%d", i+1)
			return ps, repeat.HintStop(e)
		} else if !ok {
			break
		}
	}
	return
}

func (f ProxyDB) nextPage(ctx context.Context) (next bool, e error) {
	nodes := make([]*cdp.Node, 0, 2)
	//check if "Next Page" button exists
	if e = chromedp.Run(ctx,
		chromedp.Nodes(`#paging-form > nav > ul > button`, &nodes),
	); e != nil {
		e = errors.Wrapf(e, "failed to get next page node")
		return
	}

	if len(nodes) == 0 {
		return false, nil
	}

	nextPageSel := `#paging-form > nav > ul > button`
	nextNode := nodes[0]

	// simulate mouse movement
	var l, t, w, h float64
	if e = chromedp.Run(ctx,
		chromedp.ScrollIntoView(nextPageSel),
		chromedp.JavascriptAttribute(nextPageSel, "offsetLeft", &l),
		chromedp.JavascriptAttribute(nextPageSel, "offsetTop", &t),
		chromedp.JavascriptAttribute(nextPageSel, "offsetWidth", &w),
		chromedp.JavascriptAttribute(nextPageSel, "offsetHeight", &h),
	); e != nil {
		e = errors.Wrapf(e, "failed to get next page button coordinate attributes")
		return
	}
	log.Debugf("next page button rect: %.2f, %.2f, %.2f, %.2f", l, t, w, h)
	x, y, tx, ty := 0., 0., l+w/2., t+h/2.
	dur := 2000.
	st := 100.
	d := math.Max(tx/dur*st, ty/dur*st)
	for x < tx && y < ty {
		if e = chromedp.Run(ctx,
			chromedp.MouseClickXY(x, y, chromedp.ButtonNone),
		); e != nil {
			e = errors.Wrapf(e, "failed move mouse to (%f, %f)", x, y)
			return
		}
		time.Sleep(time.Millisecond * time.Duration(st))
		x, y = math.Min(tx, x+d), math.Min(ty, y+d)
		log.Debugf("mouse: %.2f, %.2f", x, y)
	}

	//get button class
	class := ""
	ok := false
	if e = chromedp.Run(ctx,
		chromedp.AttributeValue(nextPageSel, "class", &class, &ok),
	); e != nil {
		e = errors.Wrapf(e, "failed to get next page button class")
		return
	} else if ok {
		//remove g-recaptcha
		class = strings.TrimSpace(strings.ReplaceAll(class, "g-recaptcha", ""))
		if e = chromedp.Run(ctx,
			chromedp.SetAttributeValue(nextPageSel, "class", class),
		); e != nil {
			e = errors.Wrapf(e, "failed to get next page button class")
			return
		}
	}

	if e = chromedp.Run(ctx,
		// chromedp.MouseClickXY(tx, ty, chromedp.ButtonLeft),
		chromedp.MouseClickNode(nextNode),
		// chromedp.Submit(`#paging-form > nav > ul > button`),
	); e != nil {
		e = errors.Wrapf(e, "failed to submit form for the next page button")
		return
	}

	if e = waitPageLoaded(ctx); e != nil {
		e = errors.Wrapf(e, "failed to wait page load event")
		return
	}

	//double check if last element is loaded
	// if e = chromedp.Run(ctx,
	// 	chromedp.WaitReady(`body > div:nth-child(10)`),
	// ); e != nil {
	// 	e = errors.Wrapf(e, "failed to check page refresh div")
	// 	return
	// }

	return true, nil
}

func (f ProxyDB) extract(ctx context.Context) (i, t, a, l []string, e error) {
	i = make([]string, 0, 4)
	t = make([]string, 0, 4)
	a = make([]string, 0, 4)
	l = make([]string, 0, 4)
	if e = chromedp.Run(ctx,
		chromedp.WaitReady(`body div div.table-responsive table`),
		chromedp.Evaluate(jsGetText(`body > div.container-fluid > div.table-responsive > `+
			`table > tbody > tr:nth-child(n+1) > td:nth-child(1) > a`), &i),
		chromedp.Evaluate(jsGetText(`body > div.container-fluid > div.table-responsive > `+
			`table > tbody > tr:nth-child(n+1) > td:nth-child(3) > abbr`), &l),
		chromedp.Evaluate(jsGetText(`body > div.container-fluid > div.table-responsive > `+
			`table > tbody > tr:nth-child(n+1) > td:nth-child(5)`), &t),
		chromedp.Evaluate(jsGetText(`body > div.container-fluid > div.table-responsive > `+
			`table > tbody > tr:nth-child(n+1) > td:nth-child(6)`), &a),
	); e != nil {
		e = errors.Wrapf(e, "failed to extract proxy info")
	}
	return
}

func (f ProxyDB) parse(addr, ts, anon, locs []string) (ps []*types.ProxyServer, e error) {
	for i, d := range addr {
		if len(anon) <= i {
			break
		}
		if len(locs) <= i {
			break
		}
		if len(ts) <= i {
			break
		}

		a := strings.TrimSpace(anon[i])
		if strings.EqualFold(a, "Transparent") {
			//non anonymous proxy
			continue
		}

		host, port, e := net.SplitHostPort(strings.TrimSpace(d))
		if e != nil {
			log.Warnf("%s possible invalid ip & port string %+v, skipping %+v", f.UID(), d, e)
			continue
		}
		host, port = strings.TrimSpace(host), strings.TrimSpace(port)

		t := strings.ToLower(strings.TrimSpace(ts[i]))
		if strings.Contains(t, "http") {
			t = "http"
		} else if strings.Contains(t, "socks5") {
			t = "socks5"
		} else {
			log.Debugf("%s unsupported proxy type: %+v", f.UID(), t)
			continue
		}

		loc := strings.TrimSpace(locs[i])

		ps = append(ps, types.NewProxyServer(f.UID(), host, port, t, loc))
	}
	return
}
