package main

import (
	"fmt"
	"io/ioutil"
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

var sshKeyOnce sync.Once

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
	validateFixture(ctx, &fixture, fixtureFile)

	// Synchronize go routine
	if ch != nil {
		ch <- fixture
	}
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
				fixtures = append(fixtures, fixture)
			}
		}
		if ctx.Debug > 0 {
			lib.Printf("Final threads join\n")
		}
		for nThreads > 0 {
			fixture := <-ch
			nThreads--
			fixtures = append(fixtures, fixture)
		}
	} else {
		if ctx.Debug > 0 {
			lib.Printf("Now processing %d fixture files using ST version\n", len(fixtureFiles))
		}
		for _, fixtureFile := range fixtureFiles {
			if fixtureFile == "" {
				continue
			}
			fixtures = append(fixtures, processFixtureFile(nil, ctx, fixtureFile))
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
	knownDsTypes := make(map[string]struct{})
	for _, fixture := range fixtures {
		for _, dataSource := range fixture.DataSources {
			knownDsTypes[dataSource.Slug] = struct{}{}
			for _, endpoint := range dataSource.Endpoints {
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
	ctx.ExecFatal = false
	ctx.ExecOutput = true
	ctx.ExecOutputStderr = true
	defer func() {
		ctx.ExecFatal = true
		ctx.ExecOutput = false
		ctx.ExecOutputStderr = false
	}()
	return processTasks(ctx, &tasks)
}

func processTasks(ctx *lib.Ctx, ptasks *[]lib.Task) error {
	tasks := *ptasks
	thrN := lib.GetThreadsNum(ctx)
	failed := [][2]int{}
	processed := 0
	all := len(tasks)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGUSR1)
	go func() {
		for {
			sig := <-sigs
			lib.Printf("Processed %d/%d (%.2f%%)\n", processed, all, (float64(processed)*100.0)/float64(all))
			if sig == syscall.SIGINT {
				os.Exit(1)
			}
		}
	}()
	if thrN > 1 {
		if ctx.Debug >= 0 {
			lib.Printf("Processing %d tasks using MT%d version\n", len(tasks), thrN)
		}
		ch := make(chan [2]int)
		nThreads := 0
		for idx, task := range tasks {
			go processTask(ch, ctx, idx, &task)
			nThreads++
			if nThreads == thrN {
				res := <-ch
				nThreads--
				if res[1] != 0 {
					failed = append(failed, res)
				}
				processed++
			}
		}
		if ctx.Debug > 0 {
			lib.Printf("Final threads join\n")
		}
		for nThreads > 0 {
			res := <-ch
			nThreads--
			if res[1] != 0 {
				failed = append(failed, res)
			}
			processed++
		}
	} else {
		if ctx.Debug >= 0 {
			lib.Printf("Processing %d tasks using ST version\n", len(tasks))
		}
		for idx, task := range tasks {
			res := processTask(nil, ctx, idx, &task)
			if res[1] != 0 {
				failed = append(failed, res)
			}
			processed++
		}
	}
	lFailed := len(failed)
	if lFailed > 0 {
		lib.Printf("Failed tasks: %+v\n", lFailed)
		for _, res := range failed {
			lib.Printf("Failed: %+v: %s\n", tasks[res[0]], lib.ErrorStrings[res[1]])
		}
	}
	return nil
}

func addSSHPrivKey(ctx *lib.Ctx, key string) bool {
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
	if ctx.Debug > 0 {
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
			lAry := len(ary)
			e = append(e, ary[lAry-2])
			e = append(e, ary[lAry-1])
		} else {
			e = append(e, endpoint)
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
	//discourse\|jenkins\|dockerhub\|bugzilla\|meetup
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
	} else {
		fail = true
	}
	return
}

func processTask(ch chan [2]int, ctx *lib.Ctx, idx int, task *lib.Task) (res [2]int) {
	// Ensure to unlock thread when finishing
	defer func() {
		// Synchronize go routine
		if ch != nil {
			ch <- res
		}
	}()
	if ctx.Debug > 1 {
		lib.Printf("Processing: %s\n", task)
	}
	res[0] = idx

	// Handle DS slug
	ds := task.DsSlug
	idxSlug := task.FxSlug + "-" + ds
	idxSlug = strings.Replace(idxSlug, "/", "-", -1)
	var commandLine []string
	if ctx.CmdDebug > 0 {
		commandLine = []string{
			"p2o.py",
			"--enrich",
			"--index",
			idxSlug + "-raw",
			"--index-enrich",
			idxSlug,
			"-e",
			ctx.ElasticURL,
			"--debug",
			"--db-host",
			ctx.ShHost,
			"--db-sortinghat",
			ctx.ShDB,
			"--db-user",
			ctx.ShUser,
			"--db-password",
			ctx.ShPass,
		}
	} else {
		commandLine = []string{
			"p2o.py",
			"--enrich",
			"--index",
			idxSlug + "-raw",
			"--index-enrich",
			idxSlug,
			"-e",
			ctx.ElasticURL,
			"--db-host",
			ctx.ShHost,
			"--db-sortinghat",
			ctx.ShDB,
			"--db-user",
			ctx.ShUser,
			"--db-password",
			ctx.ShPass,
		}
	}
	if ctx.EsBulkSize > 0 {
		commandLine = append(commandLine, "--bulk-size")
		commandLine = append(commandLine, strconv.Itoa(ctx.EsBulkSize))
	}
	if strings.Contains(ds, "/") {
		ary := strings.Split(ds, "/")
		if len(ary) != 2 {
			lib.Printf("%+v: %s\n", task, lib.ErrorStrings[1])
			res[1] = 1
			return
		}
		commandLine = append(commandLine, ary[0])
		commandLine = append(commandLine, "--category")
		commandLine = append(commandLine, ary[1])
		ds = ary[0]
	} else {
		commandLine = append(commandLine, ds)
	}

	// Handle DS endpoint
	eps := massageEndpoint(task.Endpoint, ds)
	if len(eps) == 0 {
		lib.Printf("%+v: %s\n", task, lib.ErrorStrings[2])
		res[1] = 2
		return
	}
	for _, ep := range eps {
		commandLine = append(commandLine, ep)
	}

	// Handle DS config options
	multiConfig, fail := massageConfig(ctx, &task.Config, ds)
	if fail == true {
		lib.Printf("%+v: %s\n", task, lib.ErrorStrings[3])
		res[1] = 3
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
	// FIXME: remove this once all types of data sources are handled
	if ds == lib.Git || ds == lib.GitHub || ds == lib.Confluence || ds == lib.Gerrit || ds == lib.Jira || ds == lib.Slack || ds == lib.GroupsIO || ds == lib.Pipermail {
		if false {
			return
		}
		trials := 0
		dtStart := time.Now()
		for {
			str, err := lib.ExecCommand(ctx, commandLine, nil)
			// p2o.py do not return error even if its backend execution fails
			// we need to capture STDERR and check if there was python exception there
			if strings.Contains(str, lib.PyException) {
				err = fmt.Errorf("%s", str)
			}
			if err == nil {
				if ctx.Debug > 0 {
					dtEnd := time.Now()
					lib.Printf("%+v: finished in %v, retries: %d\n", task, dtEnd.Sub(dtStart), trials)
				}
				break
			}
			trials++
			if trials < ctx.MaxRetry {
				time.Sleep(time.Duration(trials) * time.Second)
				continue
			}
			dtEnd := time.Now()
			lib.Printf("Error for %+v (took %v, tried %d times): %+v: %s\n", commandLine, dtEnd.Sub(dtStart), trials, err, str)
			res[1] = 4
			return
		}
	} else {
		lib.Printf("%+v\n", commandLine)
	}
	return
}

func main() {
	var ctx lib.Ctx
	dtStart := time.Now()
	ctx.Init()
	err := ensureGrimoireStackAvail(&ctx)
	if err != nil {
		lib.Fatalf("Grimoire stack not available: %+v\n", err)
	}
	err = syncGrimoireStack(&ctx)
	if err != nil {
		lib.Fatalf("Grimoire stack sync error: %+v\n", err)
	}
	dtEnd := time.Now()
	lib.Printf("Sync time: %v\n", dtEnd.Sub(dtStart))
}
