package fetcher

import (
	"context"
	"testing"
	"time"

	"github.com/carusyte/roprox/conf"
	"github.com/carusyte/roprox/types"
	"github.com/chromedp/chromedp"
)

func TestFetch_SpysOne(t *testing.T) {
	//FIXME unable to fetch 500 records by selecting the record number
	log.Infof("config file used: %s", conf.ConfigFileUsed())
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
		}
	}()
	gp := &SpysOne{}
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.Args.HTTPTimeOut) * time.Second)
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
