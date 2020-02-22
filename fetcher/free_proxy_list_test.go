package fetcher

import (
	"testing"

	"github.com/agux/roprox/types"
)

func TestFetch_FreeProxyList(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("found proxy: %+v", px)
		}
	}()
	p := &FreeProxyList{}
	for i, url := range p.Urls() {
		fetchFor(i, url, chpx, p)
	}
}
