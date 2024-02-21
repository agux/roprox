package targets

import (
	"testing"

	"github.com/agux/roprox/internal/types"
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
		FetchFor(i, url, chpx, p)
	}
}
