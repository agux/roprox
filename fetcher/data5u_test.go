package fetcher

import (
	"testing"

	"github.com/agux/roprox/types"
)

func TestFetchData5u(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	suc := false
	go func() {
		for px := range chpx {
			log.Debugf("found proxy: %+v", px)
			suc = true
		}
	}()
	p := &Data5u{}
	for i, url := range p.Urls() {
		fetchFor(i, url, chpx, p)
	}
	if !suc {
		t.Fail()
	}
}
