package main

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/carusyte/roprox/conf"
	"github.com/carusyte/roprox/types"
	"github.com/carusyte/roprox/util"
)

func check(wg *sync.WaitGroup) {
	defer wg.Done()

	chjobs := make(chan *types.ProxyServer, 128)
	probe(chjobs)
	collectStaleServers(chjobs)

}

func collectStaleServers(chjobs chan<- *types.ProxyServer) {
	//kickoff at once and repeatedly
	queryStaleServers(chjobs)
	ticker := time.NewTicker(time.Duration(conf.Args.CheckInterval) * time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			queryStaleServers(chjobs)
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func queryStaleServers(chjobs chan<- *types.ProxyServer) {
	logrus.Debug("collecting stale servers...")
	var list []*types.ProxyServer
	query := `SELECT 
					*
				FROM
					proxy_list
				WHERE
					last_check <= ?
					order by last_check`
	//TODO do we need to filter out failed servers to lower the workload?
	_, e := db.Select(&list, query, time.Now().Add(
		-time.Duration(conf.Args.CheckInterval)*time.Second).Format(util.DateTimeFormat))
	if e != nil {
		logrus.Errorln("failed to query stale proxy servers", e)
		return
	}
	logrus.Debugf("%d stale servers pending for health check", len(list))
	for _, p := range list {
		chjobs <- p
	}
}

func probe(chjobs <-chan *types.ProxyServer) {
	for i := 0; i < conf.Args.CheckerPoolSize; i++ {
		go func() {
			for ps := range chjobs {
				status := types.FAIL
				if util.ValidateProxy(ps.Type, ps.Host, ps.Port) {
					status = types.OK
				}
				_, e := db.Exec(`update proxy_list set status = ?, last_check = ? where host = ? and port = ?`,
					status, util.Now(), ps.Host, ps.Port)
				if e != nil {
					logrus.Errorln("failed to update proxy server status", e)
				}
			}
		}()
	}
}
