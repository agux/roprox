package util

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/carusyte/roprox/conf"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
)

//HTTPGetResponse initiates HTTP get request and returns its response
func HTTPGetResponse(link string, headers map[string]string, useMasterProxy, rotateAgent bool) (res *http.Response, e error) {
	host := DomainOf(link)
	var client *http.Client
	//determine if we must use a master proxy
	if useMasterProxy {
		// create a socks5 dialer
		dialer, err := proxy.SOCKS5("tcp", conf.Args.MasterProxyAddr, nil, proxy.Direct)
		if err != nil {
			logrus.Errorln("can't connect to the master proxy", err)
			return nil, errors.WithStack(err)
		}
		// setup a http client
		httpTransport := &http.Transport{Dial: dialer.Dial}
		client = &http.Client{Timeout: time.Second * 60, // Maximum of 60 secs
			Transport: httpTransport}
	} else {
		client = &http.Client{Timeout: time.Second * 60} // Maximum of 60 secs
	}

	for i := 0; true; i++ {
		req, err := http.NewRequest(http.MethodGet, link, nil)
		if err != nil {
			logrus.Panic(err)
		}

		req.Header.Set("Accept", "text/html,application/xhtml+xml,"+
			"application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7,zh-TW;q=0.6")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Connection", "close")
		if host != "" {
			req.Header.Set("Host", host)
		}
		req.Header.Set("Pragma", "no-cache")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		uagent := ""
		if rotateAgent {
			uagent, e = PickUserAgent()
			if e != nil {
				logrus.Errorln("failed to acquire rotate user agent", e)
				time.Sleep(time.Millisecond * time.Duration(300+rand.Intn(300)))
				continue
			}
		} else {
			uagent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_1) " +
				"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.94 Safari/537.36"
		}
		req.Header.Set("User-Agent", uagent)
		if headers != nil && len(headers) > 0 {
			for k, hv := range headers {
				req.Header.Set(k, hv)
			}
		}

		res, err = client.Do(req)
		if err != nil {
			//handle "read: connection reset by peer" error by retrying
			if i >= conf.Args.HTTPRetry {
				logrus.Errorf("http communication failed. url=%s\n%+v", link, err)
				e = err
				return
			}
			logrus.Errorf("http communication error. url=%s, retrying %d ...\n%+v", link, i+1, err)
			if res != nil {
				res.Body.Close()
			}
			time.Sleep(time.Millisecond * time.Duration(500+rand.Intn(300)))
		} else {
			return
		}
	}
	return
}
