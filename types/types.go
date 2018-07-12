package types

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/carusyte/roprox/util"
)

const (
	//OK indicates the proxy server is available per the last check result
	OK = "OK"
	//Fail indicates the proxy server is unavailable per the last check result
	Fail = "Fail"
)

//ProxyServer is a model mapping for database table proxy_list
type ProxyServer struct {
	Source      string
	Host        string
	Port        string
	Type        string
	Status      string
	LastCheck   string `db:"last_check"`
	LastScanned string `db:"last_scanned"`
}

//NewProxyServer creates an instance of ProxyServer.
func NewProxyServer(source, host, port, stype string) *ProxyServer {
	return &ProxyServer{
		Source:      source,
		Host:        host,
		Port:        port,
		Type:        stype,
		Status:      OK,
		LastCheck:   util.Now(),
		LastScanned: util.Now(),
	}
}

//FetcherSpec defines detail specifications on fetching open proxy servers from the web.
type FetcherSpec interface {
	//UID returns the unique identifier for this spec.
	UID() string
	//Urls return the server urls that provide the free proxy server lists.
	Urls() []string
	//IsGBK returns wheter the web page is GBK encoded.
	IsGBK() bool
	//UseMasterProxy returns whether the fetcher needs a master proxy server
	//to access the free proxy list provider.
	UseMasterProxy() bool
	//ListSelector returns the jQuery selectors for searching the proxy server list/table.
	ListSelector() []string
	//RefreshInterval determines how often the list should be refreshed, in minutes.
	RefreshInterval() int
	//ScanItem process each item found in the table determined by ListSelector().
	ScanItem(i int, s *goquery.Selection) (ps *ProxyServer)
}
