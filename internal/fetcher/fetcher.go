package fetcher

import (
	"github.com/agux/roprox/internal/fetcher/targets"
	t "github.com/agux/roprox/internal/types"
	//shorten type reference
)

// Fetch proxy server information using the specified fetcher specification,
// and output to the channel.
func Fetch(chpx chan<- *t.ProxyServer, fspec t.FetcherSpec) {
	urls := fspec.Urls()
	for i, url := range urls {
		targets.FetchFor(i, url, chpx, fspec)
	}
}
