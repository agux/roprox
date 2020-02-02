package fetcher

import (
	"testing"

	"github.com/carusyte/roprox/types"
	"github.com/spf13/viper"
)

func TestFetch_SSLProxies(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			log.Debugf("extracted proxy: %+v", px)
		}
	}()
	gp := &SSLProxies{}
	for i, url := range gp.Urls() {
		fetchFor(i, url, chpx, gp)
	}

	log.Infof("config file used: %s", viper.ConfigFileUsed())
}
