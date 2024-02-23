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
	probeLocal(lch)
	Tick(lch)
}

func evictBrokenServers() {
	log.Debug("evicting broken servers...")
	delete := `
		delete from proxy_servers where status = ? and score <= ? 
			and status_g = ? and score_g <= ? 
			and last_scanned <= ? 
	`
	db := data.GormDB.Exec(delete, types.FAIL, conf.Args.Proxy.EvictionScoreThreshold,
		types.FAIL, conf.Args.Proxy.EvictionScoreThreshold,
		time.Now().Add(-time.Duration(conf.Args.Proxy.EvictionTimeout)*time.Second).Format(util.DateTimeFormat))
	e := db.Error
	if e != nil {
		log.Errorln("failed to evict broken proxy servers", e)
		return
	}
	ra := db.RowsAffected
	log.Infof("%d broken servers evicted", ra)
}

func Tick(lch chan<- *types.ProxyServer) {
	//kickoff at once and repeatedly
	evictBrokenServers()
	queryServersForLocal(lch)
	probeTk := time.NewTicker(time.Duration(conf.Args.Probe.Interval) * time.Second)
	evictTk := time.NewTicker(time.Duration(conf.Args.Proxy.EvictionInterval) * time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-probeTk.C:
			queryServersForLocal(lch)
		case <-evictTk.C:
			evictBrokenServers()
		case <-quit:
			probeTk.Stop()
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
		time.Now().Add(-time.Duration(conf.Args.Probe.Interval)*time.Second).Format(util.DateTimeFormat),
		conf.Args.Probe.FailThreshold).Scan(&list).Error
	if e != nil {
		log.Errorln("failed to query proxy servers for local probe", e)
		return
	}
	log.Debugf("%d stale servers pending for health check (local)", len(list))
	for _, p := range list {
		ch <- p
	}
}

func probeLocal(chjobs <-chan *types.ProxyServer) {
	for i := 0; i < conf.Args.Probe.Size; i++ {
		time.Sleep(time.Millisecond * 3500)
		go func() {
			for ps := range chjobs {
				var e error
				now := util.Now()
				if network.ValidateProxy(ps.Type, ps.Host, ps.Port,
					conf.Args.Probe.CheckUrl, conf.Args.Probe.CheckKeyword, conf.Args.Probe.Timeout) {
					e = data.GormDB.Exec(`update proxy_servers set status = ?, `+
						`suc = suc+1, score = (suc+1)/(suc+1+fail)*100, `+
						`updated_at = ?, last_check = ? where id = ?`,
						types.OK, now, now, ps.ID).Error
				} else {
					e = data.GormDB.Exec(`update proxy_servers set status = ?, `+
						`fail = fail+1, score = suc/(suc+fail+1)*100, `+
						`updated_at = ?, last_check = ? where id = ?`,
						types.FAIL, now, now, ps.ID).Error
				}
				if e != nil {
					log.Errorln("failed to update proxy server score", e)
				}
			}
		}()
	}
}
