package targets

import (
	"regexp"
	"strings"

	"github.com/agux/roprox/internal/types"
)

// SpysMe fetches proxy info from http://spys.me/
type SpysMe struct {
	defaultFetcherSpec
}

// UID returns the unique identifier for this spec.
func (p SpysMe) UID() string {
	return "SpysMe"
}

// Urls return the server urls that provide the free proxy server lists.
func (p SpysMe) Urls() []string {
	return []string{
		`http://spys.me/proxy.txt`,
	}
}

// ParsePlainText parses plain text payload and extracts proxy information
func (p SpysMe) ParsePlainText(payload []byte) (ps []*types.ProxyServer) {
	str := string(payload)
	log.Debugf("%s returned data:\n%s", p.UID(), str)
	// ptn := `^(?P<IP>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}):(?P<Port>\d+)\s(?P<CountryCode>[^-]+)-(?P<Anonymity>[^-+]+)`
	ptn := `(?P<IP>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}):(?P<Port>\d+)\s(?P<CountryCode>[^-]+)-(?P<Anonymity>[^-+]+)`
	// ptn = regexp.QuoteMeta(ptn)
	exp := regexp.MustCompile(ptn)
	matches := exp.FindAllStringSubmatch(str, -1)
	log.Debugf("%s found matched regex: %+v", p.UID(), matches)
	for _, m := range matches {
		r := make(map[string]string)
		for i, name := range exp.SubexpNames() {
			if i != 0 && name != "" {
				r[name] = strings.TrimSpace(m[i])
			}
		}
		// drop non-anonymous proxy
		if t, ok := r["Anonymity"]; !ok {
			log.Warnf("%s, unable to extract 'Anonymity': %+v", p.UID(), r)
			continue
		} else {
			t = strings.TrimSpace(t)
			if strings.EqualFold("N", t) && strings.EqualFold("N!", t) {
				continue
			}
		}
		ps = append(ps, types.NewProxyServer(p.UID(), r["IP"], r["Port"], "http", r["CountryCode"]))
	}
	return
}

// ProxyMode returns whether the fetcher needs a master proxy server
// to access the free proxy list provider.
func (p SpysMe) ProxyMode() types.ProxyMode {
	return types.Direct
}

// RefreshInterval determines how often the list should be refreshed, in minutes.
func (p SpysMe) RefreshInterval() int {
	return 30
}
