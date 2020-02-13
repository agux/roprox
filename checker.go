package main

import (
	"sync"
	"time"

	"github.com/carusyte/roprox/conf"
	"github.com/carusyte/roprox/data"
	"github.com/carusyte/roprox/types"
	"github.com/carusyte/roprox/util"
)

func check(wg *sync.WaitGroup) {
	defer wg.Done()

	chjobs := make(chan *types.ProxyServer, 128)
	probe(chjobs)
	collectStaleServers(chjobs)
}

func evictBrokenServers() {
	log.Debug("evicting broken servers...")
	delete := `delete from proxy_list where status = ? and last_scanned <= ? and score <= ?`
	r, e := data.DB.Exec(delete, types.FAIL, time.Now().Add(
		-time.Duration(conf.Args.EvictionTimeout)*time.Second).Format(util.DateTimeFormat),
		conf.Args.EvictionScoreThreshold)
	if e != nil {
		log.Errorln("failed to evict broken proxy servers", e)
		return
	}
	ra, e := r.RowsAffected()
	if e != nil {
		log.Warnf("unable to get rows affected after eviction", e)
		return
	}
	log.Debugf("%d broken servers evicted", ra)
}

func collectStaleServers(chjobs chan<- *types.ProxyServer) {
	//kickoff at once and repeatedly
	evictBrokenServers()
	queryStaleServers(chjobs)
	probeTk := time.NewTicker(time.Duration(conf.Args.ProbeInterval) * time.Second)
	evictTk := time.NewTicker(time.Duration(conf.Args.EvictionInterval) * time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-probeTk.C:
			queryStaleServers(chjobs)
		case <-evictTk.C:
			evictBrokenServers()
		case <-quit:
			probeTk.Stop()
			evictTk.Stop()
			return
		}
	}
}

func queryStaleServers(chjobs chan<- *types.ProxyServer) {
	log.Debug("collecting stale servers...")
	var list []*types.ProxyServer
	query := `SELECT 
					*
				FROM
					proxy_list
				WHERE
					status = ?
					or last_check <= ?
					order by last_check`
	_, e := data.DB.Select(&list, query, types.UNK, time.Now().Add(
		-time.Duration(conf.Args.ProbeInterval)*time.Second).Format(util.DateTimeFormat))
	if e != nil {
		log.Errorln("failed to query stale proxy servers", e)
		return
	}
	log.Debugf("%d stale servers pending for health check", len(list))
	for _, p := range list {
		chjobs <- p
	}
}

func probe(chjobs <-chan *types.ProxyServer) {
	for i := 0; i < conf.Args.ProbeSize; i++ {
		go func() {
			for ps := range chjobs {
				var e error
				if util.ValidateProxy(ps.Type, ps.Host, ps.Port) {
					_, e = data.DB.Exec(`update proxy_list set status = ?, `+
						`suc = suc+1, score = suc/(suc+fail)*100, `+
						`last_check = ? where host = ? and port = ?`,
						types.OK, util.Now(), ps.Host, ps.Port)
				} else {
					_, e = data.DB.Exec(`update proxy_list set status = ?, `+
						`fail = fail+1, score = suc/(suc+fail)*100, `+
						`last_check = ? where host = ? and port = ?`,
						types.FAIL, util.Now(), ps.Host, ps.Port)
				}
				if e != nil {
					log.Errorln("failed to update proxy server status", e)
				}
			}
		}()
	}
}
