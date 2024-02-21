package proxy

import (
	"crypto/tls"

	"github.com/agux/roprox/internal/types"
)

var (
	userAgentBinding  map[string]string
	proxyServersCache []types.ProxyServer
	cacheLastUpdated  int64
	certStore         map[string]tls.Certificate
)

func init() {
	userAgentBinding = make(map[string]string)
	certStore = make(map[string]tls.Certificate)
}
