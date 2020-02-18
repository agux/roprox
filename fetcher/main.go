package fetcher

import "github.com/carusyte/roprox/logging"

var log = logging.Logger

type defaultFetcherSpec struct{
}

func (f defaultFetcherSpec) Retry() int {
	return 0
}