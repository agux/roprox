package network

import (
	"time"

	"github.com/agux/roprox/internal/data"
	"github.com/agux/roprox/internal/types"
)

// UpdateProxyScore for the specified proxy.
func UpdateProxyScore(p *types.ProxyServer, success bool) {
	if p == nil || p.ID <= 0 {
		return
	}
	var e error
	if success {
		e = data.GormDB.Exec(`
			update proxy_servers set suc = suc + 1, score = (suc+1)/(suc+1+fail)*100, updated_at = ?
				where id = ?
			`, p.ID, time.Now()).Error
	} else {
		e = data.GormDB.Exec(`
			update proxy_servers set fail = fail + 1, score = suc/(suc+fail+1)*100, updated_at = ?
				where id = ?
			`, p.ID, time.Now()).Error
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
