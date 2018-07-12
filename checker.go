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
	ticker := time.NewTicker(time.Duration(conf.Args.CheckInterval) * time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			logrus.Debug("collecting stale servers...")
			var list []*types.ProxyServer
			query := `SELECT 
					      *
					  FROM
					      proxy_list
					  WHERE
					      status = ?
						  and last_check <= ?
						  order by last_check`
			_, e := db.Select(&list, query, types.OK, time.Now().Add(
				-time.Duration(conf.Args.CheckInterval)*time.Second).Format(util.DateTimeFormat))
			if e != nil {
				logrus.Errorln("failed to query stale proxy servers", e)
				continue
			}
			logrus.Debugf("%d stale servers pending for health check", len(list))
			for _, p := range list {
				chjobs <- p
			}
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func probe(chjobs <-chan *types.ProxyServer) {
	for i := 0; i < conf.Args.CheckerPoolSize; i++ {
		go func() {
			for job := range chjobs {
				status := types.Fail
				if util.CheckRemote(job.Host, job.Port) {
					status = types.OK
				}
				db.Exec(`update proxy_list set status = ? and last_check = ? where host = ? and port = ?`,
					status, util.Now(), job.Host, job.Port)
			}
		}()
	}
}
