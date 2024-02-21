package targets

import (
	"testing"

	"github.com/agux/roprox/internal/types"
)

func TestFetch_ProxyListPlus(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
		}
	}()
	gp := &ProxyListPlus{}
	for i, url := range gp.Urls() {
		FetchFor(i, url, chpx, gp)
	}
}
