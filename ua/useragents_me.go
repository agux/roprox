package ua

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/data"
	"github.com/agux/roprox/types"
	"github.com/agux/roprox/util"
	"github.com/pkg/errors"
	"github.com/ssgreg/repeat"
	"gorm.io/gorm/clause"
)

type userAgentsMe struct {
}

func (uam userAgentsMe) urlMatch(url string) (matched bool) {
	prefix := "https://www.useragents.me/"
	return strings.HasPrefix(strings.ToLower(url), prefix)
}

func (uam userAgentsMe) outdated(agents []*types.UserAgent) (outdated bool, e error) {
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

func (uam userAgentsMe) get() (agents []*types.UserAgent, e error) {
	url := conf.Args.DataSource.UserAgents

	jsonStr, e := uam.getJSON(url)
	if e != nil {
		log.Errorf("failed to get agent list from %s, giving up %+v", url, e)
		return
	}

	//parse user agents
	var uamJson userAgentsMeJson
	if e = json.Unmarshal([]byte(jsonStr), &uamJson); e != nil {
		e = errors.Wrapf(e, "unable to unmarshal json from text: %s", jsonStr)
		return
	}
	for _, u := range uamJson.Data {
		ua := &types.UserAgent{
			ID:        -1,
			Source:    sql.NullString{String: `useragents.me`, Valid: true},
			UserAgent: sql.NullString{String: u.Ua, Valid: true},
			TimesSeen: sql.NullInt64{Int64: -1, Valid: false},
			Percent:   sql.NullFloat64{Float64: u.Pct, Valid: true},
			UpdatedAt: sql.NullString{String: util.Now(), Valid: true},
		}
		agents = append(agents, ua)
	}

	e = uam.mergeAgents(agents)

	return
}

func (uam userAgentsMe) load() (agents []*types.UserAgent, e error) {
	op := func(c int) error {
		if e = data.GormDB.Where("source = ?", "useragents.me").Find(&agents).Error; e != nil {
			e = errors.Wrapf(e, "#%d failed to load user_agents sourced from 'useragents.me'", c+1)
			log.Warnln(e)
			return repeat.HintTemporary(e)
		}
		return e
	}
	e = try(op, conf.Args.Database.MaxRetry, 15*time.Second)
	return
}

func (uam userAgentsMe) assignID(agents []*types.UserAgent) (maxID int64, e error) {
	op := func(c int) error {
		var v sql.NullInt64
		if e = data.GormDB.Model(&types.UserAgent{}).Select("max(id)").Row().Scan(&v); e != nil {
			e = errors.Wrapf(e, "#%d failed to query max(ID) from table user_agents", c+1)
			log.Warnln(e)
			return repeat.HintTemporary(e)
		}
		if v.Valid {
			maxID = v.Int64
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
	exAgents, e = uam.load()
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

func (uam userAgentsMe) mergeAgents(agents []*types.UserAgent) (e error) {

	if _, e = uam.assignID(agents); e != nil {
		return
	}

	op := func(c int) error {
		e = data.GormDB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "ID"}}, // columns that have unique constraints
			DoUpdates: clause.AssignmentColumns([]string{"source", "user_agent", "times_seen", "percent", "simple_software_string", "software_name", "software_version", "software_type",
				"software_sub_type", "hardware_type", "first_seen_at", "last_seen_at", "updated_at"}), // columns that should be updated
		}).Create(&agents).Error

		if e != nil {
			e = errors.Wrapf(e, "#%d failed to merge %d user_agents sourced from 'useragents.me'", c+1, len(agents))
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

func (uam userAgentsMe) getJSON(url string) (jsonStr string, e error) {
	op := func(c int) error {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			e = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			return e
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			err = errors.Wrap(err, "failed to read response body")
			return err
		}

		jsonStr = string(bodyBytes)
		return nil
	}
	// use net/http package to get jsonStr via HTTP request to url
	e = try(op, conf.Args.Network.HTTPRetry, time.Duration(conf.Args.Network.HTTPTimeout)*time.Second)
	return
}

type userAgentsMeJson struct {
	About string `json:"about"`
	Terms string `json:"terms"`
	Data  []struct {
		Ua  string  `json:"ua"`
		Pct float64 `json:"pct"`
	}
}
