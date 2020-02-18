package util

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/carusyte/roprox/conf"
	"github.com/carusyte/roprox/types"
	"github.com/pkg/errors"
	"github.com/ssgreg/repeat"
	"golang.org/x/net/proxy"
)

const RETRY int = 3

//HTTPGet initiates HTTP get request and returns its response
func HTTPGet(link string, headers map[string]string,
	px *types.ProxyServer, cookies ...*http.Cookie) (res *http.Response, e error) {
	host := ""
	r := regexp.MustCompile(`//([^/]*)/`).FindStringSubmatch(link)
	if len(r) > 0 {
		host = r[len(r)-1]
	}

	var client *http.Client
	req, e := http.NewRequest(http.MethodGet, link, nil)
	if e != nil {
		log.Panicf("unable to create http request: %+v", e)
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
	if headers != nil && len(headers) > 0 {
		for k, hv := range headers {
			req.Header.Set(k, hv)
		}
	}
	if len(req.Header.Get("User-Agent")) == 0 {
		req.Header.Set("User-Agent", conf.Args.Network.DefaultUserAgent)
	}

	var proxyAddr string
	if px == nil {
		//no proxy used
		client = &http.Client{Timeout: time.Second * time.Duration(conf.Args.Network.HTTPTimeout)}
	} else {
		proxyAddr = fmt.Sprintf("%s://%s:%s", px.Type, px.Host, px.Port)
		switch px.Type {
		case "socks5":
			// create a socks5 dialer
			dialer, e := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%s", px.Host, px.Port), nil, proxy.Direct)
			if e != nil {
				log.Warnf("can't create socks5 proxy dialer: %+v", e)
				return nil, errors.WithStack(e)
			}
			httpTransport := &http.Transport{Dial: dialer.Dial}
			client = &http.Client{Timeout: time.Second * time.Duration(conf.Args.Network.HTTPTimeout),
				Transport: httpTransport}
		case "http":
			//http proxy
			proxyAddr := fmt.Sprintf("%s://%s:%s", px.Type, px.Host, px.Port)
			proxyURL, e := url.Parse(proxyAddr)
			if e != nil {
				log.Warnf("invalid proxy: %s, %+v", proxyAddr, e)
				return nil, errors.WithStack(e)
			}
			client = &http.Client{
				Timeout:   time.Second * time.Duration(conf.Args.Network.HTTPTimeout),
				Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
		default:
			return nil, errors.Errorf("unsupported proxy: %+v", px)
		}
	}

	for _, c := range cookies {
		req.AddCookie(c)
	}

	op := func(c int) error {
		res, e = client.Do(req)
		if e != nil {
			//handle "read: connection reset by peer" error by retrying
			proxyStr := ""
			if proxyAddr != "" {
				proxyStr = fmt.Sprintf(" [proxy=%s]", proxyAddr)
				UpdateProxyScore(px, false)
			}
			log.Debugf("http communication error: [%+v]%s url=%s, retrying %d ...", e, proxyStr, link, c+1)
			if res != nil {
				res.Body.Close()
			}
			return repeat.HintTemporary(e)
		}
		return nil
	}

	e = repeat.Repeat(
		repeat.FnWithCounter(op),
		repeat.StopOnSuccess(),
		repeat.LimitMaxTries(RETRY),
		repeat.WithDelay(
			repeat.FullJitterBackoff(200*time.Millisecond).WithMaxDelay(2*time.Second).Set(),
		),
	)

	return
}

//HTTPGetResponse initiates HTTP get request and returns its response
func HTTPGetResponse(link string, headers map[string]string, useMasterProxy, rotateAgent bool) (res *http.Response, e error) {
	host := DomainOf(link)
	var client *http.Client
	//determine if we must use a master proxy
	if useMasterProxy {
		// create a socks5 dialer
		dialer, err := proxy.SOCKS5("tcp", conf.Args.Network.MasterProxyAddr, nil, proxy.Direct)
		if err != nil {
			log.Errorln("can't connect to the master proxy", err)
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
			log.Panic(err)
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
				log.Errorln("failed to acquire rotate user agent", e)
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
			if i >= conf.Args.Network.HTTPRetry {
				log.Errorf("http communication failed. url=%s\n%+v", link, err)
				e = err
				return
			}
			log.Errorf("http communication error. url=%s, retrying %d ...\n%+v", link, i+1, err)
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
