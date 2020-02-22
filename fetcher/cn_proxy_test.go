package fetcher

import (
	"testing"

	"github.com/agux/roprox/types"
)

func TestCNProxy_GetProxy(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
		}
	}()
	gp := &CNProxy{}
	for i, url := range gp.Urls() {
		fetchFor(i, url, chpx, gp)
	}
}

func TestCNProxy_Archive(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
		}
	}()
	gp := &CNProxy{}
	fetchFor(2, `http://cn-proxy.com/archives/218`, chpx, gp)
}
