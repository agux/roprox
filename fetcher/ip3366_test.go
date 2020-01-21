package fetcher

import (
	"testing"

	"github.com/carusyte/roprox/types"
)

func TestFetch_IP3366(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
		}
	}()
	gp := &IP3366{}
	for i, url := range gp.Urls() {
		fetchFor(i, url, chpx, gp)
	}
}
