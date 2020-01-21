package fetcher

import (
	"testing"

	"github.com/carusyte/roprox/types"
)

func TestFetchGouBanJia(t *testing.T) {
	chpx := make(chan *types.ProxyServer, 100)
	fetchFor(0, `http://www.goubanjia.com/`, chpx, GouBanJia{})
	for px := range chpx {
		log.Error(px)
	}
}
