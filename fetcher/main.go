package fetcher

import "github.com/carusyte/roprox/logging"

var log = logging.Logger

type defaultDynamicHTMLFetcher struct {
}

//HomePageTimeout specifies how many seconds to wait before home page navigation is timed out
func (f defaultDynamicHTMLFetcher) HomePageTimeout() int {
	return 20
}

type defaultFetcherSpec struct {
}

func (f defaultFetcherSpec) Retry() int {
	return 0
}
