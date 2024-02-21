package ua

import (
	"time"

	"github.com/agux/roprox/internal/logging"
	"github.com/agux/roprox/internal/types"
	"github.com/ssgreg/repeat"
)

var uaFetcherList = []userAgentFetcher{
	whatIsMyBrowser{},
	willsHouse{},
	userAgentsMe{},
}

var log = logging.Logger

const dateTimeFormat = "2006-01-02 15:04:05"

type userAgentFetcher interface {
	//urlMatch matches the URL for remote source and returns whether this fetcher can handle it.
	urlMatch(url string) (matched bool)
	//outdated returns whether the collection of user agent data should be refreshed.
	outdated(agents []*types.UserAgent) (outdated bool, e error)
	//get fresh user agents from the remote source.
	get() (agents []*types.UserAgent, e error)
	//load persistent data from local source.
	load() (agents []*types.UserAgent, e error)
}

func try(op func(int) error, maxRetry int, maxDelay time.Duration) error {
	return repeat.Repeat(
		repeat.FnWithCounter(op),
		repeat.StopOnSuccess(),
		repeat.LimitMaxTries(maxRetry),
		repeat.WithDelay(
			repeat.FullJitterBackoff(500*time.Millisecond).WithMaxDelay(maxDelay).Set(),
		),
	)
}
