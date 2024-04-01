package ua

import (
	"math/rand"
	"sync"
	"time"

	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/types"
	"github.com/pkg/errors"
)

var (
	agentPool []string
	uaLock    = sync.RWMutex{}
)

// PickUserAgent selects a random user agent string from the pool.
// If the pool is empty or the user agent strings are outdated, it attempts to refresh them from remote service.
// The noRefresh parameter can be set to true to avoid refreshing the pool even if it's outdated.
//
// Returns a user agent string or an error if the operation fails.
func PickUserAgent(noRefresh bool) (ua string, e error) {
	uaLock.Lock()
	defer uaLock.Unlock()

	if len(agentPool) > 0 {
		return agentPool[rand.Intn(len(agentPool))], nil
	}

	url := conf.Args.DataSource.UserAgents
	var uaFetcher userAgentFetcher
	for _, f := range uaFetcherList {
		if f.urlMatch(url) {
			uaFetcher = f
			break
		}
	}
	if uaFetcher == nil {
		e = errors.Errorf("unable to find user agent fetcher for the configured URL: %s", url)
		return
	}

	//first, load from database
	var agents []*types.UserAgent
	if agents, e = uaFetcher.load(); e != nil {
		return
	}
	if len(agents) > 0 {
		ua = agents[rand.Intn(len(agents))].UserAgent.String
	}
	if !noRefresh {
		outdated := false
		if len(agents) != 0 {
			if outdated, e = uaFetcher.outdated(agents); e != nil {
				return
			}
		}
		//if none, or outdated, refresh table from remote server
		if outdated || len(agents) == 0 {
			//download sample file and load into database server
			log.Info("fetching user agent list from remote server...")
			if agents, e = uaFetcher.get(); e != nil {
				return
			}
			log.Infof("successfully fetched %d user agents from remote server.", len(agents))
			//reload agents from database
			if agents, e = uaFetcher.load(); e != nil {
				return
			}
		}
		for _, a := range agents {
			agentPool = append(agentPool, a.UserAgent.String)
		}
	}

	if len(agentPool) == 0 {
		e = errors.New("user agent strings are not available at this moment")
		log.Warn(e)
		return
	}
	return agentPool[rand.Intn(len(agentPool))], nil
}

func GetUserAgent(proxyURL string) (string, error) {
	return uaCache.GetUserAgent(proxyURL)
}

type userAgentCache struct {
	rwMtx            sync.RWMutex
	upgMtx           sync.Mutex // used during upgrading from read lock to write lock
	userAgentBinding map[string]string
	cacheLastUpdated int64
}

func (cache *userAgentCache) r_lock() {
	cache.rwMtx.RLock()
}

func (cache *userAgentCache) r_unlock() {
	cache.rwMtx.RUnlock()
}

func (cache *userAgentCache) upgradeToWriteLock() {
	cache.upgMtx.Lock()   // Ensure that only one goroutine can attempt to upgrade from RLock.
	cache.rwMtx.RUnlock() // Release the read lock.
	cache.rwMtx.Lock()    // Acquire the write lock.
}

func (cache *userAgentCache) downgradeAndUnlock() {
	cache.rwMtx.Unlock()  // Release the write lock.
	cache.upgMtx.Unlock() // Release the upgradeMutex to allow other upgrades.
}

func (cache *userAgentCache) GetUserAgent(proxyURL string) (string, error) {
	cache.r_lock() // FIXME  easily get dead lock at this point
	if value, ok := cache.userAgentBinding[proxyURL]; ok {
		cache.r_unlock()
		return value, nil
	} else {
		cache.upgradeToWriteLock()
		defer cache.downgradeAndUnlock()
		if ua, e := PickUserAgent(false); e != nil {
			return ua, e
		} else {
			cache.userAgentBinding[proxyURL] = ua
			cache.cacheLastUpdated = time.Now().Unix()
			return ua, nil
		}
	}
}
