package targets

import (
	"testing"

	"github.com/agux/roprox/internal/types"
)

func TestFetchGatherProxy(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Tracef("extracted proxy: %+v", px)
		}
	}()
	gp := &GatherProxy{}
	for i, url := range gp.Urls() {
		FetchFor(i, url, chpx, gp)
	}
}
