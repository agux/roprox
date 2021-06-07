package fetcher

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/types"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"github.com/ssgreg/repeat"
)

//HideMyName fetches proxy info from this web
type HideMyName struct {
	defaultDynamicHTMLFetcher
}

//UID returns the unique identifier for this spec.
func (f HideMyName) UID() string {
	return "HideMyName"
}

func (f HideMyName) Retry() int {
	return conf.Args.WebDriver.MaxRetry
}

func (f HideMyName) HomePageTimeout() int {
	return conf.Args.DataSource.HideMyName.HomePageTimeout
}

//Urls return the server urls that provide the free proxy server lists.
func (f HideMyName) Urls() []string {
	return []string{
		`https://hidemy.name/en/proxy-list/?type=hs5&anon=234`,
	}
}

//ProxyMode returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f HideMyName) ProxyMode() types.ProxyMode {
	return types.ProxyMode(conf.Args.DataSource.HideMyName.ProxyMode)
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f HideMyName) RefreshInterval() int {
	return conf.Args.DataSource.HideMyName.RefreshInterval
}

// func (f HideMyName) extract(ctx context.Context) (i, p, a, t, l []string, e error) {
// 	i = make([]string, 0, 4)
// 	p = make([]string, 0, 4)
// 	a = make([]string, 0, 4)
// 	t = make([]string, 0, 4)
// 	l = make([]string, 0, 4)
// 	if e = chromedp.Run(ctx,
// 		//body > div.wrap > div.services_proxylist.services > div > div.table_block > table > tbody > tr > td:nth-child(1)
// 		chromedp.Evaluate(jsGetText(`table.proxy__t > tbody > tr > td:nth-child(1)`), &i),
// 		chromedp.Evaluate(jsGetText(`table.proxy__t > tbody > tr > td:nth-child(2)`), &p),
// 		chromedp.Evaluate(jsGetText(`table.proxy__t > tbody > tr > td:nth-child(3)`), &l),
// 		chromedp.Evaluate(jsGetText(`table.proxy__t > tbody > tr > td:nth-child(5)`), &t),
// 		chromedp.Evaluate(jsGetText(`table.proxy__t > tbody > tr > td:nth-child(6)`), &a),
// 	); e != nil {
// 		e = errors.Wrap(e, "failed to extract proxy info")
// 	}
// 	return
// }

func (f HideMyName) extract(html string) (i, p, a, t, l []string, e error) {
	r := strings.NewReader(html)
	doc, e := goquery.NewDocumentFromReader(r)
	if e != nil {
		log.Errorf("failed to create goquery reader from %s: %+v", html, e)
		return
	}
	// ret, e := doc.Html()
	// log.Debugf("document html:\n%s", ret)
	//traverse the table rows
	doc.Find(`body div.wrap div.services_proxylist.services div div.table_block table tbody tr`).Each(
		func(idx int, s *goquery.Selection) {
			ip := strings.TrimSpace(s.Find(`td:nth-child(1)`).Text())
			port := strings.TrimSpace(s.Find(`td:nth-child(2)`).Text())
			loc := strings.TrimSpace(s.Find(`td:nth-child(3)`).Text())
			tp := strings.TrimSpace(s.Find(`td:nth-child(5)`).Text())
			anon := strings.TrimSpace(s.Find(`td:nth-child(6)`).Text())
			//drop the following proxy
			if len(ip) == 0 || len(port) == 0 ||
				!(strings.Contains(tp, "HTTP") || strings.Contains(tp, "SOCKS5")) ||
				strings.EqualFold(anon, "no") {
				return
			}

			i = append(i, ip)
			p = append(p, port)
			a = append(a, anon)
			t = append(t, tp)
			l = append(l, loc)
		})

	return
}

func (f HideMyName) parse(ips, ports, anon, ts, locs []string) (ps []*types.ProxyServer) {
	for i, ip := range ips {
		if len(ports) <= i {
			break
		}
		if len(anon) <= i {
			break
		}
		if len(ts) <= i {
			break
		}
		if len(locs) <= i {
			break
		}

		if strings.EqualFold(strings.TrimSpace(anon[i]), "No") {
			continue
		}

		if strings.EqualFold(ts[i], "SOCKS4") {
			return
		}

		t := "http"
		if strings.Contains(ts[i], "SOCKS5") {
			t = "socks5"
		}

		ip = strings.TrimSpace(ip)
		port := strings.TrimSpace((ports[i]))
		loc := strings.TrimSpace((locs[i]))
		ps = append(ps, types.NewProxyServer(f.UID(), ip, port, t, loc))
	}
	return
}

