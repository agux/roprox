package main

import (
	f "github.com/carusyte/roprox/fetcher"
	t "github.com/carusyte/roprox/types"
)

var fslist = []t.FetcherSpec{
	f.FreeProxyList{},
	f.KuaiDaiLi{},
	f.Data5u{},
	f.HinkyDink{},
	f.IP3366{},
	// f.SocksProxy{},
	// f.Z66IP{},
	f.SSLProxies{},
	f.CoderBusy{},
	// f.ProxyDB{},
	f.CoolProxy{},
	f.GouBanJia{},
	f.CNProxy{},
	f.FreeProxyCZ{},
	// f.GatherProxy{},
	f.ProxyListOrg{},
	f.ProxyListPlus{},
	f.Xroxy{},
	f.ProxyFish{},
}

var proxies = make(map[string]t.FetcherSpec)

func init() {
	for _, f := range fslist {
		proxies[f.UID()] = f
	}
}
