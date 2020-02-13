package fetcher

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/carusyte/roprox/types"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"github.com/ssgreg/repeat"
)

//GatherProxy fetches proxy server from http://www.gatherproxy.com
type GatherProxy struct{}

//UID returns the unique identifier for this spec.
func (f GatherProxy) UID() string {
	return "GatherProxy"
}

//Urls return the server urls that provide the free proxy server lists.
func (f GatherProxy) Urls() []string {
	return []string{
		`http://www.gatherproxy.com/proxylist/anonymity/?t=Elite`,
		`http://www.gatherproxy.com/proxylist/anonymity/?t=Anonymous`,
		`http://www.gatherproxy.com/proxylist/country/?c=China`,
	}
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f GatherProxy) UseMasterProxy() bool {
	return true
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f GatherProxy) RefreshInterval() int {
	return 20
}

func (f GatherProxy) extract(ctx context.Context) (i, p, a, l []string, e error) {
	i = make([]string, 0, 4)
	p = make([]string, 0, 4)
	a = make([]string, 0, 4)
	l = make([]string, 0, 4)
	if e = chromedp.Run(ctx,
		chromedp.WaitReady("#tblproxy"),
		chromedp.Evaluate(jsGetText(`#tblproxy > tbody > tr > td:nth-child(2)`), &i),
		chromedp.Evaluate(jsGetText(`#tblproxy > tbody > tr > td:nth-child(3)`), &p),
		chromedp.Evaluate(jsGetText(`#tblproxy > tbody > tr > td:nth-child(4)`), &a),
		chromedp.Evaluate(jsGetText(`#tblproxy > tbody > tr > td:nth-child(5)`), &l),
	); e != nil {
		e = errors.Wrap(e, "failed to extract proxy info")
	}
	return
}

func (f GatherProxy) parse(ips, ports, anon, locs []string) (ps []*types.ProxyServer) {
	for i, ip := range ips {
		if len(ports) <= i {
			break
		}
		if len(anon) <= i {
			break
		}
		if len(locs) <= i {
			break
		}
		if strings.EqualFold(strings.TrimSpace(anon[i]), "Transparent") {
			continue
		}
		ip = strings.TrimSpace(ip)
		port := strings.TrimSpace((ports[i]))
		loc := strings.TrimSpace((locs[i]))
		ps = append(ps, types.NewProxyServer(f.UID(), ip, port, "http", loc))
	}
	return
}

//Fetch the proxy info
func (f GatherProxy) Fetch(ctx context.Context, urlIdx int, url string) (ps []*types.ProxyServer, e error) {
	var ips, ports, anon, locs []string
	//extract first page
	if ips, ports, anon, locs, e = f.extract(ctx); e != nil {
		e = errors.Wrapf(e, "target url: %s", url)
		log.Error(e)
	} else {
		newPS := f.parse(ips, ports, anon, locs)
		ps = append(ps, newPS...)
	}

	if e = chromedp.Run(ctx,
		//click "Show Full List"
		chromedp.WaitReady("#body > form > p > input"),
		chromedp.Click("#body > form > p > input"),
	); e != nil {
		e = errors.Wrapf(e, "failed to visit full list: %s", url)
		log.Error(e)
		return ps, repeat.HintStop(e)
	}

	var numPage int
	if e = chromedp.Run(ctx,
		chromedp.WaitReady(`#psbform > div`),
		chromedp.JavascriptAttribute(`#psbform > div`, "childElementCount", &numPage),
	); e != nil {
		e = errors.Wrapf(e, "failed to page num of pages: %s", url)
		log.Error(e)
		return ps, repeat.HintStop(e)
	}

	if numPage == 1 {
		return
	} else if numPage > 5 {
		numPage = 5
	}

	extraPages := make([]string, numPage-1)
	for i := 0; i < numPage-1; i++ {
		extraPages[i] = strconv.Itoa(i + 2)
	}

	for _, ep := range extraPages {
		if e = chromedp.Run(ctx,
			chromedp.Click(fmt.Sprintf("#psbform > div > a:nth-child(%s)", ep)),
			chromedp.WaitReady(fmt.Sprintf(`//*[@id="psbform"]/div/span[text()='%s']`, ep)),
		); e != nil {
			e = errors.Wrapf(e, "failed to flip to page #%s : %s", ep, url)
			log.Error(e)
			return ps, repeat.HintStop(e)
		}

		if ips, ports, anon, locs, e = f.extract(ctx); e != nil {
			e = errors.Wrapf(e, "failed to extract page #%s : %s", ep, url)
			log.Error(e)
			continue
		}
		newPS := f.parse(ips, ports, anon, locs)
		ps = append(ps, newPS...)
	}

	return
}
