package network

import (
	"github.com/agux/roprox/data"
	"github.com/agux/roprox/types"
)

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
		log.Printf("failed to increase scoring counter for proxy %+v", p)
	}
}

//UpdateProxyScoreGlobal updates globa score for the specified proxy.
func UpdateProxyScoreGlobal(p *types.ProxyServer, success bool) {
	if p == nil {
		return
	}
	var e error
	if success {
		_, e = data.DB.Exec(`update proxy_list set suc_g = suc_g + 1, score_g = suc_g/(suc_g+fail_g)*100 `+
			`where host = ? and port = ?`, p.Host, p.Port)
	} else {
		_, e = data.DB.Exec(`update proxy_list set fail_g = fail_g + 1, score_g = suc_g/(suc_g+fail_g)*100 `+
			`where host = ? and port = ?`, p.Host, p.Port)
	}
	if e != nil {
		log.Printf("failed to increase scoring counter for proxy %+v", p)
	}
}
