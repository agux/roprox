package network

import (
	"testing"

	"github.com/agux/roprox/internal/types"
)

func TestHttpbin(t *testing.T) {
	var ps *types.ProxyServer
	var e error
	var ip, trueIp string
	for i := 0; i < 10; i++ {
		if ps, e = PickProxy(); e != nil {
			t.Error(e)
		}
		if ip, e = httpbin(ps); e != nil {
			continue
		} else {
			if trueIp, e = httpbin(nil); e != nil {
				t.Errorf("httpbin() failed to get true IP: %v", e)
			} else if ip == trueIp {
				t.Logf("httpbin() got = %v, which is equal to true IP %s", ip, trueIp)
				break
			}
			// mark test as success and exit.
			t.Logf("Proxy IP %v is different from true IP %v", ip, trueIp)
			t.Skip("Test passed with different proxy and true IP")
		}
	}
	// if e != nil {
	// 	t.Errorf("httpbin() failed: %+v", e)
	// }
}

func TestAdhoc(t *testing.T) {
	url := "https://ipinfo.io/ip"
	ps := types.NewProxyServer("test", "98.162.25.7", "31653", "socks5", "")
	if ip, e := rebounceIpAsText(url, ps); e != nil {
		t.Error(e)
	} else {
		t.Logf("your IP: %s", ip)
	}
}

func TestRebounceIpAsText(t *testing.T) {
	urls := []string{"https://icanhazip.com/", "https://api.ipify.org", "https://ifconfig.me/ip", "ipinfo.io/ip", "https://api.seeip.org", "http://myexternalip.com/raw"}
	okList := make([]string, 0, 8)
	failList := make([]string, 0, 8)
	failProxy := make(map[string]string)
nextURL:
	for _, url := range urls {
		t.Logf("Testing rebouncer %s", url)
		var ps *types.ProxyServer
		var e error
		var ip, trueIp string
		maxRetry := 10
		if trueIp, e = rebounceIpAsText(url, nil); e != nil {
			t.Errorf("rebounceIpAsText() failed to get true IP: %v", e)
			failList = append(failList, url)
			continue
		}
		for i := 1; i < maxRetry; i++ {
			if ps, e = PickProxy(); e != nil {
				t.Error(e)
			}
			if ip, e = rebounceIpAsText(url, ps); e != nil {
				t.Errorf("proxy %s failed: %+v", ps.UrlString(), e)
				failProxy[ps.UrlString()] = e.Error()
				if i+1 == maxRetry {
					failList = append(failList, url)
				}
				continue
			} else {
				if ip == trueIp {
					t.Logf("WARNING rebounceIpAsText() got = %v using proxy %s, which is equal to true IP %s", ip, ps.UrlString(), trueIp)
					break
				}
				// mark test as success and exit.
				t.Logf("Good news for %s: Proxy IP %v is different from true IP %v", url, ip, trueIp)
				okList = append(okList, url)
				continue nextURL
			}
		}
	}
	t.Logf("these rebouncer failed the test: %+v", failList)
	t.Logf("failed proxies: %+v", failProxy)
	t.Skipf("these rebouncer passed the test: %+v", okList)
}
