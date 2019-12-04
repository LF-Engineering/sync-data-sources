package main

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	lib "github.com/LF-Engineering/sync-data-sources/sources"
	yaml "gopkg.in/yaml.v2"
)

var (
	sshKeyOnce   sync.Once
	randInitOnce sync.Once
)

func ensureGrimoireStackAvail(ctx *lib.Ctx) error {
	if ctx.Debug > 0 {
		lib.Printf("Checking grimoire stack availability\n")
	}
	dtStart := time.Now()
	ctx.ExecOutput = true
	info := ""
	defer func() {
		ctx.ExecOutput = false
	}()
	res, err := lib.ExecCommand(
		ctx,
		[]string{
			"perceval",
			"--version",
		},
		nil,
	)
	dtEnd := time.Now()
	if err != nil {
		lib.Printf("Error for perceval (took %v): %+v\n", dtEnd.Sub(dtStart), err)
		fmt.Fprintf(os.Stderr, "%v: Error for perceval (took %v): %+v\n", dtEnd, dtEnd.Sub(dtStart), res)
		return err
	}
	info = "perceval: " + res
	res, err = lib.ExecCommand(
		ctx,
		[]string{
			"p2o.py",
			"--help",
		},
		nil,
	)
	dtEnd = time.Now()
	if err != nil {
		lib.Printf("Error for p2o.py (took %v): %+v\n", dtEnd.Sub(dtStart), err)
		fmt.Fprintf(os.Stderr, "%v: Error for p2o.py (took %v): %+v\n", dtEnd, dtEnd.Sub(dtStart), res)
		return err
	}
	res, err = lib.ExecCommand(
		ctx,
		[]string{
			"sortinghat",
			"--version",
		},
		nil,
	)
	dtEnd = time.Now()
	if err != nil {
		lib.Printf("Error for sortinghat (took %v): %+v\n", dtEnd.Sub(dtStart), err)
		fmt.Fprintf(os.Stderr, "%v: Error for sortinghat (took %v): %+v\n", dtEnd, dtEnd.Sub(dtStart), res)
		return err
	}
	info += "sortinghat: " + res
	if ctx.Debug > 0 {
		lib.Printf("Grimoire stack available\n%s\n", info)
	}
	return nil
}

func syncGrimoireStack(ctx *lib.Ctx) error {
	dtStart := time.Now()
	ctx.ExecOutput = true
	defer func() {
		ctx.ExecOutput = false
	}()
	res, err := lib.ExecCommand(
		ctx,
		[]string{
			"find",
			"data/",
			"-type",
			"f",
			"-iname",
			"*.y*ml",
		},
		nil,
	)
	dtEnd := time.Now()
	if err != nil {
		lib.Printf("Error finding fixtures (took %v): %+v\n", dtEnd.Sub(dtStart), err)
		fmt.Fprintf(os.Stderr, "%v: Error finding fixtures (took %v): %+v\n", dtEnd, dtEnd.Sub(dtStart), res)
		return err
	}
	fixtures := strings.Split(res, "\n")
	if ctx.Debug > 0 {
		lib.Printf("Fixtures to process: %+v\n", fixtures)
	}
	return processFixtureFiles(ctx, fixtures)
}

func validateConfig(ctx *lib.Ctx, fixture *lib.Fixture, dataSource *lib.DataSource, cfg *lib.Config) {
	if cfg.Name == "" {
		lib.Fatalf("Config %+v name in data source %+v in fixture %+v is empty or undefined\n", cfg, dataSource, fixture)
	}
	if cfg.Value == "" {
		lib.Fatalf("Config %+v value in data source %+v in fixture %+v is empty or undefined\n", cfg, dataSource, fixture)
	}
}

func validateEndpoint(ctx *lib.Ctx, fixture *lib.Fixture, dataSource *lib.DataSource, endpoint *lib.Endpoint) {
	if endpoint.Name == "" {
		lib.Fatalf("Endpoint %+v name in data source %+v in fixture %+v is empty or undefined\n", endpoint, dataSource, fixture)
	}
}

