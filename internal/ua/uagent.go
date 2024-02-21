package ua

import (
	"math/rand"
	"sync"

	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/types"
	"github.com/pkg/errors"
)

var (
	agentPool []string
	uaLock    = sync.RWMutex{}
)

// PickUserAgent picks a user agent string from the pool randomly.
// if the pool is not populated, it will trigger the initialization process
// to fetch user agent lists from remote server.
func PickUserAgent() (ua string, e error) {
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
	if len(agentPool) == 0 {
		e = errors.New("user agent strings are not available at this moment")
		log.Warn(e)
		return
	}
	return agentPool[rand.Intn(len(agentPool))], nil
}
