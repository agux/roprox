package ua

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/data"
	"github.com/agux/roprox/types"
	"github.com/ssgreg/repeat"
)

type whatIsMyBrowser struct {
}

func (w whatIsMyBrowser) urlMatch(url string) (matched bool) {
	prefix := "https://developers.whatismybrowser.com/"
	return strings.HasPrefix(strings.ToLower(url), prefix)
}

func (w whatIsMyBrowser) outdated(agents []*types.UserAgent) (outdated bool, e error) {
	if len(agents) == 0 || !agents[0].UpdatedAt.Valid {
		outdated = true
		return
	}
	var latest time.Time
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

func (w whatIsMyBrowser) get() (agents []*types.UserAgent, err error) {
	exePath, e := os.Executable()
	if e != nil {
		log.Panicln("failed to get executable path", e)
	}
	path, e := filepath.EvalSymlinks(exePath)
	if e != nil {
		log.Panicln("failed to evaluate symlinks, ", exePath, e)
	}
	local := filepath.Join(filepath.Dir(path), filepath.Base(conf.Args.DataSource.UserAgents))
	if _, e := os.Stat(local); e == nil {
		os.Remove(local)
	}
	e = w.downloadFile(local, conf.Args.DataSource.UserAgents)
	defer os.Remove(local)
	if e != nil {
		log.Panicln("failed to download user agent sample file ", conf.Args.DataSource.UserAgents, e)
	}
	agents, e = w.readCSV(local)
	if e != nil {
		log.Panicln("failed to download and read csv, ", local, e)
	}
	w.mergeAgents(agents)
	return
}

func (w whatIsMyBrowser) load() (agents []*types.UserAgent, e error) {
	_, e = data.DB.Select(&agents, "select * from user_agents where hardware_type = ? order by updated_at desc", "computer")
	if e != nil {
		if e.Error() != "sql: no rows in result set" {
			log.Panicln("failed to run sql", e)
		}
	}
	return
}

func (w whatIsMyBrowser) mergeAgents(agents []*types.UserAgent) (e error) {
	fields := []string{
		"id", "user_agent", "times_seen", "simple_software_string", "software_name", "software_version", "software_type",
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
		valueArgs = append(valueArgs, a.UserAgent)
		valueArgs = append(valueArgs, a.TimesSeen)
		valueArgs = append(valueArgs, a.SimpleSoftwareString)
		valueArgs = append(valueArgs, a.SoftwareName)
		valueArgs = append(valueArgs, a.SoftwareVersion)
		valueArgs = append(valueArgs, a.SoftwareType)
		valueArgs = append(valueArgs, a.SoftwareSubType)
		valueArgs = append(valueArgs, a.HardWareType)
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

	retry := 5
	rt := 0
	stmt := fmt.Sprintf("INSERT INTO user_agents (%s) VALUES %s on duplicate key update %s",
		strings.Join(fields, ","), strings.Join(valueStrings, ","), strings.Join(updFieldStr, ","))
	for ; rt < retry; rt++ {
		_, e = data.DB.Exec(stmt, valueArgs...)
		if e != nil {
			log.Warn(e)
			if strings.Contains(e.Error(), "Deadlock") {
				continue
			} else {
				log.Panicln("failed to merge user_agent", e)
			}
		}
		return nil
	}
	log.Panicln("failed to merge user_agent", e)
	return
}

func (w whatIsMyBrowser) readCSV(src string) (agents []*types.UserAgent, err error) {
	f, err := os.Open(src)
	if err != nil {
		return
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return
	}

	tarReader := tar.NewReader(gzf)

	for {
		var header *tar.Header
		header, err = tarReader.Next()
		if err == io.EOF {
			return
		} else if err != nil {
			return
		}

		name := header.Name

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			if !strings.EqualFold(".csv", filepath.Ext(name)) {
				continue
			}
		default:
			continue
		}

		csvReader := csv.NewReader(tarReader)
		var lines [][]string
		lines, err = csvReader.ReadAll()
		if err != nil {
			return
		}

		for i, ln := range lines {
			if i == 0 {
				//skip header line
				continue
			}
			var id, times_seen int64
			if id, err = strconv.ParseInt(ln[0], 10, 64); err != nil {
				return
			}
			if times_seen, err = strconv.ParseInt(ln[2], 10, 64); err != nil {
				return
			}
			agents = append(agents, &types.UserAgent{
				ID:                   int(id),
				UserAgent:            sql.NullString{String: ln[1], Valid: true},
				TimesSeen:            sql.NullInt64{Int64: times_seen, Valid: true},
				SimpleSoftwareString: sql.NullString{String: ln[3], Valid: true},
				SoftwareName:         sql.NullString{String: ln[7], Valid: true},
				SoftwareVersion:      sql.NullString{String: ln[10], Valid: true},
				SoftwareType:         sql.NullString{String: ln[22], Valid: true},
				SoftwareSubType:      sql.NullString{String: ln[23], Valid: true},
				HardWareType:         sql.NullString{String: ln[25], Valid: true},
				FirstSeenAt:          sql.NullString{String: ln[35], Valid: true},
				LastSeenAt:           sql.NullString{String: ln[36], Valid: true},
				UpdatedAt:            sql.NullString{String: time.Now().Format(dateTimeFormat), Valid: true},
			})
		}
		break
	}
	return
}

func (w whatIsMyBrowser) downloadFile(filepath string, url string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	var resp *http.Response
	op := func(c int) error {
		resp, err = http.Get(url)
		return err
	}
	err = repeat.Repeat(
		repeat.FnWithCounter(op),
		repeat.StopOnSuccess(),
		repeat.LimitMaxTries(conf.Args.Network.HTTPRetry),
		repeat.WithDelay(
			repeat.FullJitterBackoff(500*time.Millisecond).WithMaxDelay(10*time.Second).Set(),
		),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
