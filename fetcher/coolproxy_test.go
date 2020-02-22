package fetcher

import (
	"testing"

	"github.com/agux/roprox/types"
)

func TestFetchCoolProxy(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("found proxy: %+v", px)
		}
	}()
	fetchFor(0, `https://cool-proxy.net/proxies.json`,
		chpx, CoolProxy{})
}
