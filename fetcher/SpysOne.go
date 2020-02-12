package fetcher

import (
	"strings"

	"github.com/carusyte/roprox/types"
	"github.com/chromedp/chromedp"
)

//SpysOne fetches proxy server from http://spys.one
type SpysOne struct{}

//UID returns the unique identifier for this spec.
func (f SpysOne) UID() string {
	//TODO this website requires dynamic html parsing
	return "SpysOne"
}

//Urls return the server urls that provide the free proxy server lists.
func (f SpysOne) Urls() []string {
	return []string{
		`http://spys.one/en/anonymous-proxy-list/`,
		`http://spys.one/en/socks-proxy-list/`,
	}
}

//UseMasterProxy returns whether the fetcher needs a master proxy server
//to access the free proxy list provider.
func (f SpysOne) UseMasterProxy() bool {
	return true
}

//RefreshInterval determines how often the list should be refreshed, in minutes.
func (f SpysOne) RefreshInterval() int {
	return 10
}

//Actions return the webdriver actions after the specified url is visited
func (f SpysOne) Actions(urlIdx int, url string) (tasks chromedp.Tasks, values interface{}) {
	data := &spysOneData{
		ipPort:    make([]string, 0, 4),
		types:     make([]string, 0, 4),
		locations: make([]string, 0, 4),
	}
	var nores interface{}
	tasks = chromedp.Tasks{
		chromedp.WaitReady(`#xpp`),
		chromedp.Evaluate(jsSelect(`#xpp`, "500"), &nores),
		chromedp.WaitReady(`body table:nth-child(3) tbody tr:nth-child(5) td ` +
			`table tbody tr:nth-child(200)`),
		//get ip and port
		chromedp.Evaluate(jsGetText(`body table:nth-child(3) tbody tr:nth-child(5) `+
			`td table tbody tr td:nth-child(1) font.spy14`), &(data.ipPort)),
		//get types
		chromedp.Evaluate(jsGetText(`body table:nth-child(3) tbody tr:nth-child(5) `+
			`td table tbody tr td:nth-child(2) a`), &(data.types)),
		//get anonymity
		chromedp.Evaluate(jsGetText(`body table:nth-child(3) tbody tr:nth-child(5) `+
			`td table tbody tr td:nth-child(3) a font`), &(data.anon)),
		//get location
		chromedp.Evaluate(jsGetText(`body table:nth-child(3) tbody tr:nth-child(5) `+
			`td table tbody tr td:nth-child(4) a font.spy14`), &(data.locations)),
	}
	return tasks, data
}

//OnComplete parses the selected values into proxy server instances
func (f SpysOne) OnComplete(values interface{}) (ps []*types.ProxyServer) {
	data := values.(*spysOneData)
	for i, d := range data.ipPort {
		if len(data.anon) <= i {
			break
		}
		if len(data.locations) <= i {
			break
		}
		if len(data.types) <= i {
			break
		}

		a := strings.TrimSpace(data.anon[i])
		if strings.EqualFold(a, "NOA") {
			//non anonymous proxy
			continue
		}

		ss := strings.Split(strings.TrimSpace(d), ":")
		if len(ss) != 2 {
			log.Warnf("%s possible invalid ip & port string: %+v", f.UID(), d)
			continue
		}
		host, port := ss[0], ss[1]

		t := strings.ToLower(strings.TrimSpace(data.types[i]))
		if strings.Contains(t, "http") {
			t = "http"
		} else if strings.Contains(t, "socks5") {
			t = "socks5"
		} else {
			log.Debugf("%s unsupported proxy type: %+v", f.UID(), t)
			continue
		}

		loc := strings.TrimSpace(data.locations[i])

		ps = append(ps, &types.ProxyServer{
			Source: f.UID(),
			Host:   host,
			Port:   port,
			Type:   t,
			Loc:    loc,
		})
	}
	return
}

type spysOneData struct {
	ipPort    []string
	types     []string
	locations []string
	anon      []string
}