func validateDataSource(ctx *lib.Ctx, fixture *lib.Fixture, dataSource *lib.DataSource) {
	if dataSource.Slug == "" {
		lib.Fatalf("Data source %+v in fixture %+v has empty slug or no slug property, slug property must be non-empty\n", dataSource, fixture)
	}
	if ctx.Debug > 2 {
		lib.Printf("Config for %s/%s: %+v\n", fixture.Fn, dataSource.Slug, dataSource.Config)
	}
	for _, cfg := range dataSource.Config {
		validateConfig(ctx, fixture, dataSource, &cfg)
	}
	st := make(map[string]lib.Config)
	for _, cfg := range dataSource.Config {
		name := cfg.Name
		cfg2, ok := st[name]
		if ok {
			lib.Fatalf("Duplicate name %s in config: %+v and %+v, data source: %+v, fixture: %+v\n", name, cfg, cfg2, dataSource, fixture)
		}
		st[name] = cfg
	}
	for _, endpoint := range dataSource.Endpoints {
		validateEndpoint(ctx, fixture, dataSource, &endpoint)
	}
	ste := make(map[string]lib.Endpoint)
	for _, endpoint := range dataSource.Endpoints {
		name := endpoint.Name
		endpoint2, ok := ste[name]
		if ok {
			lib.Fatalf("Duplicate name %s in endpoints: %+v and %+v, data source: %+v, fixture: %+v\n", name, endpoint, endpoint2, dataSource, fixture)
		}
		ste[name] = endpoint
	}
}

func validateFixture(ctx *lib.Ctx, fixture *lib.Fixture, fixtureFile string) {
	if len(fixture.Native) == 0 {
		lib.Fatalf("Fixture file %s has no 'native' property which is required\n", fixtureFile)
	}
	slug, ok := fixture.Native["slug"]
	if !ok {
		lib.Fatalf("Fixture file %s 'native' property has no 'slug' property which is required\n", fixtureFile)
	}
	if slug == "" {
		lib.Fatalf("Fixture file %s 'native' property 'slug' is empty which is forbidden\n", fixtureFile)
	}
	if len(fixture.DataSources) == 0 {
		lib.Fatalf("Fixture file %s must have at least one data source defined in 'data_sources' key\n", fixtureFile)
	}
	fixture.Fn = fixtureFile
	fixture.Slug = slug
	for _, dataSource := range fixture.DataSources {
		validateDataSource(ctx, fixture, &dataSource)
	}
	st := make(map[string]lib.DataSource)
	for _, dataSource := range fixture.DataSources {
		slug := dataSource.Slug
		dataSource2, ok := st[slug]
		if ok {
			lib.Fatalf("Duplicate slug %s in data sources: %+v and %+v, fixture: %+v\n", slug, dataSource, dataSource2, fixture)
		}
		st[slug] = dataSource
	}
}

func processFixtureFile(ch chan lib.Fixture, ctx *lib.Ctx, fixtureFile string) (fixture lib.Fixture) {
	// Synchronize go routine
	defer func() {
		if ch != nil {
			ch <- fixture
		}
	}()
	if ctx.Debug > 0 {
		lib.Printf("Processing: %s\n", fixtureFile)
	}
	// Read defined projects
	data, err := ioutil.ReadFile(fixtureFile)
	if err != nil {
		lib.Printf("Error reading file: %s\n", fixtureFile)
	}
	lib.FatalOnError(err)
	err = yaml.Unmarshal(data, &fixture)
	if err != nil {
		lib.Printf("Error parsing YAML file: %s\n", fixtureFile)
	}
	lib.FatalOnError(err)
	if ctx.Debug > 0 {
		lib.Printf("Loaded %s fixture: %+v\n", fixtureFile, fixture)
	}
	if fixture.Disabled == true {
		return
	}
	validateFixture(ctx, &fixture, fixtureFile)
	return
}

