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

	chpx := make(chan *t.ProxyServer, 128)
	chjobs := make(chan string, 16)

	var wgs sync.WaitGroup
	wgs.Add(1)
	go collectProxyServer(&wgs, chpx)
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
		go func(fs t.FetcherSpec) {
			for {
				select {
				case <-ticker.C:
					logrus.Debugf("refreshing list from source %s", fs.UID())
					chjobs <- fs.UID()
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}(fs)
	}
}

func launchScanners(chjobs <-chan string, chpx chan<- *t.ProxyServer) {
	for i := 0; i < conf.Args.ScannerPoolSize; i++ {
		go func() {
			for uid := range chjobs {
				fetcher.Fetch(chpx, proxies[uid])
			}
		}()
	}
}

func collectProxyServer(wgs *sync.WaitGroup, chpx <-chan *t.ProxyServer) {
	defer wgs.Done()
	size := 32
	bucket := make([]*t.ProxyServer, 0, size)
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
			if len(bucket) > 0 {
				saveProxyServer(bucket)
				bucket = make([]*t.ProxyServer, 0, size)
			}
		case ps := <-chpx:
			if len(bucket) < size {
				bucket = append(bucket, ps)
			}
			saveProxyServer(bucket)
			bucket = make([]*t.ProxyServer, 0, size)
		}
	}
}

func saveProxyServer(bucket []*t.ProxyServer) {
	if len(bucket) == 0 {
		return
	}

	valueStrings := make([]string, 0, len(bucket))
	valueArgs := make([]interface{}, 0, len(bucket)*7)
	for _, e := range bucket {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs, e.Source)
		valueArgs = append(valueArgs, e.Host)
		valueArgs = append(valueArgs, e.Port)
		valueArgs = append(valueArgs, e.Type)
		valueArgs = append(valueArgs, e.Status)
		valueArgs = append(valueArgs, e.LastCheck)
		valueArgs = append(valueArgs, e.LastScanned)
	}
	stmt := fmt.Sprintf("INSERT INTO proxy_list (source,host,port,type,status,last_check,last_scanned) VALUES %s "+
		"on duplicate key update status=values(status),last_check=values(last_check),last_scanned=values(last_scanned)",
		strings.Join(valueStrings, ","))
	retry := 10
	rt := 0
	for ; rt < retry; rt++ {
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
