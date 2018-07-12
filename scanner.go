package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/carusyte/roprox/conf"
	"github.com/carusyte/roprox/fetcher"
	t "github.com/carusyte/roprox/types"
	"github.com/sirupsen/logrus"
)

func scan(wg *sync.WaitGroup) {
	defer wg.Done()

	chpx := make(chan []*t.ProxyServer, 64)
	chjobs := make(chan string, 16)

	var wgs sync.WaitGroup
	wgs.Add(1)
	go saveProxyServer(&wgs, chpx)
	launchScanners(chjobs, chpx)
	launchDispatcher(chjobs)

	wgs.Wait()
	close(chpx)
	close(chjobs)
}

func launchDispatcher(chjobs chan<- string) {
	for _, fs := range fslist {
		//kick start refreshing at once
		chjobs <- fs.UID()
		ticker := time.NewTicker(time.Duration(fs.RefreshInterval()) * time.Minute)
		quit := make(chan struct{})
		go func() {
			for {
				select {
				case <-ticker.C:
					chjobs <- fs.UID()
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()
	}
}

func launchScanners(chjobs <-chan string, chpx chan<- []*t.ProxyServer) {
	for i := 0; i < conf.Args.ScannerPoolSize; i++ {
		go func() {
			for job := range chjobs {
				fetcher.Fetch(chpx, proxies[job])
			}
		}()
	}
}

func saveProxyServer(wgs *sync.WaitGroup, chpx <-chan []*t.ProxyServer) {
	defer wgs.Done()
	for ps := range chpx {
		if len(ps) == 0 {
			continue
		}
		retry := 10
		rt := 0
		for ; rt < retry; rt++ {
			valueStrings := make([]string, 0, len(ps))
			valueArgs := make([]interface{}, 0, len(ps)*7)
			for _, e := range ps {
				valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?)")
				valueArgs = append(valueArgs, e.Source)
				valueArgs = append(valueArgs, e.Host)
				valueArgs = append(valueArgs, e.Port)
				valueArgs = append(valueArgs, e.Type)
				valueArgs = append(valueArgs, e.Status)
				valueArgs = append(valueArgs, e.LastCheck)
				valueArgs = append(valueArgs, e.LastScanned)
			}
			stmt := fmt.Sprintf("INSERT IGNORE INTO proxy_list (source,host,port,type,status,last_check,last_scanned) VALUES %s",
				strings.Join(valueStrings, ","))
			_, err := db.Exec(stmt, valueArgs...)
			if err != nil {
				logrus.Println(err)
				if strings.Contains(err.Error(), "Deadlock") {
					continue
				} else {
					logrus.Errorln("failed to update proxy_list", err)
					break
				}
			}
			break
		}
		if rt >= retry {
			logrus.Errorln("failed to update proxy_list")
		}
	}
}
