package fetcher

import (
	"testing"

	"github.com/agux/roprox/types"
)

func TestFetch_HinkyDink(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
		}
	}()
	gp := &HinkyDink{}
	for i, url := range gp.Urls() {
		fetchFor(i, url, chpx, gp)
	}
}
