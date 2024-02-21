package targets

import (
	"testing"

	"github.com/agux/roprox/internal/types"
)

func TestFetchCoolProxy(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	suc := false
	go func() {
		for px := range chpx {
			log.Debugf("found proxy: %+v", px)
			suc = true
		}
	}()
	FetchFor(0, `https://cool-proxy.net/proxies.json`,
		chpx, CoolProxy{})
	if !suc {
		t.Fail()
	}
}
