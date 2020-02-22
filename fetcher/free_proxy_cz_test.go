package fetcher

import (
	"testing"

	"github.com/agux/roprox/types"
)

func TestFetch_FreeProxyCZ(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("found proxy: %+v", px)
		}
	}()
	p := &FreeProxyCZ{}
	for i, url := range p.Urls() {
		fetchFor(i, url, chpx, p)
	}
}
