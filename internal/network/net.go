package network

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/data"
	"github.com/agux/roprox/internal/types"
	"github.com/agux/roprox/internal/ua"
	"github.com/pkg/errors"
	"golang.org/x/net/proxy"
)

// DomainOf the specified url.
func DomainOf(url string) (domain string) {
	r := regexp.MustCompile(`//([^/]*)/`).FindStringSubmatch(url)
	if len(r) > 0 {
		domain = r[len(r)-1]
	}
	return
}

// ValidateProxy checks the status of remote listening port, and further checks if it's a valid proxy server
func ValidateProxy(stype, host, port, link, keyword string, probeTimeout int) bool {
	timeout := time.Second * time.Duration(probeTimeout)
	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	if err != nil {
		log.Tracef("proxy validation failed [%s]: %+v", addr, err)
		return false
	}
	if conn == nil {
		log.Tracef("proxy validation timed out [%s]", addr)
		return false
	}
	conn.Close()

	var client *http.Client
	if strings.EqualFold("socks5", stype) {
		dialer, err := proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
		if err != nil {
			log.Warnln(addr, " failed, ", err)
			return false
		}
		httpTransport := &http.Transport{
			Dial:            dialer.Dial,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Timeout: timeout, Transport: httpTransport}
	} else if strings.EqualFold("http", stype) {
		addr = fmt.Sprintf("%s://%s:%s", stype, host, port)
		proxyURL, e := url.Parse(addr)
		if e != nil {
			log.Errorf("invalid proxy address: %s, %+v", addr, e)
			return false
		}
		client = &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Proxy:           http.ProxyURL(proxyURL),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}}
	} else {
		log.Warn("proxy protocol checking not supported", stype)
	}

	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		log.Warnf("%s failed to create get request: %+v", addr, err)
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,"+
		"application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7,zh-TW;q=0.6")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "close")
	req.Header.Set("Host", link)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	uagent, e := ua.PickUserAgent()
	if e != nil {
		log.Errorln("failed to acquire rotate user agent, using default value", e)
		uagent = conf.Args.Network.DefaultUserAgent
	}
	req.Header.Set("User-Agent", uagent)

	res, err := client.Do(req)
	if err != nil {
		log.Tracef("%s failed to visit validation site: %+v", addr, err)
		return false
	}
	defer res.Body.Close()

	// read all data from res.Body and convert to string
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Tracef("%s failed to read validation site's response body: %+v", addr, err)
		return false
	}
	bodyString := string(bodyBytes)

	// if bodyString contains keyword, the validation is deemed successful. otherwise it's failure and return false.
	if strings.Contains(bodyString, keyword) {
		log.Tracef("%s success", addr)
		return true
	}

	log.Tracef("%s failed to identify target on validation site", addr)
	return false
}

// PickProxy randomly chooses a proxy from database.
func PickProxy() (proxy *types.ProxyServer, e error) {
	proxyList := make([]*types.ProxyServer, 0, 64)
	query := `
		SELECT 
			*
		FROM
			proxy_servers
		WHERE
			score >= ?
	`
	e = data.GormDB.Raw(query, conf.Args.Network.RotateProxyScoreThreshold).Scan(&proxyList).Error
	if e != nil {
		log.Println("failed to query proxy server from database", e)
		return proxy, errors.WithStack(e)
	}
	log.Infof("successfully fetched %d free proxy servers from database.", len(proxyList))
	if host, port, e := net.SplitHostPort(conf.Args.Network.MasterProxyAddr); e != nil {
		log.Errorf("unable to parse master proxy address %s: %+v", conf.Args.Network.MasterProxyAddr, e)
	} else {
		proxyList = append(proxyList, types.NewProxyServer("config", host, port, "socks5", ""))
	}
	return proxyList[rand.Intn(len(proxyList))], nil
}

// PickGlobalProxy randomly chooses a global proxy from database.
func PickGlobalProxy() (proxy *types.ProxyServer, e error) {
	proxyList := make([]*types.ProxyServer, 0, 64)
	query := `
		SELECT 
			*
		FROM
			proxy_servers
		WHERE
			score_g >= ?
	`
	e = data.GormDB.Raw(query, conf.Args.Network.RotateProxyGlobalScoreThreshold).Scan(&proxyList).Error
	if e != nil {
		log.Println("failed to query global proxy server from database", e)
		return proxy, errors.WithStack(e)
	}
	log.Infof("successfully fetched %d free global proxy servers from database.", len(proxyList))
	str := strings.Split(conf.Args.Network.MasterProxyAddr, ":")
	proxyList = append(proxyList, types.NewProxyServer("config", str[0], str[1], "socks5", ""))
	return proxyList[rand.Intn(len(proxyList))], nil
}
