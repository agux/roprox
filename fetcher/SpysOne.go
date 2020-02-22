package fetcher

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/carusyte/roprox/conf"
	"github.com/carusyte/roprox/types"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
)

//SpysOne fetches proxy server from http://spys.one
type SpysOne struct {
	URLs []string
}

func (f SpysOne) HomePageTimeout() int {
	return 20
}

//UID returns the unique identifier for this spec.
func (f SpysOne) UID() string {
	return "SpysOne"
}

func (f SpysOne) Retry() int {
	return conf.Args.DataSource.SpysOne.Retry
}

//Urls return the server urls that provide the free proxy server lists.
func (f SpysOne) Urls() []string {
	if len(f.URLs) > 0 {
		return f.URLs
	}
	return []string{
		`http://spys.one/free-proxy-list/CN/`,
		`http://spys.one/asia-proxy/`,
		`http://spys.one/en/anonymous-proxy-list/`,
		`http://spys.one/en/socks-proxy-list/`,
	}
}

//ProxyMode returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f SpysOne) ProxyMode() types.ProxyMode {
	return types.ProxyMode(conf.Args.DataSource.SpysOne.ProxyMode)
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f SpysOne) RefreshInterval() int {
	return conf.Args.DataSource.SpysOne.RefreshInterval
}

//Headless for web driver
func (f SpysOne) Headless() bool {
	return conf.Args.DataSource.SpysOne.Headless
}

//check if it shows the ban page
func (f SpysOne) isBanned(parent context.Context) (b bool, e error) {
	ctx, c := context.WithTimeout(parent, time.Second*5)
	defer c()
	var msg string
	msgSel := `body > table > tbody > tr > td > table > tbody > tr:nth-child(2) > td`
	if e = chromedp.Run(ctx,
		chromedp.WaitReady(msgSel),
		chromedp.TextContent(msgSel, &msg),
	); e != nil {
		return b, errors.Wrapf(e, "failed to query node with selector '%s' for ban page", msgSel)
	}
	b = strings.EqualFold(
		strings.TrimSpace(msg),
		`This IP address has been blocked for excessive page loads.`)
	return
}

func (f SpysOne) fetchProxyForBot(parent context.Context) (ps []*types.ProxyServer, e error) {
	//TODO implements me
	log.Debugf("%s fetching proxy info from page for bot", f.UID())
	// if e = chromedp.Run(parent,

	// )
	return
}

