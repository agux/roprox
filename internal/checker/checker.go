package checker

import (
	"sync"
	"time"

	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/data"
	"github.com/agux/roprox/internal/logging"
	"github.com/agux/roprox/internal/network"
	"github.com/agux/roprox/internal/types"
	"github.com/agux/roprox/internal/util"
)

var log = logging.Logger

func Check(wg *sync.WaitGroup) {
	defer wg.Done()

	lch := make(chan *types.ProxyServer, 8192)
	gch := make(chan *types.ProxyServer, 8192)
	probeLocal(lch)
	probeGlobal(gch)
	Tick(lch, gch)
}

func evictBrokenServers() {
	log.Debug("evicting broken servers...")
	delete := `
		delete from proxy_servers where status = ? and score <= ? 
			and status_g = ? and score_g <= ? 
			and last_scanned <= ? 
	`
	db := data.GormDB.Exec(delete, types.FAIL, conf.Args.EvictionScoreThreshold,
		types.FAIL, conf.Args.EvictionScoreThreshold,
		time.Now().Add(-time.Duration(conf.Args.EvictionTimeout)*time.Second).Format(util.DateTimeFormat))
	e := db.Error
	if e != nil {
		log.Errorln("failed to evict broken proxy servers", e)
		return
	}
	ra := db.RowsAffected
	log.Infof("%d broken servers evicted", ra)
}

func Tick(lch chan<- *types.ProxyServer, gch chan<- *types.ProxyServer) {
	//kickoff at once and repeatedly
	evictBrokenServers()
	queryServersForLocal(lch)
	queryServersForGlobal(gch)
	probeTk := time.NewTicker(time.Duration(conf.Args.LocalProbeInterval) * time.Second)
	probeTkG := time.NewTicker(time.Duration(conf.Args.GlobalProbeInterval) * time.Second)
	evictTk := time.NewTicker(time.Duration(conf.Args.EvictionInterval) * time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-probeTk.C:
			queryServersForLocal(lch)
		case <-probeTkG.C:
			queryServersForGlobal(gch)
		case <-evictTk.C:
			evictBrokenServers()
		case <-quit:
			probeTk.Stop()
			probeTkG.Stop()
			evictTk.Stop()
			return
		}
	}
}

func queryServersForLocal(ch chan<- *types.ProxyServer) {
	log.Debug("collecting servers for local probe...")
	var list []*types.ProxyServer
	query := `SELECT 
					*
				FROM
					proxy_servers
				WHERE
					status = ?
					or (last_check <= ? and (suc > 0 or fail <= ?))
					order by last_check`
	e := data.GormDB.Raw(query, types.UNK,
		time.Now().Add(-time.Duration(conf.Args.LocalProbeInterval)*time.Second).Format(util.DateTimeFormat),
		conf.Args.LocalProbeRetry).Scan(&list).Error
	if e != nil {
		log.Errorln("failed to query proxy servers for local probe", e)
		return
	}
	log.Debugf("%d stale servers pending for health check (local)", len(list))
	for _, p := range list {
		ch <- p
	}
}

func queryServersForGlobal(ch chan<- *types.ProxyServer) {
	log.Debug("collecting servers for global probe...")
	var list []*types.ProxyServer
	//TODO might need to use separate 'last_check' field for globa/local proxy checking
	query := `SELECT 
					*
				FROM
					proxy_servers
				WHERE
					status_g = ?
					or (last_check <= ? and (suc_g > 0 or fail_g <= ?))
					order by last_check`
	e := data.GormDB.Raw(query, types.UNK,
		time.Now().Add(-time.Duration(conf.Args.GlobalProbeInterval)*time.Second).Format(util.DateTimeFormat),
		conf.Args.GlobalProbeRetry).Scan(&list).Error
	if e != nil {
		log.Errorln("failed to query proxy servers for global probe", e)
		return
	}
	log.Debugf("%d stale servers pending for health check (global)", len(list))
	for _, p := range list {
		ch <- p
	}
}

func probeLocal(chjobs <-chan *types.ProxyServer) {
	for i := 0; i < conf.Args.LocalProbeSize; i++ {
		time.Sleep(time.Millisecond * 3500)
		go func() {
			for ps := range chjobs {
				var e error
				if network.ValidateProxy(ps.Type, ps.Host, ps.Port,
					`http://www.baidu.com`, "#wrapper", conf.Args.LocalProbeTimeout) {
					e = data.GormDB.Exec(`update proxy_servers set status = ?, `+
						`suc = suc+1, score = (suc+1)/(suc+1+fail)*100, `+
						`last_check = ? where id = ?`,
						types.OK, util.Now(), ps.ID).Error
				} else {
					e = data.GormDB.Exec(`update proxy_servers set status = ?, `+
						`fail = fail+1, score = suc/(suc+fail+1)*100, `+
						`last_check = ? where id = ?`,
						types.FAIL, util.Now(), ps.ID).Error
				}
				if e != nil {
					log.Errorln("failed to update proxy server score", e)
				}
			}
		}()
	}
}

func probeGlobal(ch <-chan *types.ProxyServer) {
	for i := 0; i < conf.Args.GlobalProbeSize; i++ {
		time.Sleep(time.Millisecond * 3500)
		go func() {
			for ps := range ch {
				var e error
				if network.ValidateProxy(ps.Type, ps.Host, ps.Port,
					`http://www.google.com`, `#tsf`, conf.Args.GlobalProbeTimeout) {
					e = data.GormDB.Exec(`update proxy_servers set status_g = ?, `+
						`suc_g = suc_g+1, score_g = (suc_g+1)/(suc_g+1+fail_g)*100, `+
						`last_check = ? where id = ?`,
						types.OK, util.Now(), ps.ID).Error
				} else {
					e = data.GormDB.Exec(`update proxy_servers set status_g = ?, `+
						`fail_g = fail_g+1, score_g = suc_g/(suc_g+fail_g+1)*100, `+
						`last_check = ? where id = ?`,
						types.FAIL, util.Now(), ps.ID).Error
				}
				if e != nil {
					log.Errorln("failed to update proxy server score", e)
				}
			}
		}()
	}
}
