package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/types"
	"github.com/chromedp/chromedp"
)

func TestFetch_SpysOne(t *testing.T) {
	log.Infof("config file used: %s", conf.ConfigFileUsed())
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Tracef("extracted proxy: %+v", px)
		}
	}()
	// gp := &SpysOne{[]string{`http://spys.one/free-proxy-list/CN/`}}
	gp := &SpysOne{}
	// gp := &SpysOne{}
	for i, url := range gp.Urls() {
		fetchFor(i, url, chpx, gp)
	}
}

func TestSelect_SpysOne(t *testing.T) {
	log.Infof("config file used: %s", conf.ConfigFileUsed())
	chpx := make(chan *types.ProxyServer, 100)
	go testServer(":8544")
	gp := &SpysOne{URLs: []string{
		"http://localhost:8544",
	}}
	for i, url := range gp.Urls() {
		fetchFor(i, url, chpx, gp)
	}

}

func TestDynamicSpysOne(t *testing.T) {
	// create context
	o := append(chromedp.DefaultExecAllocatorOptions[:],
		//... any options here
		chromedp.ProxyServer("socks5://localhost:1080"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.Args.Network.HTTPTimeout)*time.Second)
	defer cancel()

	ctx, cancel = chromedp.NewExecAllocator(ctx, o...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// run task list
	// var res string
	// var nodes []*cdp.Node
	var texts []string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`http://spys.one/en/anonymous-proxy-list/`),
		chromedp.WaitReady(`body table:nth-child(3) tbody tr:nth-child(5) td table tbody tr td:nth-child(1) font font.spy2`),
		// chromedp.Nodes(`body table:nth-child(3) tbody tr:nth-child(5) td table tbody tr td:nth-child(1) font.spy14`,
		// 	&nodes),
		chromedp.EvaluateAsDevTools(jsGetText(`body table:nth-child(3) tbody tr:nth-child(5) td table tbody tr td:nth-child(1) font.spy14`),
			&texts),
		// chromedp.Text(``, &res),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug(texts)

	// for i, n := range nodes {
	// 	log.Debugf("#%d value: %+v, text: %+v", i, n.Value, n.AttributeValue("Text"))
	// }

	// log.Println(strings.TrimSpace(res))
}

// testServer is a simple HTTP server that displays the passed headers in the html.
func testServer(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(res http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(res, indexHTML)
	})
	return http.ListenAndServe(addr, mux)
}

const indexHTML = `<!doctype html>
<html>
<head>
  <title>example</title>
</head>
<body>
  <div id="box1" style="display:none">
    <div id="box2">
      <p>box2</p>
    </div>
  </div>
  <div id="box3">
    <h2>box3</h3>
    <p id="box4">
      box4 text
      <input id="input1" value="some value"><br><br>
      <textarea id="textarea1" style="width:500px;height:400px">textarea</textarea><br><br>
      <input id="input2" type="submit" value="Next">
      <select id="xpp1" onchange="this.form.submit();">
        <option value="one">1</option>
        <option value="two">2</option>
        <option value="three">3</option>
		<option value="four">4</option>
		<option value="five">5</option>
		<option value="six">6</option>
	  </select>
	  <select id="long_list" onchange="this.form.submit();" size="2000">
        <option value="one">this is a very long list</option>
        <option value="two">2</option>
        <option value="three">3</option>
		<option value="four">4</option>
		<option value="five">5</option>
		<option value="six">6</option>
	  </select>
	  <br/>
	  <p>You've reached the end of page.</p>
    </p>
  </div>
</body>
</html>`