//Headless for web driver
func (f HideMyName) Headless() bool {
	return conf.Args.DataSource.HideMyName.Headless
}

//Fetch the proxy info
func (f HideMyName) Fetch(ctx context.Context, urlIdx int, url string) (ps []*types.ProxyServer, e error) {
	// var rect *dom.Rect
	// if rect, e = forceViewportEmulation(ctx); e != nil {
	// 	return
	// }
	// log.Debugf("page rect: %+v", *rect)

	// if e = chromedp.Run(ctx,
	// 	chromedp.WaitNotPresent(`div.attribution`),
	// ); e != nil {
	// 	e = errors.Wrap(e, "failed to wait 'div.attribution' to exit")
	// 	return
	// }

	// log.Debug("div.attribution exited")

	// dumpHTML(ctx, f.UID())
	// captureScreen(ctx, f.UID(), 90)

	html := ""
	selTable := `body div.wrap div.services_proxylist.services div div.table_block table tbody`
	if e = chromedp.Run(ctx,
		chromedp.WaitReady(selTable),
		chromedp.OuterHTML(`html`, &html),
	); e != nil {
		e = errors.Wrap(e, "failed to wait html table")
		return
	}

	log.Debug("html table is ready")
	log.Tracef("html source:\n %+v", html)

	var ips, ports, anon, ts, locs []string
	//extract first page
	if ips, ports, anon, ts, locs, e = f.extract(html); e != nil {
		e = errors.Wrap(e, "unable to extract proxy info")
		log.Error(e)
	} else {
		log.Tracef("hosts: %+q", ips)
		log.Tracef("ports: %+q", ports)
		log.Tracef("anon: %+q", anon)
		log.Tracef("types: %+q", ts)
		log.Tracef("locs: %+q", locs)
		newPS := f.parse(ips, ports, anon, ts, locs)
		log.Debugf("%d proxy info extracted at first page", len(newPS))
		ps = append(ps, newPS...)
	}

	//page page num
	var strPage string
	if e = chromedp.Run(ctx,
		chromedp.WaitReady(`body div div.services_proxylist.services div div.pagination`),
		chromedp.Text(`body div div.services_proxylist.services div div.pagination ul li:nth-last-child(2)`, &strPage),
	); e != nil {
		e = errors.Wrapf(e, "failed to get page num")
		log.Error(e)
		return ps, repeat.HintStop(e)
	}
	var numPage int64
	if numPage, e = strconv.ParseInt(strPage, 10, 64); e != nil {
		e = errors.Wrapf(e, "failed to convert page num string to integer: %s", strPage)
		log.Error(e)
		return ps, repeat.HintStop(e)
	}

	if numPage < 2 {
		return
	} else if numPage > 20 {
		numPage = 20
	}

	log.Debugf("#pages: %d", numPage)

	st := time.Millisecond * 2000
	selNextPage := `body div.wrap div.services_proxylist.services div div.pagination ul li.next_array`
	selNextTable := `body div div.services_proxylist.services div div.table_block table tbody`
	for i := 1; i < int(numPage); i++ {
		log.Debugf("flipping to page #%d", i+1)

		if e = chromedp.Run(ctx,
			chromedp.ScrollIntoView(`body div.wrap div.services_proxylist.services div div.pagination`),
			chromedp.Click(selNextPage),
			chromedp.WaitReady(selNextTable),
			chromedp.OuterHTML(`html`, &html),
		); e != nil {
			e = errors.Wrapf(e, "failed to flip to page #%d", i+1)
			return ps, repeat.HintStop(e)
		}

		log.Tracef("page#%d html source:\n %+v", i+1, html)
		log.Debugf("extracting page #%d", i+1)

		// if e = waitPageLoaded(ctx); e != nil {
		// 	e = errors.Wrapf(e, "failed to wait page load event at #%d", i+1)
		// 	return ps, repeat.HintStop(e)
		// }

		time.Sleep(st)

		if ips, ports, anon, ts, locs, e = f.extract(html); e != nil {
			e = errors.Wrapf(e, "unable to extract proxy info at page #%d", i+1)
			log.Error(e)
			continue
		}
		newPS := f.parse(ips, ports, anon, ts, locs)
		log.Debugf("%d proxy info extracted at page #%d", len(newPS), i+1)
		ps = append(ps, newPS...)
	}

	return
}
