package targets

import (
	"testing"

	"github.com/agux/roprox/internal/types"
)

func TestFetch_IP3366(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	suc := false
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
			suc = true
		}
	}()
	gp := &IP3366{}
	for i, url := range gp.Urls() {
		FetchFor(i, url, chpx, gp)
	}
	if !suc {
		t.Fail()
	}
}
