package fetcher

import (
	"testing"

	"github.com/carusyte/roprox/types"
	"github.com/sirupsen/logrus"
)

func TestFetchGatherProxy(t *testing.T) {
	t.Fail()
	chpx := make(chan *types.ProxyServer, 100)
	go func() {
		for px := range chpx {
			logrus.Error(px)
		}
	}()
	fetchFor(0, `http://www.gatherproxy.com/proxylist/anonymity/?t=Elite`,
		chpx, GatherProxy{})
}
