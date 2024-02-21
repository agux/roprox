package ua

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/data"
	"github.com/agux/roprox/internal/types"
	"github.com/agux/roprox/internal/util"
	"github.com/agux/roprox/pkg/browser"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"github.com/ssgreg/repeat"
)

type willsHouse struct {
}

func (w willsHouse) urlMatch(url string) (matched bool) {
	prefix := "https://techblog.willshouse.com/"
	return strings.HasPrefix(strings.ToLower(url), prefix)
}

func (w willsHouse) outdated(agents []*types.UserAgent) (outdated bool, e error) {
	var latest time.Time
	if len(agents) == 0 || !agents[0].UpdatedAt.Valid {
		outdated = true
		return
	}
	latest, e = time.Parse(dateTimeFormat, agents[0].UpdatedAt.String)
	if e != nil {
		return
	}
	if time.Since(latest).Hours() >=
		float64(time.Duration(conf.Args.DataSource.UserAgentLifespan*24)*time.Hour) {
		outdated = true
	}
	return
}

func (w willsHouse) get() (agents []*types.UserAgent, e error) {
	url := conf.Args.DataSource.UserAgents

	jsonStr, e := w.getJSON(url)
	if e != nil {
		log.Errorf("failed to get agent list from %s, giving up %+v", url, e)
		return
	}

	//parse user agents
	var uaJson []willshouseUserAgent
	if e = json.Unmarshal([]byte(jsonStr), &uaJson); e != nil {
		e = errors.Wrapf(e, "unable to unmarshal json from text: %s", jsonStr)
		return
	}
	for _, u := range uaJson {
		percent := .0
		if percent, e = strconv.ParseFloat(strings.Trim(u.Percent, "%"), 64); e != nil {
			log.Errorf("failed to convert percent value for user agent: %+v", u)
		}
		ua := &types.UserAgent{
			ID:                   -1,
			Source:               sql.NullString{String: `willshouse`, Valid: true},
			UserAgent:            sql.NullString{String: u.UserAgent, Valid: true},
			SimpleSoftwareString: sql.NullString{String: u.System, Valid: true},
			Percent:              sql.NullFloat64{Float64: percent, Valid: true},
			UpdatedAt:            sql.NullString{String: util.Now(), Valid: true},
		}
		agents = append(agents, ua)
	}
	w.mergeAgents(agents)
	return
}

func (w willsHouse) load() (agents []*types.UserAgent, e error) {
	op := func(c int) error {
		if _, e = data.DB.Select(&agents, "select * from user_agents where source = ? order by updated_at desc", "willshouse"); e != nil {
			e = errors.Wrapf(e, "#%d failed to load user_agents sourced from 'willshouse'", c+1)
			log.Warnln(e)
			return repeat.HintTemporary(e)
		}
		return nil
	}
	e = try(op, conf.Args.Database.MaxRetry, 15*time.Second)
	return
}

func (w willsHouse) assignID(agents []*types.UserAgent) (maxID int64, e error) {
	op := func(c int) error {
		maxID, e = data.DB.SelectInt(`select max(id) from user_agents`)
		if e != nil {
			e = errors.Wrapf(e, "#%d failed to query max(ID) from table user_agents", c+1)
			log.Warnln(e)
			return repeat.HintTemporary(e)
		}
		return nil
	}
	if e = try(op, conf.Args.Database.MaxRetry, 15*time.Second); e != nil {
		return
	}
	for _, a := range agents {
		if a.ID > int(maxID) {
			maxID = int64(a.ID)
		}
	}
	var exAgents []*types.UserAgent
	exAgents, e = w.load()
	if len(exAgents) > 0 {
		var exAgentStrings []string
		ua2id := make(map[string]int)
		for _, a := range exAgents {
			if a.UserAgent.Valid {
				exAgentStrings = append(exAgentStrings, a.UserAgent.String)
				ua2id[a.UserAgent.String] = a.ID
			}
		}
		sort.Strings(exAgentStrings)
		for _, a := range agents {
			if a.UserAgent.Valid && a.ID < 0 {
				if sort.SearchStrings(exAgentStrings, a.UserAgent.String) < len(exAgentStrings) {
					a.ID = ua2id[a.UserAgent.String]
				}
			}
		}
	}
	for _, a := range agents {
		if a.UserAgent.Valid && a.ID < 0 {
			maxID++
			a.ID = int(maxID)
		}
	}
	return
}