func processFixtureFiles(ctx *lib.Ctx, fixtureFiles []string) error {
	// Get number of CPUs available
	thrN := lib.GetThreadsNum(ctx)
	fixtures := []lib.Fixture{}
	if thrN > 1 {
		if ctx.Debug > 0 {
			lib.Printf("Now processing %d fixture files using MT%d version\n", len(fixtureFiles), thrN)
		}
		ch := make(chan lib.Fixture)
		nThreads := 0
		for _, fixtureFile := range fixtureFiles {
			if fixtureFile == "" {
				continue
			}
			go processFixtureFile(ch, ctx, fixtureFile)
			nThreads++
			if nThreads == thrN {
				fixture := <-ch
				nThreads--
				if fixture.Disabled != true {
					fixtures = append(fixtures, fixture)
				}
			}
		}
		if ctx.Debug > 0 {
			lib.Printf("Final threads join\n")
		}
		for nThreads > 0 {
			fixture := <-ch
			nThreads--
			if fixture.Disabled != true {
				fixtures = append(fixtures, fixture)
			}
		}
	} else {
		if ctx.Debug > 0 {
			lib.Printf("Now processing %d fixture files using ST version\n", len(fixtureFiles))
		}
		for _, fixtureFile := range fixtureFiles {
			if fixtureFile == "" {
				continue
			}
			fixture := processFixtureFile(nil, ctx, fixtureFile)
			if fixture.Disabled != true {
				fixtures = append(fixtures, fixture)
			}
		}
	}
	if len(fixtures) == 0 {
		lib.Fatalf("No fixtures read, this is error, please define at least one\n")
	}
	if ctx.Debug > 0 {
		lib.Printf("Fixtures: %+v\n", fixtures)
	}
	// Then for all fixtures defined, all slugs must be unique - check this also
	st := make(map[string]lib.Fixture)
	for _, fixture := range fixtures {
		slug := fixture.Native["slug"]
		fixture2, ok := st[slug]
		if ok {
			lib.Fatalf("Duplicate slug %s in fixtures: %+v and %+v\n", slug, fixture, fixture2)
		}
		st[slug] = fixture
	}
	tasks := []lib.Task{}
	nodeIdx := ctx.NodeIdx
	nodeNum := ctx.NodeNum
	knownDsTypes := make(map[string]struct{})
	for _, fixture := range fixtures {
		for _, dataSource := range fixture.DataSources {
			knownDsTypes[dataSource.Slug] = struct{}{}
			for _, endpoint := range dataSource.Endpoints {
				if ctx.NodeHash {
					str := fixture.Slug + dataSource.Slug + endpoint.Name
					_, run := lib.Hash(str, nodeIdx, nodeNum)
					if !run {
						continue
					}
				}
				tasks = append(
					tasks,
					lib.Task{
						Endpoint: endpoint.Name,
						Config:   dataSource.Config,
						DsSlug:   dataSource.Slug,
						FxSlug:   fixture.Slug,
						FxFn:     fixture.Fn,
					},
				)
			}
		}
	}
	dss := []string{}
	for k := range knownDsTypes {
		dss = append(dss, k)
	}
	sort.Strings(dss)
	dssStr := strings.Join(dss, ", ")
	lib.Printf("%d Tasks, %d data source types: %+v\n", len(tasks), len(dss), dssStr)
	if ctx.Debug > 1 {
		lib.Printf("Tasks: %+v\n", tasks)
	}
	randInitOnce.Do(func() {
		rand.Seed(time.Now().UnixNano())
	})
	rand.Shuffle(len(tasks), func(i, j int) { tasks[i], tasks[j] = tasks[j], tasks[i] })
	ctx.ExecFatal = false
	ctx.ExecOutput = true
	ctx.ExecOutputStderr = true
	defer func() {
		ctx.ExecFatal = true
		ctx.ExecOutput = false
		ctx.ExecOutputStderr = false
	}()
	return processTasks(ctx, &tasks, dss)
}

func saveCSV(ctx *lib.Ctx, tasks []lib.Task) {
	var writer *csv.Writer
	csvFile := fmt.Sprintf("/root/.perceval/tasks_%d_%d.csv", ctx.NodeIdx, ctx.NodeNum)
	oFile, err := os.Create(csvFile)
	if err != nil {
		lib.Printf("CSV create error: %+v\n", err)
		return
	}
	defer func() { _ = oFile.Close() }()
	writer = csv.NewWriter(oFile)
	defer writer.Flush()
	hdr := []string{"project", "filename", "datasource", "endpoint", "config", "commandline", "duration", "seconds", "retries", "error"}
	err = writer.Write(hdr)
	if err != nil {
		lib.Printf("CSV write header error: %+v\n", err)
		return
	}
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].FxSlug == tasks[j].FxSlug {
			if tasks[i].DsSlug == tasks[j].DsSlug {
				return tasks[i].Endpoint < tasks[j].Endpoint
			}
			return tasks[i].DsSlug < tasks[j].DsSlug
		}
		return tasks[i].FxSlug < tasks[j].FxSlug
	})
	for _, task := range tasks {
		err = writer.Write(task.ToCSV())
		if err != nil {
			lib.Printf("CSV write row (%+v) error: %+v\n", task, err)
			return
		}
	}
}

