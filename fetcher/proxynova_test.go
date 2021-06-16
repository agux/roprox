package fetcher

import (
	"testing"

	"github.com/agux/roprox/types"
)

func TestProxyNova_GetProxy(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	suc := false
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
			suc = true
		}
	}()
	gp := &ProxyNova{}
	for i, url := range gp.Urls() {
		fetchFor(i, url, chpx, gp)
	}
	if !suc {
		t.Fail()
	}
}
