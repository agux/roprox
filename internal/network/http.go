package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/types"
	"github.com/agux/roprox/internal/ua"
	"github.com/agux/roprox/internal/util"
	"github.com/pkg/errors"
	"github.com/ssgreg/repeat"
	"golang.org/x/net/proxy"
)

// TODO: need to continuously identify unrecoverable errors and stop retrying immediately.
// Don't waste time retrying.
// Possible situations:
// Bad Request
// Method Not Allowed
// no Host in request URL
// check whether the error message within e contains any of the key words above, ignoring case.
var unrecoverableErrors = []string{
	"Bad Request",
	"Method Not Allowed",
	"no Host in request URL",
}

// HTTPGet initiates HTTP get request and returns its response
func HTTPGet(link string, headers map[string]string, px *types.ProxyServer, timeout, maxTimeout time.Duration,
	cookies ...*http.Cookie) (res *http.Response, e error) {

	//extract host from the url link
	host := ""
	u, err := url.Parse(link)
	if err != nil {
		e = errors.Wrap(err, "failed to parse URL")
		return
	}
	host = u.Host

	req, e := http.NewRequest(http.MethodGet, link, strings.NewReader(""))
	if e != nil {
		log.Panicf("unable to create http request: %+v", e)
	}

	req.RequestURI = "" // Request.RequestURI can't be set in client requests
	if req.URL.Scheme == "" {
		req.URL.Scheme = "https"
	}
	req.URL.Host = host
	if req.Host == "" {
		req.Host = host
	}

	req.Header.Set("Accept", "*/*")
	if len(headers) > 0 {
		for k, hv := range headers {
			req.Header.Set(k, hv)
		}
	}
	if len(req.Header.Get("User-Agent")) == 0 {
		uaStr := conf.Args.Network.DefaultUserAgent
		if px != nil {
			uaStr, e = ua.GetUserAgent(px.UrlString())
		}
		if e != nil {
			log.Error("failed to get user agent from the pool. falling back to default user-agent", e)
		}
		req.Header.Set("User-Agent", uaStr)
	}

	var client *http.Client
	var transport *http.Transport
	if transport, e = GetTransport(px, true); e != nil {
		return
	}
	client = &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	for _, c := range cookies {
		req.AddCookie(c)
	}

	op := func(c int) error {
		if c == 0 && px != nil {
			log.Tracef("sending HTTP request via proxy [%s]:\n%+v", px.UrlString(), req)
		}
		res, e = client.Do(req)
		if e != nil {
			proxyStr := ""
			if px != nil {
				proxyStr = fmt.Sprintf(" [proxy=%s]", px.UrlString())
			}
			if res != nil {
				res.Body.Close()
			}
			for _, unrecoverableError := range unrecoverableErrors {
				if strings.Contains(strings.ToLower(e.Error()), strings.ToLower(unrecoverableError)) {
					return repeat.HintStop(e)
				}
			}
			log.Debugf("http communication error: [%+v]%s url=%s, retrying %d ...", e, proxyStr, link, c+1)
			return repeat.HintTemporary(e)
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), maxTimeout)
	rp := repeat.WithContext(ctx)
	defer cancel()
	e = rp.Repeat(
		repeat.FnWithCounter(op),
		repeat.StopOnSuccess(),
		repeat.WithDelay(
			repeat.FullJitterBackoff(200*time.Millisecond).WithMaxDelay(2*time.Second).Set(),
		),
	)

	return
}

// HTTPGetResponse initiates HTTP get request and returns its response
func HTTPGetResponse(link string, headers map[string]string, useMasterProxy, rotateAgent bool) (res *http.Response, e error) {
	host := DomainOf(link)
	var client *http.Client
	timeout := time.Second * time.Duration(conf.Args.Network.HTTPTimeout)
	//determine if we must use a master proxy
	if useMasterProxy {
		ps := util.GetMasterProxy()
		var transport *http.Transport
		if transport, e = GetTransport(ps, true); e != nil {
			return
		}
		client = &http.Client{
			Timeout:   timeout,
			Transport: transport,
		}
	} else {
		client = &http.Client{Timeout: timeout}
	}

	for i := 0; true; i++ {
		req, err := http.NewRequest(http.MethodGet, link, nil)
		if err != nil {
			log.Panic(err)
		}

		req.Header.Set("Accept", "text/html,application/xhtml+xml,"+
			"application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Connection", "close")
		if host != "" {
			req.Header.Set("Host", host)
		}
		req.Header.Set("Pragma", "no-cache")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		uagent := ""
		if rotateAgent {
			uagent, e = ua.PickUserAgent()
			if e != nil {
				log.Errorln("failed to acquire rotate user agent", e)
				time.Sleep(time.Millisecond * time.Duration(300+rand.Intn(300)))
				continue
			}
		} else {
			uagent = conf.Args.Network.DefaultUserAgent
		}
		req.Header.Set("User-Agent", uagent)
		if len(headers) > 0 {
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

func GetTransport(ps *types.ProxyServer, insecureSkipVerify bool) (transport *http.Transport, e error) {
	transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
	}
	if ps == nil {
		transport.Proxy = nil
		return
	}
	if strings.HasPrefix(ps.Type, "http") {
		var proxyURL *url.URL
		if proxyURL, e = url.Parse(ps.UrlString()); e != nil {
			e = errors.Wrapf(e, "Error parsing proxy URL: %s", ps.UrlString())
			return
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	} else {
		var dialer proxy.Dialer
		if dialer, e = proxy.SOCKS5("tcp", fmt.Sprintf("%s:%s", ps.Host, ps.Port), nil, proxy.Direct); e != nil {
			e = errors.Wrapf(e, "Error creating SOCKS5 dialer")
			return
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
	}
	return
}
