package network

import (
	"math/rand"
	"net"
	"reflect"
	"regexp"
	"runtime"
	"time"

	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/data"
	"github.com/agux/roprox/internal/types"
	"github.com/pkg/errors"
)

type rebounceIp func(ps *types.ProxyServer) (yourIp string, e error)

var rebouncers = []rebounceIp{
	httpbin,
	icanhazip,
	ipify,
	ifconfigme,
	ipinfo,
	seeip,
	myexternalip,
}

// DomainOf the specified url.
func DomainOf(url string) (domain string) {
	r := regexp.MustCompile(`//([^/]*)/`).FindStringSubmatch(url)
	if len(r) > 0 {
		domain = r[len(r)-1]
	}
	return
}

// ValidateProxy checks the status of remote listening port,
// and further checks if it's a valid proxy server by calling
// 3rd-party services that return source IP for validation.
// Note that some of the 3rd-party proxy may further relay the request to downstream nodes,
// therefore as long as the returned IP address is not the same as our exposed external IP,
// It will be considered a qualified proxy server at this point.
func ValidateProxy(ps *types.ProxyServer, probeTimeout int) bool {
	host := ps.Host
	port := ps.Port
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

	getOutboundIp := rebouncers[rand.Intn(len(rebouncers))]
	funcName := runtime.FuncForPC(reflect.ValueOf(getOutboundIp).Pointer()).Name()
	if ip, e := getOutboundIp(ps); e != nil {
		log.Tracef("%s failed to validate via %v", addr, funcName)
		return false
	} else if host == ip {
		// IP rebouncer returns the same IP as proxy host, means the proxy is valid to some extent
		return true
	} else {
		// as long as the returned IP address is not the same as our outbound IP,
		// It shall be considered a qualified proxy server at this point (return true rather than false)
		if trueIp, e := getOutboundIp(nil); e != nil {
			log.Warnf("validating proxy %s, failed to get true outbound IP via %v.", addr, funcName)
			return false
		} else {
			return ip != trueIp
		}
	}
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
	return proxyList[rand.Intn(len(proxyList))], nil
}