func processTasks(ctx *lib.Ctx, ptasks *[]lib.Task, dss []string) error {
	tasks := *ptasks
	saveCSV(ctx, tasks)
	thrN := lib.GetThreadsNum(ctx)
	failed := [][2]int{}
	processed := 0
	all := len(tasks)
	byDs := make(map[string][3]int)
	byFx := make(map[string][3]int)
	for _, task := range tasks {
		dsSlug := task.DsSlug
		dataDs, ok := byDs[dsSlug]
		if ok {
			dataDs[0]++
			byDs[dsSlug] = dataDs
		} else {
			byDs[dsSlug] = [3]int{1, 0, 0}
		}
		fxSlug := task.FxSlug
		dataFx, ok := byFx[fxSlug]
		if ok {
			dataFx[0]++
			byFx[fxSlug] = dataFx
		} else {
			byFx[fxSlug] = [3]int{1, 0, 0}
		}
	}
	fxs := []string{}
	for k := range byFx {
		fxs = append(fxs, k)
	}
	sort.Strings(fxs)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGUSR1, syscall.SIGALRM)
	processing := make(map[int]struct{})
	startTimes := make(map[int]time.Time)
	endTimes := make(map[int]time.Time)
	durations := make(map[int]time.Duration)
	var mtx = &sync.RWMutex{}
	info := func() {
		mtx.RLock()
		if len(processing) > 0 {
			lib.Printf("Processing:\n")
			for idx := range processing {
				since := startTimes[idx]
				if ctx.Debug > 0 {
					lib.Printf("%+v: %+v, since: %+v\n", tasks[idx].ShortString(), time.Now().Sub(since), since)
				} else {
					lib.Printf("%+v: %+v\n", time.Now().Sub(since), tasks[idx].ShortString())
				}
			}
		}
		if len(durations) > 0 {
			lib.Printf("Longest running tasks (finished):\n")
			durs := make(map[time.Duration]int)
			dursAry := []time.Duration{}
			for idx, dur := range durations {
				durs[dur] = idx
				dursAry = append(dursAry, dur)
			}
			sort.Slice(dursAry, func(i, j int) bool {
				return dursAry[j] < dursAry[i]
			})
			n := ctx.NLongest
			nDur := len(dursAry)
			if nDur < n {
				n = nDur
			}
			for i, dur := range dursAry[0:n] {
				lib.Printf("#%d) %+v: %+v\n", i+1, dur, tasks[durs[dur]].ShortStringCmd(ctx))
			}
			if len(processing) > 0 {
				lib.Printf("Longest running tasks (in progress):\n")
				durs = make(map[time.Duration]int)
				dursAry = []time.Duration{}
				for idx := range processing {
					dur := time.Now().Sub(startTimes[idx])
					durs[dur] = idx
					dursAry = append(dursAry, dur)
				}
				sort.Slice(dursAry, func(i, j int) bool {
					return dursAry[j] < dursAry[i]
				})
				n := ctx.NLongest
				nDur := len(dursAry)
				if nDur < n {
					n = nDur
				}
				for i, dur := range dursAry[0:n] {
					lib.Printf("#%d) %+v: %+v\n", i+1, dur, tasks[durs[dur]].ShortString())
				}
			}
		}
		lib.Printf("Processed %d/%d (%.2f%%), failed: %d (%.2f%%)\n", processed, all, (float64(processed)*100.0)/float64(all), len(failed), (float64(len(failed))*100.0)/float64(all))
		strAry := []string{}
		for _, res := range failed {
			strAry = append(strAry, fmt.Sprintf("Failed: %+v: %s", tasks[res[0]], lib.ErrorStrings[res[1]]))
		}
		sort.Strings(strAry)
		for _, str := range strAry {
			lib.Printf("%s\n", str)
		}
		if len(failed) > 0 {
			lib.Printf("Processed %d/%d (%.2f%%), failed: %d (%.2f%%)\n", processed, all, (float64(processed)*100.0)/float64(all), len(failed), (float64(len(failed))*100.0)/float64(all))
		}
		out := false
		for _, ds := range dss {
			data := byDs[ds]
			allDs := data[0]
			failedDs := data[1]
			processedDs := data[2]
			if failedDs > 0 || processedDs != allDs {
				lib.Printf("Data source: %s, Processed %d/%d (%.2f%%), failed: %d (%.2f%%)\n", ds, processedDs, allDs, (float64(processedDs)*100.0)/float64(allDs), failedDs, (float64(failedDs)*100.0)/float64(allDs))
				out = true
			}
		}
		for _, fx := range fxs {
			data := byFx[fx]
			allFx := data[0]
			failedFx := data[1]
			processedFx := data[2]
			if failedFx > 0 || processedFx != allFx {
				lib.Printf("Fixture: %s, Processed %d/%d (%.2f%%), failed: %d (%.2f%%)\n", fx, processedFx, allFx, (float64(processedFx)*100.0)/float64(allFx), failedFx, (float64(failedFx)*100.0)/float64(allFx))
				out = true
			}
		}
		if out {
			lib.Printf("Processed %d/%d (%.2f%%), failed: %d (%.2f%%)\n", processed, all, (float64(processed)*100.0)/float64(all), len(failed), (float64(len(failed))*100.0)/float64(all))
		}
		saveCSV(ctx, tasks)
		mtx.RUnlock()
	}
	go func() {
		for {
			sig := <-sigs
			info()
			if sig == syscall.SIGINT {
				lib.Printf("Exiting due to SIGINT\n")
				os.Exit(1)
			} else if sig == syscall.SIGALRM {
				lib.Printf("Timeout after %d seconds\n", ctx.TimeoutSeconds)
				os.Exit(2)
			}
		}
	}()
	lastTime := time.Now()
	dtStart := lastTime
	if thrN > 1 {
		if ctx.Debug >= 0 {
			lib.Printf("Processing %d tasks using MT%d version\n", len(tasks), thrN)
		}
		ch := make(chan lib.TaskResult)
		nThreads := 0
		for idx, task := range tasks {
			mtx.Lock()
			processing[idx] = struct{}{}
			startTimes[idx] = time.Now()
			mtx.Unlock()
			go processTask(ch, ctx, idx, task)
			nThreads++
			if nThreads == thrN {
				result := <-ch
				res := result.Code
				tIdx := res[0]
				tasks[tIdx].CommandLine = result.CommandLine
				tasks[tIdx].Retries = result.Retries
				tasks[tIdx].Err = result.Err
				nThreads--
				ds := tasks[tIdx].DsSlug
				fx := tasks[tIdx].FxSlug
				mtx.Lock()
				delete(processing, tIdx)
				endTimes[tIdx] = time.Now()
				durations[tIdx] = endTimes[tIdx].Sub(startTimes[tIdx])
				tasks[tIdx].Duration = durations[tIdx]
				dataDs := byDs[ds]
				dataFx := byFx[fx]
				if res[1] != 0 {
					failed = append(failed, res)
					dataDs[1]++
					dataFx[1]++
				}
				dataDs[2]++
				dataFx[2]++
				byDs[ds] = dataDs
				byFx[fx] = dataFx
				processed++
				mtx.Unlock()
				lib.ProgressInfo(processed, all, dtStart, &lastTime, time.Duration(2)*time.Minute, tasks[tIdx].ShortString())
			}
		}
		if ctx.Debug > 0 {
			lib.Printf("Final threads join\n")
		}
		for nThreads > 0 {
			result := <-ch
			res := result.Code
			tIdx := res[0]
			tasks[tIdx].CommandLine = result.CommandLine
			tasks[tIdx].Retries = result.Retries
			tasks[tIdx].Err = result.Err
			nThreads--
			ds := tasks[tIdx].DsSlug
			fx := tasks[tIdx].FxSlug
			mtx.Lock()
			delete(processing, tIdx)
			endTimes[tIdx] = time.Now()
			durations[tIdx] = endTimes[tIdx].Sub(startTimes[tIdx])
			tasks[tIdx].Duration = durations[tIdx]
			dataDs := byDs[ds]
			dataFx := byFx[fx]
			if res[1] != 0 {
				failed = append(failed, res)
				dataDs[1]++
				dataFx[1]++
			}
			dataDs[2]++
			dataFx[2]++
			byDs[ds] = dataDs
			byFx[fx] = dataFx
			processed++
			mtx.Unlock()
			lib.ProgressInfo(processed, all, dtStart, &lastTime, time.Duration(2)*time.Minute, tasks[tIdx].ShortString())
		}
	} else {
		if ctx.Debug >= 0 {
			lib.Printf("Processing %d tasks using ST version\n", len(tasks))
		}
		for idx, task := range tasks {
			processing[idx] = struct{}{}
			result := processTask(nil, ctx, idx, task)
			res := result.Code
			tIdx := res[0]
			tasks[tIdx].CommandLine = result.CommandLine
			tasks[tIdx].Retries = result.Retries
			tasks[tIdx].Err = result.Err
			ds := tasks[tIdx].DsSlug
			fx := tasks[tIdx].FxSlug
			mtx.Lock()
			delete(processing, tIdx)
			endTimes[tIdx] = time.Now()
			durations[tIdx] = endTimes[tIdx].Sub(startTimes[tIdx])
			tasks[tIdx].Duration = durations[tIdx]
			dataDs := byDs[ds]
			dataFx := byFx[fx]
			if res[1] != 0 {
				failed = append(failed, res)
				dataDs[1]++
				dataFx[1]++
			}
			dataDs[2]++
			dataFx[2]++
			byDs[ds] = dataDs
			byFx[fx] = dataFx
			processed++
			mtx.Unlock()
			lib.ProgressInfo(processed, all, dtStart, &lastTime, time.Duration(2)*time.Minute, tasks[tIdx].ShortString())
		}
	}
	info()
	saveCSV(ctx, tasks)
	return nil
}

