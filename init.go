package main

import "github.com/agux/roprox/types"

var (
	userAgentBinding  map[string]string
	proxyServersCache []types.ProxyServer
	cacheLastUpdated  int64
)

func init() {
	userAgentBinding = make(map[string]string)
}
