package proxy

import (
	"crypto/tls"
	"sync"
	"time"

	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/data"
	"github.com/agux/roprox/internal/types"
	"gorm.io/gorm"
)

var (
	certStore  map[string]tls.Certificate
	proxyCache *proxyServerCache
)

type proxyServerCache struct {
	sync.RWMutex
	proxyServers     []types.ProxyServer
	cacheLastUpdated int64
}

// RefreshData refreshes the data from the database.
func (cache *proxyServerCache) RefreshData(db *gorm.DB) error {
	var servers []types.ProxyServer
	currentTime := time.Now().Unix()

	cache.Lock()

	if err := db.Find(&servers, "score >= ?",
		conf.Args.Network.RotateProxyScoreThreshold).Error; err != nil {
		return err
	}
	log.Infof("reloaded %d qualified proxy from the backend pool", len(servers))
	cache.proxyServers = servers
	cache.cacheLastUpdated = currentTime

	cache.Unlock()

	return nil
}

// GetData safely returns the in-memory data.
func (cache *proxyServerCache) GetData() []types.ProxyServer {
	cache.RLock()
	defer cache.RUnlock()
	return cache.proxyServers
}

func init() {
	certStore = make(map[string]tls.Certificate)

	if !conf.Args.Proxy.BypassTraffic {
		refreshProxyCache()
	}
}

func refreshProxyCache() {
	proxyCache = &proxyServerCache{}
	if e := proxyCache.RefreshData(data.GormDB); e != nil {
		log.Fatal("failed to load proxy from database", e)
	}

	// Set up a ticker to refresh the proxy cache
	ticker := time.NewTicker(time.Duration(conf.Args.Proxy.MemCacheLifespan) * time.Second)
	go func() {
		for range ticker.C {
			go func() {
				if err := proxyCache.RefreshData(data.GormDB); err != nil {
					log.Errorln("failed to refresh proxy list from database", err)
				}
			}()
		}
	}()
}
