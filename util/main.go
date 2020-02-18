package util

import (
	"github.com/carusyte/roprox/data"
	"github.com/carusyte/roprox/logging"
	"github.com/carusyte/roprox/types"
)

var log = logging.Logger

//UpdateProxyScore for the specified proxy.
func UpdateProxyScore(p *types.ProxyServer, success bool) {
	if p == nil {
		return
	}
	var e error
	if success {
		_, e = data.DB.Exec(`update proxy_list set suc = suc + 1, score = suc/(suc+fail)*100 `+
			`where host = ? and port = ?`, p.Host, p.Port)
	} else {
		_, e = data.DB.Exec(`update proxy_list set fail = fail + 1, score = suc/(suc+fail)*100 `+
			`where host = ? and port = ?`, p.Host, p.Port)
	}
	if e != nil {
		log.Printf("failed to increase fail counter for proxy %+v", p)
	}
}