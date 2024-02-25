package util

import (
	"net/url"

	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/logging"
	"github.com/agux/roprox/internal/types"
)

var log = logging.Logger

// construct a new *types.ProxyServer instance from the `conf.Args.Network.MasterProxyAddr` string.
func GetMasterProxy() *types.ProxyServer {
	// sample: `http://127.0.0.1:1087`, `socks5://127.0.0.1:1080`
	// parse the string and assign to corresponding ProxyServer struct attributes as follows:
	// {Host: "127.0.0.1", Port: "1087", Type: "http"}
	// {Host: "127.0.0.1", Port: "1080", Type: "socks5"}
	masterProxy := &types.ProxyServer{ID: 0, Source: "config"}
	u, err := url.Parse(conf.Args.Network.MasterProxyAddr)
	if err != nil {
		log.Errorf("Error parsing master proxy address: %v\n", err)
		return nil
	}
	masterProxy.Host = u.Hostname()
	masterProxy.Port = u.Port()
	switch u.Scheme {
	case "http", "https":
		masterProxy.Type = u.Scheme
	case "socks5":
		masterProxy.Type = "socks5"
	default:
		log.Errorf("Unsupported proxy scheme: %s\n", u.Scheme)
		return nil
	}
	return masterProxy
}