//Fetch the proxy info.
func (f SpysOne) Fetch(parent context.Context, urlIdx int, url string) (ps []*types.ProxyServer, e error) {

	//check if #xpp is present (valid). otherwise the source IP has been banned
	ctx, c := context.WithTimeout(parent, time.Second*10)
	defer c()
	if e = chromedp.Run(ctx,
		chromedp.WaitVisible(`#xpp`),
	); e != nil {
		e = errors.Wrapf(e, "%s #xpp cannot be detected", f.UID())
		log.Error(e)
		//check if banned page is shown
		if b, e := f.isBanned(parent); e != nil {
			e = errors.Wrapf(e, "failed to check ban page")
			return ps, e
		} else if b {
			return f.fetchProxyForBot(parent)
		}
		//unknown state
		return ps, e
	}

	ipPort := make([]string, 0, 4)
	ts := make([]string, 0, 4)
	anon := make([]string, 0, 4)
	locations := make([]string, 0, 4)
	var xppLen int
	var str string

	if e = chromedp.Run(ctx,
		// chromedp.WaitVisible(`#xpp`),
		chromedp.JavascriptAttribute(`#xpp`, `length`, &xppLen),
		chromedp.TextContent(`#xpp option:last-child`, &str),
		chromedp.SetAttributeValue(`#xpp`, "multiple", ""),
		chromedp.SetAttributeValue(`#xpp`, "size", strconv.Itoa(xppLen)),
		chromedp.Sleep(time.Millisecond*1500),
		chromedp.Click(`#xpp option:last-child`),
	); e != nil {
		e = errors.Wrapf(e, "failed to manipulate #xpp")
		return ps, e
	}
	log.Debugf("#xpp len: %d, max record string: %s", xppLen, str)
	// if max, e = strconv.Atoi(str); e != nil {
	// 	return ps, errors.Wrapf(e, "unable to convert max record string: %s", str)
	// }

	if e = chromedp.Run(ctx,
		chromedp.WaitReady(`body > table:nth-child(3) > tbody > tr:nth-child(5) > td > table`),
		chromedp.WaitReady(`body > table:nth-child(3) > tbody > tr:nth-child(5) > td `+
			`> table > tbody > tr:nth-child(30)`),
	); e != nil {
		return ps, errors.Wrapf(e, "failed to wait page refresh")
	}

	if e = scrollToBottom(ctx); e != nil {
		return
	}

	typeSel := `body > table:nth-child(3) > tbody > tr:nth-child(5) ` +
		`> td > table > tbody > tr > td:nth-child(2) > a`
	if strings.Contains(url, "socks-proxy-list") {
		typeSel = `body > table:nth-child(3) > tbody > tr:nth-child(5) ` +
			`> td > table > tbody > tr:not(:nth-child(2)) > td:nth-child(2)`
	}
	anonSel := `body > table:nth-child(3) > tbody > tr:nth-child(5) > td > table > tbody > tr > td:nth-child(3) > a > font`
	locSel := `body > table:nth-child(3) > tbody > tr:nth-child(5) > td > table > tbody > tr > td:nth-child(4)`
	if strings.Contains(url, "http://spys.one/free-proxy-list/CN") ||
		strings.Contains(url, "http://spys.one/asia-proxy/") {
		anonSel = `body > table:nth-child(3) > tbody > tr:nth-child(5) > td > table > tbody > tr:nth-child(n+4) > td:nth-child(3)`
		locSel = `body > table:nth-child(3) > tbody > tr:nth-child(5) > td > table > tbody > tr:nth-child(n+4) > td:nth-child(4)`
	}

	if e = chromedp.Run(ctx,
		// chromedp.WaitReady(fmt.Sprintf(`body > table:nth-child(3) > tbody > tr:nth-child(5) > td `+
		// 	`> table > tbody > tr:nth-child(%d)`, max)),
		//get ip and port
		chromedp.Evaluate(jsGetText(`body > table:nth-child(3) > tbody > tr:nth-child(5) `+
			`> td > table > tbody > tr > td:nth-child(1) > font.spy14`), &ipPort),
		//get types
		chromedp.Evaluate(jsGetText(typeSel), &ts),
		//get anonymity
		chromedp.Evaluate(jsGetText(anonSel), &anon),
		//get location
		chromedp.Evaluate(jsGetText(locSel), &locations),
	); e != nil {
		return ps, errors.Wrapf(e, "failed to extract proxy info")
	}

	return f.parse(ipPort, ts, anon, locations), nil
}

//parses the selected values into proxy server instances
func (f SpysOne) parse(ipPort, ts, anon, locations []string) (ps []*types.ProxyServer) {
	for i, d := range ipPort {
		if len(anon) <= i {
			break
		}
		if len(locations) <= i {
			break
		}
		if len(ts) <= i {
			break
		}

		a := strings.TrimSpace(anon[i])
		if strings.EqualFold(a, "NOA") {
			//non anonymous proxy
			continue
		}

		ss := strings.Split(strings.TrimSpace(d), ":")
		if len(ss) != 2 {
			log.Warnf("%s possible invalid ip & port string, skipping: %+v", f.UID(), d)
			continue
		}
		host, port := strings.TrimSpace(ss[0]), strings.TrimSpace(ss[1])

		t := strings.ToLower(strings.TrimSpace(ts[i]))
		if strings.Contains(t, "http") {
			t = "http"
		} else if strings.Contains(t, "socks5") {
			t = "socks5"
		} else {
			log.Debugf("%s unsupported proxy type: %+v", f.UID(), t)
			continue
		}

		loc := strings.TrimSpace(strings.ReplaceAll(locations[i], "!", ""))

		ps = append(ps, types.NewProxyServer(f.UID(), host, port, t, loc))
	}
	return
}
