package main

import (
	f "github.com/agux/roprox/fetcher"
	t "github.com/agux/roprox/types"
)

var fslist = []t.FetcherSpec{
	f.FreeProxyList{},
	f.KuaiDaiLi{},
	f.HinkyDink{},
	f.IP3366{},
	f.SSLProxies{},
	f.CoolProxy{},
	f.CNProxy{},
	f.ProxyListOrg{},
	f.ProxyListPlus{},
	f.Xroxy{},
	f.SpysOne{},
	f.SpysMe{},
	f.HideMyName{},
	f.ProxyNova{},
	// deprecated proxies:
	// f.ProxyFish{},
	// f.FreeProxyCZ{},
	// f.GatherProxy{},
	// f.GouBanJia{},
	// f.CoderBusy{},
	// f.ProxyDB{},
	// f.SocksProxy{},
	// f.Z66IP{},
	// f.Data5u{},
}

var proxies = make(map[string]t.FetcherSpec)

func init() {
	for _, f := range fslist {
		proxies[f.UID()] = f
	}
}
