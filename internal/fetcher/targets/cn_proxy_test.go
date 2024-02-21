package targets

import (
	"testing"

	"github.com/agux/roprox/internal/types"
)

func TestCNProxy_GetProxy(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	suc := false
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
			suc = true
		}
	}()
	gp := &CNProxy{}
	for i, url := range gp.Urls() {
		FetchFor(i, url, chpx, gp)
	}
	if !suc {
		t.Fail()
	}
}
