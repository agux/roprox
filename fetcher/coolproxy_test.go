package fetcher

import (
	"testing"

	"github.com/carusyte/roprox/types"
)

func TestFetchCoolProxy(t *testing.T) {
	t.Fail()
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Error(px)
		}
	}()
	fetchFor(0, `https://www.cool-proxy.net/proxies/http_proxy_list/country_code:/port:/anonymous:1/page:1`,
		chpx, CoolProxy{})
}
