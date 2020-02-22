package fetcher

import (
	"testing"

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/types"
)

func TestFetch_ProxyFish(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
		}
	}()
	gp := &ProxyFish{}
	for i, url := range gp.Urls() {
		fetchFor(i, url, chpx, gp)
	}

	log.Infof("config file used: %s", conf.ConfigFileUsed())
}