func addSSHPrivKey(ctx *lib.Ctx, key string) bool {
	if ctx.DryRun {
		return true
	}
	home := os.Getenv("HOME")
	dir := home + "/.ssh"
	cmd := exec.Command("mkdir", dir)
	_ = cmd.Run()
	fn := dir + "/id_rsa"
	err := ioutil.WriteFile(fn, []byte(key), 0600)
	if err != nil {
		lib.Printf("Error adding SSH Key %s: %+v\n", fn, err)
		return false
	}
	if ctx.Debug >= 0 {
		lib.Printf("Added SSH Key: %s\n", fn)
	}
	return true
}

// massageEndpoint - this function is used to make sure endpoint is correct for a given datasource
func massageEndpoint(endpoint string, ds string) (e []string) {
	defaults := map[string]struct{}{
		lib.Git:        {},
		lib.Confluence: {},
		lib.Gerrit:     {},
		lib.Jira:       {},
		lib.Slack:      {},
		lib.GroupsIO:   {},
		lib.Pipermail:  {},
		lib.Discourse:  {},
		lib.Jenkins:    {},
		lib.DockerHub:  {},
		lib.Bugzilla:   {},
		lib.MeetUp:     {},
	}
	if ds == lib.GitHub {
		if strings.Contains(endpoint, "/") {
			ary := strings.Split(endpoint, "/")
			nAry := []string{}
			for _, e := range ary {
				if e != "" {
					nAry = append(nAry, e)
				}
			}
			lAry := len(nAry)
			repo := nAry[lAry-1]
			if strings.HasSuffix(repo, ".git") {
				lRepo := len(repo)
				repo = repo[:lRepo-4]
			}
			e = append(e, nAry[lAry-2])
			e = append(e, repo)
		} else {
			ep := endpoint
			if strings.HasSuffix(ep, ".git") {
				lEp := len(ep)
				ep = ep[:lEp-4]
			}
			e = append(e, ep)
		}
	} else if ds == lib.DockerHub {
		if strings.Contains(endpoint, " ") {
			ary := strings.Split(endpoint, " ")
			nAry := []string{}
			for _, e := range ary {
				if e != "" {
					nAry = append(nAry, e)
				}
			}
			lAry := len(nAry)
			e = append(e, nAry[lAry-2])
			e = append(e, nAry[lAry-1])
		} else {
			e = append(e, endpoint)
		}
	} else {
		_, ok := defaults[ds]
		if ok {
			e = append(e, endpoint)
		}
	}
	return
}

