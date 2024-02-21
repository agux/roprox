package network

import (
	"github.com/agux/roprox/internal/data"
	"github.com/agux/roprox/internal/types"
)

// UpdateProxyScore for the specified proxy.
func UpdateProxyScore(p *types.ProxyServer, success bool) {
	if p == nil {
		return
	}
	var e error
	if success {
		e = data.GormDB.Exec(`
			update proxy_servers set suc = suc + 1, score = suc/(suc+fail)*100
				where host = ? and port = ?
			`,
			p.Host, p.Port).Error
	} else {
		e = data.GormDB.Exec(`
			update proxy_servers set fail = fail + 1, score = suc/(suc+fail)*100
				where host = ? and port = ?
			`, p.Host, p.Port).Error
	}
	if e != nil {
		log.Errorf("failed to increase scoring counter for proxy %+v", p)
	}
}

// UpdateProxyScoreGlobal updates globa score for the specified proxy.
func UpdateProxyScoreGlobal(p *types.ProxyServer, success bool) {
	if p == nil {
		return
	}
	var e error
	if success {
		e = data.GormDB.Exec(`
			update proxy_servers set suc_g = suc_g + 1, score_g = suc_g/(suc_g+fail_g)*100
				where host = ? and port = ?
			`, p.Host, p.Port).Error
	} else {
		e = data.GormDB.Exec(`
			update proxy_servers set fail_g = fail_g + 1, score_g = suc_g/(suc_g+fail_g)*100
				where host = ? and port = ?
			`, p.Host, p.Port).Error
	}
	if e != nil {
		log.Errorf("failed to increase scoring counter for proxy %+v", p)
	}
}
