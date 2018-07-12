package util

import (
	"net"
	"regexp"
	"time"

	"github.com/carusyte/roprox/conf"
	"github.com/sirupsen/logrus"
)

//DomainOf the specified url.
func DomainOf(url string) (domain string) {
	r := regexp.MustCompile(`//([^/]*)/`).FindStringSubmatch(url)
	if len(r) > 0 {
		domain = r[len(r)-1]
	}
	return
}

//CheckRemote checks the status of remote listening port
func CheckRemote(host, port string) bool {
	timeout := time.Second * time.Duration(conf.Args.CheckTimeout)
	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		logrus.Warnf("%s failed: %+v", addr, err)
		return false
	}
	if conn != nil {
		logrus.Infof("%s success", addr)
		conn.Close()
		return true
	}
	logrus.Warnf("%s failed", addr)
	return false
}
