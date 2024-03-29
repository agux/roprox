package scanner

import (
	"math/rand"
	"sync"
	"time"

	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/data"
	"github.com/agux/roprox/internal/fetcher"
	"github.com/agux/roprox/internal/logging"
	t "github.com/agux/roprox/internal/types"
	"gorm.io/gorm/clause"
)

var log = logging.Logger

func Scan(wg *sync.WaitGroup) {
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
	for _, fs := range fetcher.FetcherList {
		//kick start refreshing at once
		chjobs <- fs.UID()
		ticker := time.NewTicker(time.Duration(fs.RefreshInterval()) * time.Minute)
		quit := make(chan struct{})
		go func(fs t.FetcherSpec) {
			for {
				select {
				case <-ticker.C:
					log.Debugf("refreshing list from source %s", fs.UID())
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
	for i := 0; i < conf.Args.Scanner.PoolSize; i++ {
		go func() {
			for uid := range chjobs {
				fetcher.Fetch(chpx, fetcher.FetcherMap[uid])
			}
		}()
		time.Sleep(time.Millisecond * time.Duration(5000+rand.Intn(20000)))
	}
}

func collectProxyServer(wgs *sync.WaitGroup, chpx <-chan *t.ProxyServer) {
	defer wgs.Done()
	size := 32
	bucket := make([]*t.ProxyServer, 0, size)
	ticker := time.NewTicker(time.Second * 5)
out:
	for {
		select {
		case <-ticker.C:
			if len(bucket) > 0 {
				saveProxyServer(bucket)
				bucket = make([]*t.ProxyServer, 0, size)
			}
		case ps, ok := <-chpx:
			if ok {
				bucket = append(bucket, ps)
				if len(bucket) >= size {
					saveProxyServer(bucket)
					bucket = make([]*t.ProxyServer, 0, size)
				}
			} else {
				//channel has been closed
				ticker.Stop()
				if len(bucket) > 0 {
					saveProxyServer(bucket)
					bucket = nil
				}
				break out
			}
		}
	}
}

func saveProxyServer(bucket []*t.ProxyServer) {
	if len(bucket) == 0 {
		return
	}

	if err := data.GormDB.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(bucket, 32).Error; err != nil {
		log.Errorf("failed to insert/update proxy server: %v", err)
	}
}

// func saveProxyServer(bucket []*t.ProxyServer) {
// 	if len(bucket) == 0 {
// 		return
// 	}

// 	valueStrings := make([]string, 0, len(bucket))
// 	valueArgs := make([]interface{}, 0, len(bucket)*9)
// 	for _, el := range bucket {
// 		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?)")
// 		valueArgs = append(valueArgs, el.Source)
// 		valueArgs = append(valueArgs, el.Host)
// 		valueArgs = append(valueArgs, el.Port)
// 		valueArgs = append(valueArgs, el.Type)
// 		valueArgs = append(valueArgs, el.Loc)
// 		valueArgs = append(valueArgs, el.Status)
// 		valueArgs = append(valueArgs, el.StatusG)
// 		valueArgs = append(valueArgs, el.LastCheck)
// 		valueArgs = append(valueArgs, el.LastScanned)
// 	}
// 	stmt := fmt.Sprintf("INSERT IGNORE INTO proxy_list ("+
// 		"source,host,port,type,loc,status,status_g,last_check,last_scanned) VALUES %s",
// 		// "on duplicate key update status=values(status),last_check=values(last_check),last_scanned=values(last_scanned)",
// 		strings.Join(valueStrings, ","))
// 	retry := 10
// 	rt := 0
// 	for ; rt < retry; rt++ {
// 		_, err := data.DB.Exec(stmt, valueArgs...)
// 		if err != nil {
// 			log.Warn(err)
// 			if strings.Contains(err.Error(), "Deadlock") {
// 				continue
// 			} else {
// 				log.Errorln("failed to update proxy_list", err)
// 				break
// 			}
// 		}
// 		break
// 	}
// 	if rt >= retry {
// 		log.Errorln("failed to update proxy_list")
// 	}
// }