func (w willsHouse) mergeAgents(agents []*types.UserAgent) (e error) {

	_, e = w.assignID(agents)

	fields := []string{
		"id", "source", "user_agent", "times_seen", "percent", "simple_software_string", "software_name", "software_version", "software_type",
		"software_sub_type", "hardware_type", "first_seen_at", "last_seen_at", "updated_at",
	}
	numFields := len(fields)
	holders := make([]string, numFields)
	for i := range holders {
		holders[i] = "?"
	}
	holderString := fmt.Sprintf("(%s)", strings.Join(holders, ","))
	valueStrings := make([]string, 0, len(agents))
	valueArgs := make([]interface{}, 0, len(agents)*numFields)
	for _, a := range agents {
		valueStrings = append(valueStrings, holderString)
		valueArgs = append(valueArgs, a.ID)
		valueArgs = append(valueArgs, a.Source)
		valueArgs = append(valueArgs, a.UserAgent)
		valueArgs = append(valueArgs, a.TimesSeen)
		valueArgs = append(valueArgs, a.Percent)
		valueArgs = append(valueArgs, a.SimpleSoftwareString)
		valueArgs = append(valueArgs, a.SoftwareName)
		valueArgs = append(valueArgs, a.SoftwareVersion)
		valueArgs = append(valueArgs, a.SoftwareType)
		valueArgs = append(valueArgs, a.SoftwareSubType)
		valueArgs = append(valueArgs, a.HardwareType)
		valueArgs = append(valueArgs, a.FirstSeenAt)
		valueArgs = append(valueArgs, a.LastSeenAt)
		valueArgs = append(valueArgs, a.UpdatedAt)
	}

	var updFieldStr []string
	for _, f := range fields {
		if f == "id" {
			continue
		}
		updFieldStr = append(updFieldStr, fmt.Sprintf("%[1]s=values(%[1]s)", f))
	}

	stmt := fmt.Sprintf("INSERT INTO user_agents (%s) VALUES %s on duplicate key update %s",
		strings.Join(fields, ","), strings.Join(valueStrings, ","), strings.Join(updFieldStr, ","))

	op := func(c int) error {
		if _, e = data.DB.Exec(stmt, valueArgs...); e != nil {
			e = errors.Wrapf(e, "#%d failed to merge %d user_agents sourced from 'willshouse'", c+1, len(agents))
			log.Warnln(e)
			return repeat.HintTemporary(e)
		}
		return nil
	}
	if e = try(op, conf.Args.Database.MaxRetry, 15*time.Second); e != nil {
		log.Panicln(e)
	}
	return
}

func (w willsHouse) getJSON(url string) (jsonStr string, e error) {
	// Get the data
	op := func(c int) error {
		var chrome *browser.Chrome
		ts := time.Now()
		timeout := time.Duration(conf.Args.DataSource.UserAgentsTimeout) * time.Second
		deadline := ts.Add(timeout)
		if chrome, e = browser.LaunchChrome(url, "", nil, timeout); e != nil {
			chrome.Cancel()
			return repeat.HintTemporary(e)
		}
		timeout = time.Until(deadline) //calculate remaining time
		sel := `#post-2229 > div.entry-content > textarea:nth-child(10)`
		if e = chrome.Run(
			timeout,
			chromedp.WaitReady(sel),
			// chromedp.WaitVisible(sel),
			// chromedp.ScrollIntoView(sel),
			chromedp.Value(sel, &jsonStr),
		); e != nil {
			chrome.Cancel()
			e = errors.Wrapf(e, "failed to get text area value using selector [%s], in url [%s]", sel, url)
			log.Errorln(e)
			return repeat.HintTemporary(e)
		}

		chrome.Cancel()
		return nil
	}
	e = try(op, conf.Args.WebDriver.MaxRetry, 10*time.Second)
	return
}

type willshouseUserAgent struct {
	Percent   string `json:"percent"`
	UserAgent string `json:"useragent"`
	System    string `json:"system"`
}
