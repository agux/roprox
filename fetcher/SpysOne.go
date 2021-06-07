package fetcher

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/types"
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
		`http://spys.one/en/http-proxy-list/`,
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

//isBanned check if it shows the ban page
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

//Fetch the proxy info.
func (f SpysOne) Fetch(parent context.Context, urlIdx int, url string) (ps []*types.ProxyServer, e error) {

	//check if #xpp is present (valid). otherwise the source IP might have been banned
	ctx, c := context.WithTimeout(parent, time.Second*10)
	defer c()
	if e = chromedp.Run(ctx,
		chromedp.WaitVisible(`#xpp`),
	); e != nil {
		e = errors.Wrapf(e, "%s #xpp cannot be detected", f.UID())
		log.Error(e)
		//check if banned page is shown
		// if b, e := f.isBanned(parent); e != nil {
		// 	e = errors.Wrapf(e, "failed to check ban page")
		// 	return ps, e
		// } else if b {
		// 	// return f.fetchProxyForBot(parent)
		// }
		//unknown state
		return ps, e
	}

	ipPort := make([]string, 0, 4)
	ts := make([]string, 0, 4)
	anon := make([]string, 0, 4)
	locations := make([]string, 0, 4)
	var xppLen int
	var str string

	if e = chromedp.Run(parent,
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

	if e = chromedp.Run(parent,
		chromedp.WaitReady(`body > table:nth-child(3) > tbody > tr:nth-child(4) > td > table`),
		chromedp.WaitReady(`body > table:nth-child(3) > tbody > tr:nth-child(4) > td > table > tbody > tr:nth-child(30)`),
	); e != nil {
		return ps, errors.Wrapf(e, "failed to wait page refresh")
	}

	if e = scrollToBottom(parent); e != nil {
		return
	}

	var startRow int
	if startRow, e = f.findStartingRow(parent); e != nil {
		return
	}

	baseSel := `body > table:nth-child(3) > tbody > tr:nth-child(4) > td > table > tbody > tr:nth-child(n+%d) > td:nth-child(%d)`
	ipPortSel := fmt.Sprintf(baseSel, startRow, 1)
	typeSel := fmt.Sprintf(baseSel, startRow, 2)
	anonSel := fmt.Sprintf(baseSel, startRow, 3)
	locSel := fmt.Sprintf(baseSel, startRow, 4)

	if e = chromedp.Run(parent,
		// chromedp.WaitReady(fmt.Sprintf(`body > table:nth-child(3) > tbody > tr:nth-child(5) > td `+
		// 	`> table > tbody > tr:nth-child(%d)`, max)),
		//get ip and port
		chromedp.Evaluate(jsGetText(ipPortSel), &ipPort),
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

func (f SpysOne) findStartingRow(ctx context.Context) (startRow int, e error) {
	baseSel := `body > table:nth-child(3) > tbody > tr:nth-child(4) > td > table > tbody > tr:nth-child(%d)`
	ipPortPattern := `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{1,5}`
	re := regexp.MustCompile(ipPortPattern)
	var rowStr string
	for i := 1; i <= 10; i++ {
		sel := fmt.Sprintf(baseSel, i)
		if e = chromedp.Run(ctx,
			chromedp.Text(sel, &rowStr),
		); e != nil {
			e = errors.Wrapf(e, `failed to get row string with selector "%s"`, sel)
			return
		}
		log.Debugf("row#%d string: %s", i, rowStr)
		if re.MatchString(rowStr) {
			startRow = i
			return
		}
	}
	e = errors.Wrapf(e, `failed to match ip:port anchor string with base selector "%s"`, baseSel)
	return
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

		host, port, e := net.SplitHostPort(strings.TrimSpace(d))
		if e != nil {
			log.Warnf("%s possible invalid ip & port string %s, skipping: %+v", f.UID(), d, e)
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

		loc := strings.TrimSpace(strings.ReplaceAll(locations[i], "!", ""))

		ps = append(ps, types.NewProxyServer(f.UID(), host, port, t, loc))
	}
	return
}
