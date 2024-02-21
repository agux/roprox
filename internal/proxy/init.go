package proxy

import "github.com/agux/roprox/internal/types"

var (
	userAgentBinding  map[string]string
	proxyServersCache []types.ProxyServer
	cacheLastUpdated  int64
)

func init() {
	userAgentBinding = make(map[string]string)
}