// massageConfig - this function makes sure that given config options are valid for a given data source
// it also ensures some essential options are enabled and eventually reformats config
func massageConfig(ctx *lib.Ctx, config *[]lib.Config, ds string) (c []lib.MultiConfig, fail bool) {
	m := make(map[string]struct{})
	if ds == lib.GitHub {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			m[name] = struct{}{}
			if name == lib.APIToken {
				if strings.Contains(value, ",") {
					ary := strings.Split(value, ",")
					vals := []string{}
					for _, key := range ary {
						key = strings.Replace(key, "[", "", -1)
						key = strings.Replace(key, "]", "", -1)
						vals = append(vals, key)
					}
					c = append(c, lib.MultiConfig{Name: "-t", Value: vals})
				} else {
					c = append(c, lib.MultiConfig{Name: "-t", Value: []string{value}})
				}
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
			}
		}
		_, ok := m["sleep-for-rate"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "sleep-for-rate", Value: []string{}})
		}
		_, ok = m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}})
		}
	} else if ds == lib.Git {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			m[name] = struct{}{}
			c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
		}
		_, ok := m["latest-items"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "latest-items", Value: []string{}})
		}
	} else if ds == lib.Confluence {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			m[name] = struct{}{}
			c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}})
		}
	} else if ds == lib.Gerrit {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			if name == "ssh-key" {
				sshKeyOnce.Do(func() {
					fail = !addSSHPrivKey(ctx, value)
				})
				continue
			}
			m[name] = struct{}{}
			c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
		}
		_, ok := m["disable-host-key-check"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "disable-host-key-check", Value: []string{}})
		}
		_, ok = m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}})
		}
	} else if ds == lib.Jira {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			m[name] = struct{}{}
			if name == lib.BackendUser {
				c = append(c, lib.MultiConfig{Name: "-u", Value: []string{value}})
			} else if name == lib.BackendPassword {
				c = append(c, lib.MultiConfig{Name: "-p", Value: []string{value}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}})
		}
		_, ok = m["verify"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "verify", Value: []string{"False"}})
		}
	} else if ds == lib.Slack {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			m[name] = struct{}{}
			if name == lib.APIToken {
				c = append(c, lib.MultiConfig{Name: "-t", Value: []string{value}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}})
		}
	} else if ds == lib.GroupsIO {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			m[name] = struct{}{}
			if name == lib.Email {
				c = append(c, lib.MultiConfig{Name: "-e", Value: []string{value}})
			} else if name == lib.Password {
				c = append(c, lib.MultiConfig{Name: "-p", Value: []string{value}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
			}
		}
	} else if ds == lib.Pipermail {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			m[name] = struct{}{}
			c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
			_, ok := m["no-verify"]
			if !ok {
				c = append(c, lib.MultiConfig{Name: "no-verify", Value: []string{}})
			}
		}
	} else if ds == lib.Discourse {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			m[name] = struct{}{}
			if name == lib.APIToken {
				if strings.Contains(value, ",") {
					ary := strings.Split(value, ",")
					randInitOnce.Do(func() {
						rand.Seed(time.Now().UnixNano())
					})
					idx := rand.Intn(len(ary))
					c = append(c, lib.MultiConfig{Name: "-t", Value: []string{ary[idx]}})
				} else {
					c = append(c, lib.MultiConfig{Name: "-t", Value: []string{value}})
				}
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}})
		}
	} else if ds == lib.Jenkins {
		for _, cfg := range *config {
			name := cfg.Name
			if name == lib.FromDate {
				continue
			}
			value := cfg.Value
			m[name] = struct{}{}
			if name == lib.APIToken {
				c = append(c, lib.MultiConfig{Name: "-t", Value: []string{value}})
			} else if name == lib.BackendUser {
				c = append(c, lib.MultiConfig{Name: "-u", Value: []string{value}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}})
		}
	} else if ds == lib.DockerHub {
		for _, cfg := range *config {
			name := cfg.Name
			if name == lib.FromDate {
				continue
			}
			value := cfg.Value
			m[name] = struct{}{}
			c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
			_, ok := m["no-archive"]
			if !ok {
				c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}})
			}
		}
	} else if ds == lib.Bugzilla {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			m[name] = struct{}{}
			if name == lib.BackendUser {
				c = append(c, lib.MultiConfig{Name: "-u", Value: []string{value}})
			} else if name == lib.BackendPassword {
				c = append(c, lib.MultiConfig{Name: "-p", Value: []string{value}})
			} else if name == lib.APIToken {
				c = append(c, lib.MultiConfig{Name: "-t", Value: []string{value}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}})
		}
	} else if ds == lib.MeetUp {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			m[name] = struct{}{}
			if name == lib.APIToken {
				c = append(c, lib.MultiConfig{Name: "-t", Value: []string{value}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}})
		}
		_, ok = m["sleep-for-rate"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "sleep-for-rate", Value: []string{}})
		}
	} else {
		fail = true
	}
	return
}

