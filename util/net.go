package util

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/conf"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
)

//DomainOf the specified url.
func DomainOf(url string) (domain string) {
	r := regexp.MustCompile(`//([^/]*)/`).FindStringSubmatch(url)
	if len(r) > 0 {
		domain = r[len(r)-1]
	}
	return
}

//ValidateProxy checks the status of remote listening port, and further checks if it's a valid proxy server
func ValidateProxy(stype, host, port string) bool {
	timeout := time.Second * time.Duration(conf.Args.CheckTimeout)
	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		logrus.Warnf("%s failed: %+v", addr, err)
		return false
	}
	if conn == nil {
		logrus.Warnf("%s timed out", addr)
		return false
	}
	conn.Close()

	link := `http://www.baidu.com`

	var client *http.Client
	if strings.EqualFold("socks5", stype) {
		dialer, err := proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
		if err != nil {
			logrus.Warnln(addr, " failed, ", err)
			return false
		}
		httpTransport := &http.Transport{Dial: dialer.Dial}
		client = &http.Client{Timeout: timeout, Transport: httpTransport}
	} else if strings.EqualFold("http", stype) {
		addr = fmt.Sprintf("%s://%s:%s", stype, host, port)
		proxyURL, e := url.Parse(addr)
		if e != nil {
			logrus.Errorf("invalid proxy address: %s, %+v", addr, e)
			return false
		}
		client = &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	} else {
		logrus.Warn("proxy protocol checking not supported", stype)
	}

	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		logrus.Warnf("%s failed to create get request: %+v", addr, err)
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,"+
		"application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7,zh-TW;q=0.6")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "close")
	req.Header.Set("Host", "www.baidu.com")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	uagent, e := PickUserAgent()
	if e != nil {
		logrus.Errorln("failed to acquire rotate user agent, using default value", e)
		uagent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_1) " +
			"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.94 Safari/537.36"
	}
	req.Header.Set("User-Agent", uagent)

	res, err := client.Do(req)
	if err != nil {
		logrus.Warnf("%s failed to visit validation site: %+v", addr, err)
		return false
	}

	defer res.Body.Close()
	doc, e := goquery.NewDocumentFromReader(res.Body)
	if e != nil {
		logrus.Warnf("%s failed to read validation site's response body: %+v", addr, e)
		return false
	}
	size := doc.Find("#wrapper").Size()
	if size > 0 {
		logrus.Debugf("%s success", addr)
		return true
	}
	logrus.Warnf("%s failed to identify target on validation site", addr)
	return false
}
