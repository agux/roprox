package targets

import (
	"testing"

	"github.com/agux/roprox/internal/types"
)

func TestFetch_ProxyDB(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
		}
	}()
	gp := &ProxyDB{}
	for i, url := range gp.Urls() {
		FetchFor(i, url, chpx, gp)
	}
}