func massageDataSource(ds string) string {
	if ds == "bugzilla" {
		return "bugzillarest"
	}
	return ds
}

func processTask(ch chan lib.TaskResult, ctx *lib.Ctx, idx int, task lib.Task) (result lib.TaskResult) {
	// Ensure to unlock thread when finishing
	defer func() {
		// Synchronize go routine
		if ch != nil {
			ch <- result
		}
	}()
	if ctx.Debug > 1 {
		lib.Printf("Processing: %s\n", task)
	}
	result.Code[0] = idx

	// Handle DS slug
	ds := task.DsSlug
	idxSlug := "sds-" + task.FxSlug + "-" + ds
	idxSlug = strings.Replace(idxSlug, "/", "-", -1)
	commandLine := []string{
		"p2o.py",
		"--enrich",
		"--index",
		idxSlug + "-raw",
		"--index-enrich",
		idxSlug,
		"-e",
		ctx.ElasticURL,
	}
	if !ctx.SkipSH {
		commandLine = append(
			commandLine,
			[]string{
				"--db-host",
				ctx.ShHost,
				"--db-sortinghat",
				ctx.ShDB,
				"--db-user",
				ctx.ShUser,
				"--db-password",
				ctx.ShPass,
			}...,
		)
	}
	if ctx.CmdDebug > 0 {
		commandLine = append(commandLine, "--debug")
	}
	if ctx.EsBulkSize > 0 {
		commandLine = append(commandLine, "--bulk-size")
		commandLine = append(commandLine, strconv.Itoa(ctx.EsBulkSize))
	}
	if strings.Contains(ds, "/") {
		ary := strings.Split(ds, "/")
		if len(ary) != 2 {
			lib.Printf("%s: %+v: %s\n", ds, task, lib.ErrorStrings[1])
			result.Code[1] = 1
			return
		}
		commandLine = append(commandLine, massageDataSource(ary[0]))
		commandLine = append(commandLine, "--category")
		commandLine = append(commandLine, ary[1])
		ds = ary[0]
	} else {
		commandLine = append(commandLine, massageDataSource(ds))
	}

	// Handle DS endpoint
	eps := massageEndpoint(task.Endpoint, ds)
	if len(eps) == 0 {
		lib.Printf("%s: %+v: %s\n", task.Endpoint, task, lib.ErrorStrings[2])
		result.Code[1] = 2
		return
	}
	for _, ep := range eps {
		commandLine = append(commandLine, ep)
	}

	// Handle DS config options
	multiConfig, fail := massageConfig(ctx, &(task.Config), ds)
	if fail == true {
		lib.Printf("%+v: %s\n", task, lib.ErrorStrings[3])
		result.Code[1] = 3
		return
	}
	for _, mcfg := range multiConfig {
		if strings.HasPrefix(mcfg.Name, "-") {
			commandLine = append(commandLine, mcfg.Name)
		} else {
			commandLine = append(commandLine, "--"+mcfg.Name)
		}
		for _, val := range mcfg.Value {
			if val != "" {
				commandLine = append(commandLine, val)
			}
		}
	}
	result.CommandLine = strings.Join(commandLine, " ")
	retries := 0
	dtStart := time.Now()
	for {
		if ctx.DryRun {
			if ctx.DryRunSeconds > 0 {
				time.Sleep(time.Duration(ctx.DryRunSeconds) * time.Second)
			}
			result.Code[1] = ctx.DryRunCode
			if ctx.DryRunCode != 0 {
				result.Err = fmt.Errorf("error: %d", ctx.DryRunCode)
				result.Retries = rand.Intn(ctx.MaxRetry)
			}
			return
		}
		str, err := lib.ExecCommand(ctx, commandLine, nil)
		// p2o.py do not return error even if its backend execution fails
		// we need to capture STDERR and check if there was python exception there
		pyE := false
		if strings.Contains(str, lib.PyException) {
			pyE = true
			err = fmt.Errorf("%s", str)
		}
		if err == nil {
			if ctx.Debug > 0 {
				dtEnd := time.Now()
				lib.Printf("%+v: finished in %v, retries: %d\n", task, dtEnd.Sub(dtStart), retries)
			}
			break
		}
		retries++
		if retries <= ctx.MaxRetry {
			time.Sleep(time.Duration(retries) * time.Second)
			continue
		}
		dtEnd := time.Now()
		if pyE {
			lib.Printf("Python exception for %+v (took %v, tried %d times): %+v\n", commandLine, dtEnd.Sub(dtStart), retries, err)
		} else {
			lib.Printf("Error for %+v (took %v, tried %d times): %+v: %s\n", commandLine, dtEnd.Sub(dtStart), retries, err, str)
			str += fmt.Sprintf(": %+v", err)
		}
		result.Code[1] = 4
		strLen := len(str)
		if strLen > ctx.StripErrorSize {
			sz := ctx.StripErrorSize / 2
			str = str[0:sz] + "..." + str[strLen-sz:strLen]
		}
		result.Err = fmt.Errorf("last retry took %v: %+v", dtEnd.Sub(dtStart), str)
		result.Retries = retries
		return
	}
	result.Retries = retries
	return
}

func finishAfterTimeout(ctx lib.Ctx) {
	time.Sleep(time.Duration(ctx.TimeoutSeconds) * time.Second)
	err := syscall.Kill(syscall.Getpid(), syscall.SIGALRM)
	if err != nil {
		lib.Fatalf("Error: %+v sending timeout signal, exiting\n", err)
	}
}

func main() {
	var ctx lib.Ctx
	dtStart := time.Now()
	ctx.Init()
	err := ensureGrimoireStackAvail(&ctx)
	if err != nil {
		lib.Fatalf("Grimoire stack not available: %+v\n", err)
	}
	go finishAfterTimeout(ctx)
	err = syncGrimoireStack(&ctx)
	if err != nil {
		lib.Fatalf("Grimoire stack sync error: %+v\n", err)
	}
	dtEnd := time.Now()
	lib.Printf("Sync time: %v\n", dtEnd.Sub(dtStart))
}
