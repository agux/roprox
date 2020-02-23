package main

import (
	"sync"
	"time"

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/data"
	"github.com/agux/roprox/types"
	"github.com/agux/roprox/util"
)

func check(wg *sync.WaitGroup) {
	defer wg.Done()

	lch := make(chan *types.ProxyServer, 8192)
	gch := make(chan *types.ProxyServer, 8192)
	probeLocal(lch)
	probeGlobal(gch)
	Tick(lch, gch)
}

func evictBrokenServers() {
	log.Debug("evicting broken servers...")
	delete := `delete from proxy_list where status = ? and score <= ? ` +
		`and status_g = ? and score_g <= ? ` +
		`and last_scanned <= ? `
	r, e := data.DB.Exec(delete, types.FAIL, conf.Args.EvictionScoreThreshold,
		types.FAIL, conf.Args.EvictionScoreThreshold,
		time.Now().Add(-time.Duration(conf.Args.EvictionTimeout)*time.Second).Format(util.DateTimeFormat))
	if e != nil {
		log.Errorln("failed to evict broken proxy servers", e)
		return
	}
	ra, e := r.RowsAffected()
	if e != nil {
		log.Errorf("unable to get rows affected after eviction: %+v", e)
		return
	}
	log.Infof("%d broken servers evicted", ra)
}

func Tick(lch chan<- *types.ProxyServer, gch chan<- *types.ProxyServer) {
	//kickoff at once and repeatedly
	evictBrokenServers()
	queryServersForLocal(lch)
	queryServersForGlobal(gch)
	probeTk := time.NewTicker(time.Duration(conf.Args.ProbeInterval) * time.Second)
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
					proxy_list
				WHERE
					status = ?
					or last_check <= ?
					order by last_check`
	_, e := data.DB.Select(&list, query, types.UNK, time.Now().Add(
		-time.Duration(conf.Args.ProbeInterval)*time.Second).Format(util.DateTimeFormat))
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
					proxy_list
				WHERE
					status_g = ?
					or (last_check <= ? and (suc_g > 0 or fail <= ?))
					order by last_check`
	_, e := data.DB.Select(&list, query, types.UNK,
		time.Now().Add(-time.Duration(conf.Args.GlobalProbeInterval)*time.Second).Format(util.DateTimeFormat),
		conf.Args.GlobalProbeRetry,
	)
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
	for i := 0; i < conf.Args.ProbeSize; i++ {
		time.Sleep(time.Millisecond * 3500)
		go func() {
			for ps := range chjobs {
				var e error
				if util.ValidateProxy(ps.Type, ps.Host, ps.Port,
					`http://www.baidu.com`, "#wrapper", conf.Args.ProbeTimeout) {
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
				if util.ValidateProxy(ps.Type, ps.Host, ps.Port,
					`http://www.google.com`, `#tsf`, conf.Args.GlobalProbeTimeout) {
					_, e = data.DB.Exec(`update proxy_list set status_g = ?, `+
						`suc_g = suc_g+1, score_g = suc_g/(suc_g+fail_g)*100, `+
						`last_check = ? where host = ? and port = ?`,
						types.OK, util.Now(), ps.Host, ps.Port)
				} else {
					_, e = data.DB.Exec(`update proxy_list set status_g = ?, `+
						`fail_g = fail_g+1, score_g = suc_g/(suc_g+fail_g)*100, `+
						`last_check = ? where host = ? and port = ?`,
						types.FAIL, util.Now(), ps.Host, ps.Port)
				}
				if e != nil {
					log.Errorln("failed to update proxy server score", e)
				}
			}
		}()
	}
}
