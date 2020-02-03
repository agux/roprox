package fetcher

import (
	"testing"

	"github.com/carusyte/roprox/conf"
	"github.com/carusyte/roprox/types"
)

func TestFetch_Z66IP(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
		}
	}()
	gp := &Z66IP{}
	for i, url := range gp.Urls() {
		fetchFor(i, url, chpx, gp)
	}

	log.Infof("config file used: %s", conf.ConfigFileUsed())
}
