package checker

import (
	"strings"
	"sync"
	"testing"

	"github.com/agux/roprox/internal/types"
)

func Test_main(t *testing.T) {

}

func Test_Check(t *testing.T) {
	Check(&sync.WaitGroup{})
}

func Test_ProbeLocal(t *testing.T) {
	ch := make(chan *types.ProxyServer, 16)
	var wg sync.WaitGroup
	wg.Add(1)
	ch <- types.NewProxyServer("GatherProxy", "47.94.220.11", "3128", "http", "")
	probeLocal(ch)
	wg.Wait()
}

func Test_Title(t *testing.T) {
	log.Debugf("Title: %s", strings.Title("potential non-compliance"))
}
