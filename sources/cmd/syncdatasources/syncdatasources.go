package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/LF-Engineering/ssaw/ssawsync"
	lib "github.com/LF-Engineering/sync-data-sources/sources"
	"github.com/google/go-github/github"
	yaml "gopkg.in/yaml.v2"
)

var (
	randInitOnce  sync.Once
	gInfoExternal func()
	gAliasesFunc  func()
	gAliasesMtx   *sync.Mutex
	gKeyMtx       *sync.Mutex
	gCSVMtx       *sync.Mutex
	gToken        string
)

const cOrigin = "sds"

func ensureGrimoireStackAvail(ctx *lib.Ctx) error {
	if ctx.Debug > 0 {
		lib.Printf("Checking grimoire stack availability\n")
	}
	dtStart := time.Now()
	ctx.ExecOutput = true
	home := os.Getenv("HOME")
	dir := home + "/.perceval"
	cmd := exec.Command("mkdir", dir)
	_ = cmd.Run()
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

func validateFixtureFiles(ctx *lib.Ctx, fixtureFiles []string) {
	// Connect to GitHub
	gctx, gcs := lib.GHClient(ctx)

	fixtures := []lib.Fixture{}
	for _, fixtureFile := range fixtureFiles {
		if fixtureFile == "" {
			continue
		}
		fixture := processFixtureFile(gctx, gcs, nil, ctx, fixtureFile)
		if fixture.Disabled != true {
			fixtures = append(fixtures, fixture)
		}
	}
	if len(fixtures) == 0 {
		lib.Fatalf("No fixtures read, this is error, please define at least one")
	}
	if ctx.Debug > 0 {
		lib.Printf("Fixtures: %+v\n", fixtures)
	}
	if !ctx.SkipAliases && ctx.NoMultiAliases {
		// Check if all aliases are unique
		aliases := make(map[string]string)
		for fi, fixture := range fixtures {
			for ai, alias := range fixture.Aliases {
				for ti, to := range alias.To {
					desc := fmt.Sprintf("Fixture #%d: Fn:%s Slug:%s, Alias #%d: From:%s, To: #%d:%s", fi+1, fixture.Fn, fixture.Slug, ai+1, alias.From, ti+1, to)
					got, ok := aliases[to]
					if ok {
						lib.Fatalf("Alias conflict (multi aliases disabled), already exists:\n%s\nWhile trying to add:\n%s\n", got, desc)
					}
					aliases[to] = desc
				}
			}
		}
	}
	// Then for all fixtures defined, all slugs must be unique - check this also
	st := make(map[string]lib.Fixture)
	for _, fixture := range fixtures {
		slug := fixture.Native.Slug
		slug = strings.Replace(slug, "/", "-", -1)
		fixture2, ok := st[slug]
		if ok {
			lib.Fatalf("Duplicate slug %s in fixtures: %+v and %+v\n", slug, fixture, fixture2)
		}
		st[slug] = fixture
	}
	// Check for duplicated endpoints, they may be moved to a shared.yaml file
	checkForSharedEndpoints(&fixtures)
}

func getFixtures(ctx *lib.Ctx) (fixtures []string) {
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
		nil,
	)
	dtEnd := time.Now()
	if err != nil {
		lib.Fatalf("Error finding fixtures (took %v): %+v\n", dtEnd.Sub(dtStart), err)
	}
	fixtures = strings.Split(res, "\n")
	if ctx.Debug > 0 {
		lib.Printf("Fixtures to process: %+v\n", fixtures)
	}
	return
}

func validateConfig(ctx *lib.Ctx, fixture *lib.Fixture, dataSource *lib.DataSource, cfg *lib.Config) {
	if cfg.Name == "" {
		lib.Fatalf("Config %s name in data source %+v in fixture %+v is empty or undefined\n", cfg.RedactedString(), dataSource, fixture)
	}
	if cfg.Value == "" {
		lib.Fatalf("Config %s value in data source %+v in fixture %+v is empty or undefined\n", cfg.RedactedString(), dataSource, fixture)
	}
}

func validateEndpoint(ctx *lib.Ctx, fixture *lib.Fixture, dataSource *lib.DataSource, endpoint *lib.Endpoint) {
	if endpoint.Name == "" {
		lib.Fatalf("Endpoint %+v name in data source %+v in fixture %+v is empty or undefined\n", endpoint, dataSource, fixture)
	}
}

func validateDataSource(ctx *lib.Ctx, fixture *lib.Fixture, index int, dataSource *lib.DataSource) {
	if dataSource.Slug == "" {
		lib.Fatalf("Data source %s in fixture %+v has empty slug or no slug property, slug property must be non-empty\n", dataSource, fixture)
	}
	if ctx.Debug > 2 {
		lib.Printf("Config for %s/%s: %s\n", fixture.Fn, dataSource.Slug, dataSource.Configs())
	}
	if dataSource.MaxFrequency != "" {
		dur, err := time.ParseDuration(dataSource.MaxFrequency)
		if err != nil {
			lib.Fatalf("Cannot parse duration %s in data source: %+v, fixture: %+v\n", dataSource.MaxFrequency, dataSource, fixture)
		}
		dataSource.MaxFreq = dur
		fixture.DataSources[index].MaxFreq = dur
		if ctx.Debug > 0 {
			lib.Printf("Data source %s/%s max sync frequency: %+v\n", fixture.Slug, dataSource.Slug, dataSource.MaxFreq)
		}
	}
	fs := dataSource.Slug + dataSource.IndexSuffix
	fs = strings.Replace(fs, "/", "-", -1)
	dataSource.FullSlug = fs
	fixture.DataSources[index].FullSlug = fs
	for _, cfg := range dataSource.Config {
		validateConfig(ctx, fixture, dataSource, &cfg)
	}
	st := make(map[string]lib.Config)
	for _, cfg := range dataSource.Config {
		name := cfg.Name
		cfg2, ok := st[name]
		if ok {
			lib.Fatalf("Duplicate name %s in config: %s and %s, data source: %s, fixture: %+v\n", name, cfg.RedactedString(), cfg2.RedactedString(), dataSource, fixture)
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
			lib.Fatalf("Duplicate name %s in endpoints: %+v and %+v, data source: %s, fixture: %+v\n", name, endpoint, endpoint2, dataSource, fixture)
		}
		ste[name] = endpoint
	}
	hasProjects := len(dataSource.Projects) > 0
	if hasProjects {
		for _, cfg := range dataSource.Config {
			if cfg.Name == "project" {
				lib.Fatalf(
					"You cannot have projects section defined and 'project' config option set at the same time, config: %s, projects %+v, data source: %s, fixture: %+v\n",
					cfg.RedactedString(),
					dataSource.Projects,
					dataSource,
					fixture,
				)
			}
		}
	}
}

func validateFixture(ctx *lib.Ctx, fixture *lib.Fixture, fixtureFile string) {
	slug := fixture.Native.Slug
	if slug == "" {
		lib.Fatalf("Fixture file %s 'native' property 'slug' is empty which is forbidden\n", fixtureFile)
	}
	nAliases := len(fixture.Aliases)
	if len(fixture.DataSources) == 0 && nAliases == 0 {
		lib.Fatalf("Fixture file %s must have at least one data source defined in 'data_sources' key or at least one alias defined in 'aliases' key\n", fixtureFile)
		//lib.Printf("Fixture file %s has no datasources and no aliases\n", fixtureFile)
	}
	fixture.Fn = fixtureFile
	fixture.Slug = slug
	nEndpoints := 0
	for index, dataSource := range fixture.DataSources {
		validateDataSource(ctx, fixture, index, &dataSource)
		nEndpoints += len(dataSource.Endpoints)
	}
	if nEndpoints == 0 && nAliases == 0 {
		lib.Fatalf("Fixture file %s must have at least one endpoint defined in 'endpoints'/'projects' key or at least one alias defined in 'aliases' key\n", fixtureFile)
	}
	st := make(map[string]lib.DataSource)
	for _, dataSource := range fixture.DataSources {
		slug := dataSource.FullSlug
		dataSource2, ok := st[slug]
		if ok {
			lib.Fatalf("Duplicate slug %s in data sources: %s and %s, fixture: %+v\n", slug, dataSource, dataSource2, fixture)
		}
		st[slug] = dataSource
	}
}

func partialRun(ctx *lib.Ctx) bool {
	// We consider TasksRE and TasksSkipRE safe - they only decide run or not run at the final task level
	if ctx.FixturesRE != nil || ctx.DatasourcesRE != nil || ctx.ProjectsRE != nil || ctx.EndpointsRE != nil || ctx.FixturesSkipRE != nil || ctx.DatasourcesSkipRE != nil || ctx.ProjectsSkipRE != nil || ctx.EndpointsSkipRE != nil {
		return true
	}
	return false
}

func filterFixture(gctx context.Context, gc []*github.Client, ctx *lib.Ctx, fixture *lib.Fixture) (drop bool) {
	n := 0
	dataSources := []lib.DataSource{}
	for _, dataSource := range fixture.DataSources {
		//fmt.Printf("datasource %+v against %s --> %v\n", ctx.DatasourcesRE, dataSource.Slug, ctx.DatasourcesRE.MatchString(dataSource.Slug))
		if (ctx.DatasourcesRE != nil && !ctx.DatasourcesRE.MatchString(dataSource.Slug)) || (ctx.DatasourcesSkipRE != nil && ctx.DatasourcesSkipRE.MatchString(dataSource.Slug)) {
			continue
		}
		endpoints := []lib.Endpoint{}
		for _, endpoint := range dataSource.Endpoints {
			//fmt.Printf("projects %+v against %s --> %v\n", ctx.ProjectsRE, endpoint.Project, ctx.ProjectsRE.MatchString(endpoint.Project))
			if (ctx.ProjectsRE != nil && !ctx.ProjectsRE.MatchString(endpoint.Project)) || (ctx.ProjectsSkipRE != nil && ctx.ProjectsSkipRE.MatchString(endpoint.Project)) {
				continue
			}
			//fmt.Printf("endpoints %+v against %s --> %v\n", ctx.EndpointsRE, endpoint.Name, ctx.EndpointsRE.MatchString(endpoint.Name))
			if (ctx.EndpointsRE != nil && !ctx.EndpointsRE.MatchString(endpoint.Name)) || (ctx.EndpointsSkipRE != nil && ctx.EndpointsSkipRE.MatchString(endpoint.Name)) {
				continue
			}
			endpoints = append(endpoints, endpoint)
			n++
		}
		dataSource.Endpoints = endpoints
		dataSources = append(dataSources, dataSource)
	}
	fixture.DataSources = dataSources
	if ctx.Debug > 0 {
		fmt.Printf("%s has %d after filter\n", fixture.Fn, n)
	}
	if len(fixture.Aliases) > 0 {
		if n == 0 {
			lib.Printf("%s contains only aliases\n", fixture.Fn)
		}
		return false
	}
	return n == 0
}

func postprocessFixture(gctx context.Context, gc []*github.Client, ctx *lib.Ctx, fixture *lib.Fixture) {
	hint := -1
	cache := make(map[string][]string)
	for i, dataSource := range fixture.DataSources {
		for _, projectData := range dataSource.Projects {
			project := projectData.Name
			projectP2O := projectData.P2O
			if project == "" {
				lib.Fatalf("Empty project name entry in %+v, data source %+v, fixture %+v\n", projectData, dataSource, fixture)
			}
			for _, rawEndpoint := range projectData.RawEndpoints {
				if len(rawEndpoint.Projects) > 0 {
					lib.Fatalf("You cannot specify single endpoints projects configuration in data source's 'projects' section, it must be specified data source's 'endpoints' section: data source %+v, fixture %+v\n", dataSource, fixture)
				}
				proj := project
				if rawEndpoint.Project != "" {
					proj = rawEndpoint.Project
				}
				projP2O := projectP2O
				if rawEndpoint.ProjectP2O != nil {
					projP2O = rawEndpoint.ProjectP2O
				}
				name := rawEndpoint.Name
				if projP2O != nil && *projP2O {
					name += ":::" + proj
				}
				fixture.DataSources[i].RawEndpoints = append(
					fixture.DataSources[i].RawEndpoints,
					lib.RawEndpoint{
						Name:       name,
						Project:    proj,
						ProjectP2O: projP2O,
						Flags:      rawEndpoint.Flags,
						Skip:       rawEndpoint.Skip,
						Only:       rawEndpoint.Only,
						Timeout:    rawEndpoint.Timeout,
						CopyFrom:   rawEndpoint.CopyFrom,
					},
				)
			}
		}
		for j, rawEndpoint := range fixture.DataSources[i].RawEndpoints {
			for k := range rawEndpoint.Projects {
				fixture.DataSources[i].RawEndpoints[j].Projects[k].Origin = rawEndpoint.Name
			}
			epType, ok := rawEndpoint.Flags["type"]
			if ctx.OnlyValidate && ctx.SkipValGitHubAPI {
				ok = false
			}
			p2o := false
			if rawEndpoint.ProjectP2O != nil {
				p2o = *(rawEndpoint.ProjectP2O)
			}
			name := rawEndpoint.Name
			if p2o && rawEndpoint.Project != "" {
				name += ":::" + rawEndpoint.Project
			}
			var tmout time.Duration
			if rawEndpoint.Timeout != nil {
				var err error
				tmout, err = time.ParseDuration(*rawEndpoint.Timeout)
				lib.FatalOnError(err)
			}
			if !ok {
				fixture.DataSources[i].Endpoints = append(
					fixture.DataSources[i].Endpoints,
					lib.Endpoint{
						Name:       name,
						Project:    rawEndpoint.Project,
						ProjectP2O: p2o,
						Projects:   rawEndpoint.Projects,
						Timeout:    tmout,
						CopyFrom:   rawEndpoint.CopyFrom,
					},
				)
				continue
			}
			for _, skip := range rawEndpoint.Skip {
				skipRE, err := regexp.Compile(skip)
				lib.FatalOnError(err)
				rawEndpoint.SkipREs = append(rawEndpoint.SkipREs, skipRE)
			}
			fixture.DataSources[i].RawEndpoints[j].SkipREs = rawEndpoint.SkipREs
			for _, only := range rawEndpoint.Only {
				onlyRE, err := regexp.Compile(only)
				lib.FatalOnError(err)
				rawEndpoint.OnlyREs = append(rawEndpoint.OnlyREs, onlyRE)
			}
			fixture.DataSources[i].RawEndpoints[j].OnlyREs = rawEndpoint.OnlyREs
			switch epType {
			case "gerrit_org":
				gerrit := strings.TrimSpace(rawEndpoint.Name)
				repos, ok := cache[gerrit]
				if !ok {
					var err error
					repos, err = lib.GetGerritRepos(ctx, gerrit)
					if err != nil {
						lib.Printf("Error getting gerrit repos list for: %s: error: %+v\n", gerrit, err)
						continue
					}
					cache[gerrit] = repos
				}
				if ctx.Debug > 0 {
					lib.Printf("Gerrit %s repos: %+v\n", gerrit, repos)
				}
				for _, repo := range repos {
					if !lib.EndpointIncluded(ctx, &rawEndpoint, repo) {
						continue
					}
					if p2o && rawEndpoint.Project != "" {
						repo += ":::" + rawEndpoint.Project
					}
					fixture.DataSources[i].Endpoints = append(
						fixture.DataSources[i].Endpoints,
						lib.Endpoint{
							Name:       repo,
							Project:    rawEndpoint.Project,
							ProjectP2O: p2o,
							Projects:   rawEndpoint.Projects,
							Timeout:    tmout,
							CopyFrom:   rawEndpoint.CopyFrom,
						},
					)
				}
			case "rocketchat_server":
				srv := strings.TrimSpace(rawEndpoint.Name)
				channels, ok := cache[srv]
				if !ok {
					token := ""
					uid := ""
					for _, cfg := range dataSource.Config {
						if cfg.Name == "api-token" {
							token = cfg.Value
						} else if cfg.Name == "user-id" {
							uid = cfg.Value
						}
					}
					var err error
					channels, err = lib.GetRocketChatChannels(ctx, srv, token, uid)
					if err != nil {
						lib.Printf("Error getting channels list for rocketchat server: %s: error: %+v\n", srv, err)
						continue
					}
					cache[srv] = channels
				}
				if ctx.Debug > 0 {
					lib.Printf("RocketChat srv %s channels: %+v\n", srv, channels)
				}
				for _, channel := range channels {
					if !lib.EndpointIncluded(ctx, &rawEndpoint, channel) {
						continue
					}
					name := srv + " " + channel
					if p2o && rawEndpoint.Project != "" {
						name += ":::" + rawEndpoint.Project
					}
					fixture.DataSources[i].Endpoints = append(
						fixture.DataSources[i].Endpoints,
						lib.Endpoint{
							Name:       name,
							Project:    rawEndpoint.Project,
							ProjectP2O: p2o,
							Projects:   rawEndpoint.Projects,
							Timeout:    tmout,
							CopyFrom:   rawEndpoint.CopyFrom,
						},
					)
				}
			case "github_org":
				if hint < 0 {
					aHint, _, _, _ := lib.GetRateLimits(gctx, ctx, gc, true)
					hint = aHint
				}
				arr := strings.Split(rawEndpoint.Name, "/")
				ary := []string{}
				l := len(arr) - 1
				for i, s := range arr {
					if i == l && s == "" {
						break
					}
					ary = append(ary, s)
				}
				lAry := len(ary)
				org := ary[lAry-1]
				root := strings.Join(ary[0:lAry-1], "/")
				repos, ok := cache[org]
				if !ok {
					opt := &github.RepositoryListByOrgOptions{Type: "public"} // can also use "all"
					opt.PerPage = 100
					repos = []string{}
					for {
						repositories, response, err := gc[hint].Repositories.ListByOrg(gctx, org, opt)
						if err != nil {
							lib.Printf("Error getting repositories list for org: %s: response: %+v, error: %+v\n", org, response, err)
							break
						}
						for _, repo := range repositories {
							if repo.Name != nil {
								name := root + "/" + org + "/" + *(repo.Name)
								repos = append(repos, name)
							}
						}
						if response.NextPage == 0 {
							break
						}
						opt.Page = response.NextPage
					}
					cache[org] = repos
				}
				if ctx.Debug > 0 {
					lib.Printf("Org %s repos: %+v\n", org, repos)
				}
				for _, repo := range repos {
					if !lib.EndpointIncluded(ctx, &rawEndpoint, repo) {
						continue
					}
					name := repo
					if p2o && rawEndpoint.Project != "" {
						name += ":::" + rawEndpoint.Project
					}
					fixture.DataSources[i].Endpoints = append(
						fixture.DataSources[i].Endpoints,
						lib.Endpoint{
							Name:       name,
							Project:    rawEndpoint.Project,
							ProjectP2O: p2o,
							Projects:   rawEndpoint.Projects,
							Timeout:    tmout,
							CopyFrom:   rawEndpoint.CopyFrom,
						},
					)
				}
			case "github_user":
				if hint < 0 {
					aHint, _, _, _ := lib.GetRateLimits(gctx, ctx, gc, true)
					hint = aHint
				}
				arr := strings.Split(rawEndpoint.Name, "/")
				ary := []string{}
				l := len(arr) - 1
				for i, s := range arr {
					if i == l && s == "" {
						break
					}
					ary = append(ary, s)
				}
				lAry := len(ary)
				user := ary[lAry-1]
				root := strings.Join(ary[0:lAry-1], "/")
				repos, ok := cache[user]
				if !ok {
					opt := &github.RepositoryListOptions{Type: "public"}
					opt.PerPage = 100
					repos = []string{}
					for {
						repositories, response, err := gc[hint].Repositories.List(gctx, user, opt)
						if err != nil {
							lib.Printf("Error getting repositories list for user: %s: response: %+v, error: %+v\n", user, response, err)
							break
						}
						for _, repo := range repositories {
							if repo.Name != nil {
								name := root + "/" + user + "/" + *(repo.Name)
								repos = append(repos, name)
							}
						}
						if response.NextPage == 0 {
							break
						}
						opt.Page = response.NextPage
					}
					cache[user] = repos
				}
				if ctx.Debug > 0 {
					lib.Printf("User %s repos: %+v\n", user, repos)
				}
				for _, repo := range repos {
					if !lib.EndpointIncluded(ctx, &rawEndpoint, repo) {
						continue
					}
					name := repo
					if p2o && rawEndpoint.Project != "" {
						name += ":::" + rawEndpoint.Project
					}
					fixture.DataSources[i].Endpoints = append(
						fixture.DataSources[i].Endpoints,
						lib.Endpoint{
							Name:       name,
							Project:    rawEndpoint.Project,
							ProjectP2O: p2o,
							Projects:   rawEndpoint.Projects,
							Timeout:    tmout,
							CopyFrom:   rawEndpoint.CopyFrom,
						},
					)
				}
			default:
				lib.Printf("Warning: unknown raw endpoint type: %s\n", epType)
				name := rawEndpoint.Name
				if p2o && rawEndpoint.Project != "" {
					name += ":::" + rawEndpoint.Project
				}
				fixture.DataSources[i].Endpoints = append(
					fixture.DataSources[i].Endpoints,
					lib.Endpoint{
						Name:       name,
						Project:    rawEndpoint.Project,
						ProjectP2O: p2o,
						Projects:   rawEndpoint.Projects,
						Timeout:    tmout,
						CopyFrom:   rawEndpoint.CopyFrom,
					},
				)
				continue
			}
		}
	}
	for ai, alias := range fixture.Aliases {
		var idxSlug string
		if strings.HasPrefix(alias.From, "bitergia-") {
			idxSlug = alias.From
		} else {
			idxSlug = "sds-" + alias.From
		}
		idxSlug = strings.Replace(idxSlug, "/", "-", -1)
		fixture.Aliases[ai].From = idxSlug
		for ti, to := range alias.To {
			idxSlug := "sds-" + to
			idxSlug = strings.Replace(idxSlug, "/", "-", -1)
			fixture.Aliases[ai].To[ti] = idxSlug
		}
	}
}

func processFixtureFile(gctx context.Context, gc []*github.Client, ch chan lib.Fixture, ctx *lib.Ctx, fixtureFile string) (fixture lib.Fixture) {
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
	slug := fixture.Native.Slug
	if slug == "" {
		lib.Fatalf("Fixture file %s 'native' property has no 'slug' property (or is empty)\n", fixtureFile)
	}
	fixture.Fn = fixtureFile
	fixture.Slug = slug
	if (ctx.FixturesRE != nil && !ctx.FixturesRE.MatchString(fixture.Slug)) || (ctx.FixturesSkipRE != nil && ctx.FixturesSkipRE.MatchString(fixture.Slug)) {
		fixture.Disabled = true
		return
	}
	if fixture.Disabled == true {
		return
	}
	postprocessFixture(gctx, gc, ctx, &fixture)
	if filterFixture(gctx, gc, ctx, &fixture) {
		fixture.Disabled = true
		return
	}
	if ctx.Debug > 0 {
		lib.Printf("Post-processed and filtered %s fixture: %+v\n", fixtureFile, fixture)
	}
	validateFixture(ctx, &fixture, fixtureFile)
	return
}

func processFixtureFiles(ctx *lib.Ctx, fixtureFiles []string) {
	// Connect to GitHub
	gctx, gcs := lib.GHClient(ctx)

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
			go processFixtureFile(gctx, gcs, ch, ctx, fixtureFile)
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
			fixture := processFixtureFile(gctx, gcs, nil, ctx, fixtureFile)
			if fixture.Disabled != true {
				fixtures = append(fixtures, fixture)
			}
		}
	}
	if len(fixtures) == 0 {
		lib.Fatalf("No fixtures read, this is error, please define at least one")
	}
	if ctx.Debug > 0 {
		lib.Printf("Fixtures: %+v\n", fixtures)
	}
	if !ctx.SkipAliases && ctx.NoMultiAliases {
		// Check if all aliases are unique
		aliases := make(map[string]string)
		for fi, fixture := range fixtures {
			for ai, alias := range fixture.Aliases {
				for ti, to := range alias.To {
					desc := fmt.Sprintf("Fixture #%d: Fn:%s Slug:%s, Alias #%d: From:%s, To: #%d:%s", fi+1, fixture.Fn, fixture.Slug, ai+1, alias.From, ti+1, to)
					got, ok := aliases[to]
					if ok {
						lib.Fatalf("Alias conflict (multi aliases disabled), already exists:\n%s\nWhile trying to add:\n%s\n", got, desc)
					}
					aliases[to] = desc
				}
			}
		}
	}
	// Then for all fixtures defined, all slugs must be unique - check this also
	st := make(map[string]lib.Fixture)
	for _, fixture := range fixtures {
		slug := fixture.Native.Slug
		slug = strings.Replace(slug, "/", "-", -1)
		fixture2, ok := st[slug]
		if ok {
			lib.Fatalf("Duplicate slug %s in fixtures: %+v and %+v\n", slug, fixture, fixture2)
		}
		st[slug] = fixture
	}
	// Check for duplicated endpoints, they may be moved to a shared.yaml file
	checkForSharedEndpoints(&fixtures)
	// Drop unused indexes, rename indexes if needed, drop unused aliases
	didRenames := false
	if !ctx.SkipDropUnused && !ctx.OnlyP2O {
		if ctx.NodeNum > 1 {
			// sdsmtx is an ES wide mutex-like index for blocking between concurrent nodes
			lib.EnsureIndex(ctx, lib.SDSMtx, false)
			// all nodes lock "rename" ES mutex, but only 1st node will unlock it (after all renames if any)
			if ctx.NodeIdx > 0 {
				mtx := fmt.Sprintf("rename-node-%d", ctx.NodeIdx)
				lib.Printf("Node %d locking ES mutex: %s\n", ctx.NodeIdx, mtx)
				giantLock(ctx, mtx)
			}
		}
		didRenames = processIndexes(ctx, &fixtures)
		if ctx.NodeNum > 1 && ctx.NodeIdx > 0 {
			// now wait for 1st node to finish renames (if any)
			lib.Printf("Node %d waiting for master to finish dropping/renaming indexes\n", ctx.NodeIdx)
			giantWait(ctx, fmt.Sprintf("rename-node-%d", ctx.NodeIdx), lib.Unlocked)
			lib.Printf("Node %d mutex processing finished\n", ctx.NodeIdx)
		}
		// Aliases (don't have to be inside mutex)
		if !ctx.SkipAliases {
			dropUnusedAliases(ctx, &fixtures)
		}
	}
	// SDS data index
	if !ctx.SkipEsData {
		lib.EnsureIndex(ctx, "sdsdata", false)
	}
	// SDS sync-info index
	if !ctx.SkipSyncInfo && (!ctx.DryRun || ctx.DryRunAllowSyncInfo) {
		lib.EnsureIndex(ctx, "sdssyncinfo", false)
	}
	// Tasks
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
				name := endpoint.Name
				if endpoint.ProjectP2O {
					ary := strings.Split(name, ":::")
					name = ary[0]
				}
				tasks = append(
					tasks,
					lib.Task{
						Project:    endpoint.Project,
						ProjectP2O: endpoint.ProjectP2O,
						Projects:   endpoint.Projects,
						Timeout:    endpoint.Timeout,
						CopyFrom:   endpoint.CopyFrom,
						Endpoint:   name,
						Config:     dataSource.Config,
						DsSlug:     dataSource.Slug,
						DsFullSlug: dataSource.FullSlug,
						FxSlug:     fixture.Slug,
						FxFn:       fixture.Fn,
						MaxFreq:    dataSource.MaxFreq,
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
	if !ctx.SkipSortDuration {
		sortByDuration(ctx, tasks)
	}
	ctx.ExecFatal = false
	ctx.ExecOutput = true
	ctx.ExecOutputStderr = true
	defer func() {
		ctx.ExecFatal = true
		ctx.ExecOutput = false
		ctx.ExecOutputStderr = false
	}()
	gKeyMtx = &sync.Mutex{}
	gAliasesMtx = &sync.Mutex{}
	gCSVMtx = &sync.Mutex{}
	gAliasesFunc = func() {
		if !ctx.SkipAliases && !ctx.OnlyP2O {
			lib.Printf("Processing aliases\n")
			gAliasesMtx.Lock()
			defer func() {
				gAliasesMtx.Unlock()
			}()
			if ctx.CleanupAliases {
				processAliases(ctx, &fixtures, lib.Delete)
			}
			processAliases(ctx, &fixtures, lib.Put)
		}
	}
	if !ctx.OnlyP2O && didRenames {
		gAliasesFunc()
	}
	// We *try* to enrich external indexes, but we don't care if that actually suceeded
	ch := make(chan struct{})
	if !ctx.OnlyP2O {
		go func(ch chan struct{}) {
			enrichAndDedupExternalIndexes(ctx, &fixtures, &tasks)
			ch <- struct{}{}
		}(ch)
	}
	if !ctx.SkipSSAW {
		if ctx.SSAWFreq > 0 {
			go ssawLoop(ctx)
		}
	}
	// Most important work
	rslt := processTasks(ctx, &tasks, dss)
	if !ctx.SkipSSAW {
		ssawSync(ctx, true)
	}
	if !ctx.OnlyP2O {
		gAliasesFunc()
		<-ch
	}
	if rslt != nil {
		lib.Fatalf("Process tasks error: %+v\n", rslt)
	}
}

func sortByDuration(ctx *lib.Ctx, tasks []lib.Task) {
	if ctx.DryRun && !ctx.DryRunAllowSortDuration {
		return
	}
	lib.Printf("Determining running order for %d tasks\n", len(tasks))
	thrN := lib.GetThreadsNum(ctx)
	if thrN > 1 {
		ch := make(chan struct{})
		nThreads := 0
		for i := range tasks {
			go func(ch chan struct{}, i int) {
				defer func() {
					ch <- struct{}{}
				}()
				setLastDuration(ctx, &tasks[i])
			}(ch, i)
			nThreads++
			if nThreads == thrN {
				<-ch
				nThreads--
			}
		}
		for nThreads > 0 {
			<-ch
			nThreads--
		}
	} else {
		for i := range tasks {
			setLastDuration(ctx, &tasks[i])
		}
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].Millis < tasks[j].Millis
	})
	if ctx.Debug > 0 {
		for i, task := range tasks {
			lib.Printf("#%d %s / %s (%d)\n", i, task.DsFullSlug, task.Endpoint, task.Millis)
		}
	}
	lib.Printf("Determined running order for %d tasks\n", len(tasks))
}

func setLastDuration(ctx *lib.Ctx, task *lib.Task) {
	fds := task.DsFullSlug
	task.Millis = int64(9223372036854775807)
	idxSlug := "sds-" + task.FxSlug + "-" + fds
	idxSlug = strings.Replace(idxSlug, "/", "-", -1)
	data := fmt.Sprintf(
		`{"query":"select data_sync_success_dt::long - data_sync_attempt_dt::long as millis from sdssyncinfo where index = '%s' and endpoint = '%s' and data_sync_success_dt is not null and data_sync_attempt_dt is not null order by dt desc limit 1"}`,
		jsonEscape(idxSlug),
		jsonEscape(task.Endpoint),
	)
	payloadBytes := []byte(data)
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Post
	url := fmt.Sprintf("%s/_sql?format=csv", ctx.ElasticURL)
	rurl := "/_sql?format=csv"
	req, err := http.NewRequest(method, url, payloadBody)
	if err != nil {
		lib.Printf("new request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("do request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s status:%d data:%s\n%s\n", method, rurl, resp.StatusCode, data, body)
		return
	}
	reader := csv.NewReader(resp.Body)
	row := []string{}
	n := 0
	for {
		row, err = reader.Read()
		if err == io.EOF {
			err = nil
			break
		} else if err != nil {
			lib.Printf("Read CSV error: %v/%T\n", err, err)
			return
		}
		if n == 0 {
			n++
			continue
		}
		millis, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			lib.Printf("Cannot parse integer from '%s' error: %v\n", row[0], err)
			return
		}
		if millis > 0 {
			task.Millis = millis
			if ctx.Debug > 0 {
				lib.Printf("Last duration %s / %s ---> %d\n", idxSlug, task.Endpoint, task.Millis)
			}
		}
		break
	}
	return
}

func ssawSync(ctx *lib.Ctx, final bool) {
	if ctx.DryRun && !ctx.DryRunAllowSSAW {
		return
	}
	e := os.Setenv("SYNC_URL", ctx.SSAWURL)
	if e != nil {
		lib.Printf("ssaw failed to set SYNC_URL environment variable: %v\n", e)
	}
	origin := cOrigin + "-"
	if final {
		origin += "final"
	} else {
		origin += "partial"
	}
	e = ssawsync.Sync(origin)
	if e != nil {
		lib.Printf("ssaw sync error: %v\n", e)
	}
}

func ssawLoop(ctx *lib.Ctx) {
	if ctx.DryRun && !ctx.DryRunAllowSSAW {
		return
	}
	for {
		time.Sleep(time.Duration(ctx.SSAWFreq) * time.Second)
		ssawSync(ctx, false)
	}
}

func dedupOrigins(first, second []string) (shared, onlyFirst []string) {
	sharedMap := make(map[string]struct{})
	for _, f := range first {
		hit := false
		for _, s := range second {
			if strings.Contains(f, s) || strings.Contains(s, f) {
				sharedMap[f] = struct{}{}
				sharedMap[s] = struct{}{}
				hit = true
			}
		}
		if !hit {
			onlyFirst = append(onlyFirst, f)
		}
	}
	for s := range sharedMap {
		shared = append(shared, s)
	}
	return
}

func dropOriginsInternal(ctx *lib.Ctx, index string, origins []string) (ok bool) {
	nOrigins := len(origins)
	if nOrigins < 1 {
		return
	}
	if nOrigins > 500 {
		lib.Printf("Too many origins to delete, maximum is 500: %+v\n", origins)
		return
	}
	query := "origin:("
	lastI := nOrigins - 1
	for i, origin := range origins {
		query += "\"" + origin + "\""
		if i < lastI {
			query += " OR "
		} else {
			query += ")"
		}
	}
	if ctx.DryRun {
		if !ctx.DryRunAllowDedup {
			lib.Printf("Would dedup bitergia index %s via delete: %s\n", index, query)
			return
		}
		lib.Printf("Dry run allowed dedup bitergia index %s via delete: %s\n", index, query)
	}
	trials := 0
	for {
		deleted := deleteByQuery(ctx, index, query)
		if deleted {
			if trials > 0 {
				lib.Printf("Deleted from %s:%s after %d/%d trials\n", index, query, trials, ctx.MaxDeleteTrials)
			}
			break
		}
		trials++
		if trials == ctx.MaxDeleteTrials {
			lib.Printf("ERROR: Failed to delete from %s:%s, tried %d times\n", index, query, ctx.MaxDeleteTrials)
			return
		}
		time.Sleep(time.Duration(10*trials) * time.Millisecond)
	}
	ok = true
	return
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "killed by task timeout")
}

func dropOrigins(ctx *lib.Ctx, index string, origins []string) (ok bool) {
	nOrigins := len(origins)
	bucketSize := 500
	failed := false
	nBuckets := (nOrigins / bucketSize) + 1
	for i := 0; i < nBuckets; i++ {
		from := i * bucketSize
		to := from + bucketSize
		if to > nOrigins {
			to = nOrigins
		}
		deleted := dropOriginsInternal(ctx, index, origins[from:to])
		if !deleted {
			failed = true
			lib.Printf("dropOrigins: bucket #%d/%d failed (%d origins), continuying\n", i+1, nBuckets, nOrigins)
		}
	}
	ok = !failed
	return
}

func enrichAndDedupExternalIndexes(ctx *lib.Ctx, pfixtures *[]lib.Fixture, ptasks *[]lib.Task) {
	if ctx.SkipExternal || ctx.SkipSH || ctx.SkipAffs {
		lib.Printf("Skip External or Skip SH or Skip affs is set, skipping enriching external indices\n")
		return
	}
	st := time.Now()
	lib.Printf("Enrich external indices: check node\n")
	// If possible run on random non-master node, but if there is only one node then run on that node (master)
	if ctx.NodeNum > 1 {
		_, _, day := time.Now().Date()
		nodeIndex := (day % (ctx.NodeNum - 1)) + 1
		if ctx.NodeIdx != nodeIndex {
			lib.Printf("This will only run on #%d node on %d day of month\n", nodeIndex, day)
			return
		}
	}
	if !ctx.SkipEsData && !ctx.SkipCheckFreq {
		lib.Printf("Enrich external indices: check last run\n")
		freqOK := checkSyncFreq(ctx, nil, lib.Bitergia, lib.External, ctx.EnrichExternalFreq)
		if !freqOK {
			return
		}
	}
	lib.Printf("Enrich external indices: running\n")
	fixtures := *pfixtures
	tasks := *ptasks
	manualEnrich := make(map[string][]string)
	for _, fixture := range fixtures {
		for _, aliasFrom := range fixture.Aliases {
			if !strings.HasPrefix(aliasFrom.From, "bitergia-") {
				continue
			}
			for _, aliasTo := range aliasFrom.To {
				if !strings.HasSuffix(aliasTo, "-raw") {
					ary, ok := manualEnrich[aliasTo]
					if !ok {
						manualEnrich[aliasTo] = []string{aliasFrom.From}
					} else {
						ary = append(ary, aliasFrom.From)
						manualEnrich[aliasTo] = ary
					}
				}
			}
		}
	}
	indexToTask := make(map[string]lib.Task)
	dataSourceToTask := make(map[string]lib.Task)
	dsToCategory := make(map[string]map[string]struct{})
	for i, task := range tasks {
		idxSlug := "sds-" + task.FxSlug + "-" + task.DsSlug
		idxSlug = strings.Replace(idxSlug, "/", "-", -1)
		_, ok := indexToTask[idxSlug]
		if !ok {
			indexToTask[idxSlug] = tasks[i]
		}
		ds := task.DsSlug
		_, ok = dataSourceToTask[ds]
		if !ok {
			dataSourceToTask[ds] = tasks[i]
		}
		if strings.Contains(ds, "/") {
			ary := strings.Split(ds, "/")
			strippedDs := ary[0]
			category := ary[1]
			_, ok = dataSourceToTask[strippedDs]
			if !ok {
				dataSourceToTask[strippedDs] = tasks[i]
			}
			categories, ok := dsToCategory[strippedDs]
			if !ok {
				dsToCategory[strippedDs] = map[string]struct{}{category: {}}
			} else {
				categories[category] = struct{}{}
				dsToCategory[strippedDs] = categories
			}
		}
	}
	newTasks := []lib.Task{}
	processedIndices := make(map[string]struct{})
	for sdsIndex, bitergiaIndices := range manualEnrich {
		sdsTask, ok := indexToTask[sdsIndex]
		if !ok {
			lib.Printf("WARNING: External index/indices have no corresponding configuration in SDS: %+v\n", bitergiaIndices)
			for _, bitergiaIndex := range bitergiaIndices {
				dsSlug := figureOutDatasourceFromIndexName(bitergiaIndex)
				if dsSlug == "" {
					lib.Printf("ERROR(not fatal): External index %s: cannot guess data source type from index name\n", bitergiaIndex)
					continue
				}
				categories, ok := dsToCategory[dsSlug]
				if !ok {
					categories = map[string]struct{}{"": {}}
				}
				for category := range categories {
					ds := dsSlug
					if category != "" {
						ds += "/" + category
					}
					randomSdsTask, ok := dataSourceToTask[dsSlug]
					if !ok {
						lib.Printf("ERROR(not fatal): External index %s: guessed data source type %s from index name: cannot find any SDS index of that type\n", bitergiaIndex, ds)
						continue
					}
					endpoints, _ := figureOutEndpoints(ctx, bitergiaIndex, ds)
					for _, endpoint := range endpoints {
						newTasks = append(
							newTasks,
							lib.Task{
								Project:       "",
								ProjectP2O:    false,
								Endpoint:      endpoint,
								Config:        randomSdsTask.Config,
								DsSlug:        ds,
								DsFullSlug:    randomSdsTask.DsFullSlug,
								FxSlug:        "random:" + randomSdsTask.FxSlug,
								FxFn:          "random:" + randomSdsTask.FxFn,
								MaxFreq:       randomSdsTask.MaxFreq,
								ExternalIndex: bitergiaIndex,
							},
						)
						processedIndices[bitergiaIndex] = struct{}{}
					}
				}
			}
			continue
		}
		for _, bitergiaIndex := range bitergiaIndices {
			bitergiaEndpoints, bitergiaOrigins := figureOutEndpoints(ctx, bitergiaIndex, sdsTask.DsSlug)
			endpoints := []string{}
			if ctx.SkipDedup {
				endpoints = bitergiaEndpoints
			} else {
				sdsIndex := "sds-" + sdsTask.FxSlug + "-" + sdsTask.DsSlug
				sdsIndex = strings.Replace(sdsIndex, "/", "-", -1)
				_, sdsOrigins := figureOutEndpoints(ctx, sdsIndex, sdsTask.DsSlug)
				originsShared, originsOnlyBitergia := dedupOrigins(bitergiaOrigins, sdsOrigins)
				if ctx.Debug > 0 {
					lib.Printf("=========> %s vs. %s\nBITE %+v\nSDS  %+v\nSHAR %+v\nONLY %+v\n", bitergiaIndex, sdsIndex, bitergiaOrigins, sdsOrigins, originsShared, originsOnlyBitergia)
				}
				if len(originsShared) > 0 {
					lib.Printf("Bitergia origins %s:%+v share at least one origin with SDS origins: %s:%+v\n", bitergiaIndex, bitergiaOrigins, sdsIndex, sdsOrigins)
					lib.Printf("Deleting Bitergia/SDS shared origins %s:%+v, bitergia index will only contain origins not present in %s SDS index: %+v\n", bitergiaIndex, originsShared, sdsIndex, originsOnlyBitergia)
					if len(originsOnlyBitergia) == 0 {
						lib.Printf("NOTICE: bitergia index %s is fully duplicated in SDS, so it basically can be removed from config fixtures\n", bitergiaIndex)
					}
					// We don't do this in multiple threads because deleting data from ES is a very heavy operation and doing that in multiple threads
					// will not make it any faster. It will only result in more parallel timeouts.
					if !dropOrigins(ctx, bitergiaIndex, originsShared) {
						lib.Printf("Failed to delete %+v origins from %s\n", originsShared, bitergiaIndex)
					}
					endpoints, _ = figureOutEndpoints(ctx, bitergiaIndex, sdsTask.DsSlug)
				} else {
					endpoints = bitergiaEndpoints
				}
			}
			for _, endpoint := range endpoints {
				newTasks = append(
					newTasks,
					lib.Task{
						Project:       "",
						ProjectP2O:    false,
						Endpoint:      endpoint,
						Config:        sdsTask.Config,
						DsSlug:        sdsTask.DsSlug,
						DsFullSlug:    sdsTask.DsFullSlug,
						FxSlug:        sdsTask.FxSlug,
						FxFn:          sdsTask.FxFn,
						MaxFreq:       sdsTask.MaxFreq,
						ExternalIndex: bitergiaIndex,
					},
				)
				processedIndices[bitergiaIndex] = struct{}{}
			}
		}
	}
	// Actual processing
	thrN := lib.GetThreadsNum(ctx)
	var sshKeyMtx *sync.Mutex
	if thrN > 1 {
		sshKeyMtx = &sync.Mutex{}
	}
	remainingTasks := make(map[[3]string]struct{})
	processingTasks := make(map[[3]string]struct{})
	processedTasks := make(map[[3]string]struct{})
	succeededTasks := make(map[[3]string]struct{})
	erroredTasks := make(map[[3]string]struct{})
	for _, task := range newTasks {
		key := [3]string{task.ExternalIndex, task.DsSlug, task.Endpoint}
		remainingTasks[key] = struct{}{}
	}
	infoMtx := &sync.Mutex{}
	updateInfo := func(in bool, result [4]string) {
		key := [3]string{result[0], result[1], result[2]}
		if in {
			infoMtx.Lock()
			delete(remainingTasks, key)
			processingTasks[key] = struct{}{}
			infoMtx.Unlock()
			return
		}
		ok := result[3] == ""
		infoMtx.Lock()
		delete(processingTasks, key)
		processedTasks[key] = struct{}{}
		if ok {
			succeededTasks[key] = struct{}{}
		} else {
			erroredTasks[key] = struct{}{}
		}
		infoMtx.Unlock()
	}
	gInfoExternal = func() {
		infoMtx.Lock()
		msg := ""
		if len(processingTasks) > 0 {
			msg += fmt.Sprintf("Processing: %d\n", len(processingTasks))
			ary := []string{}
			for task := range processingTasks {
				ary = append(ary, task[0]+":"+task[1]+":"+task[2])
			}
			sort.Strings(ary)
			msg += strings.Join(ary, "\n") + "\n"
		}
		if len(processedTasks) > 0 {
			msg += fmt.Sprintf("Processed: %d\n", len(processedTasks))
		}
		if len(succeededTasks) > 0 {
			msg += fmt.Sprintf("Succeeded: %d\n", len(succeededTasks))
			ary := []string{}
			for task := range succeededTasks {
				ary = append(ary, task[0]+":"+task[1]+":"+task[2])
			}
			sort.Strings(ary)
			msg += strings.Join(ary, "\n") + "\n"
		}
		if len(erroredTasks) > 0 {
			msg += fmt.Sprintf("Errored: %d\n", len(erroredTasks))
			ary := []string{}
			for task := range erroredTasks {
				ary = append(ary, task[0]+":"+task[1]+":"+task[2])
			}
			sort.Strings(ary)
			msg += strings.Join(ary, "\n") + "\n"
		}
		if len(remainingTasks) > 0 {
			msg += fmt.Sprintf("Remaining: %d\n", len(remainingTasks))
			ary := []string{}
			for task := range remainingTasks {
				ary = append(ary, task[0]+":"+task[1]+":"+task[2])
			}
			sort.Strings(ary)
			msg += strings.Join(ary, "\n") + "\n"
		}
		infoMtx.Unlock()
		msg = strings.TrimSpace(msg)
		msgs := strings.Split(msg, "\n")
		if len(msgs) > 0 {
			lib.Printf("External indices enrichment info:\n")
		}
		for _, line := range msgs {
			lib.Printf("%s\n", line)
		}
	}
	enrichExternal := func(ch chan [4]string, tsk lib.Task) (result [4]string) {
		defer func() {
			updateInfo(false, result)
			if ch != nil {
				ch <- result
			}
		}()
		result[0] = tsk.ExternalIndex
		result[1] = tsk.DsSlug
		result[2] = tsk.Endpoint
		updateInfo(true, result)
		ds := tsk.DsSlug
		idxSlug := tsk.ExternalIndex
		commandLine := []string{
			"p2o.py",
			"--enrich",
			"--index-enrich",
			idxSlug,
			"--only-enrich",
			"--refresh-identities",
			"--no_incremental",
			"-e",
			ctx.ElasticURL,
		}
		redactedCommandLine := make([]string, len(commandLine))
		copy(redactedCommandLine, commandLine)
		redactedCommandLine[len(redactedCommandLine)-1] = lib.Redacted
		// This only enables p2o.py -g flag (so only subcommand is executed with debug mode)
		if !ctx.Silent {
			commandLine = append(commandLine, "-g")
			redactedCommandLine = append(redactedCommandLine, "-g")
		}
		// This enabled debug mode on the p2o.py subcommand als also makes ExecCommand() call run in debug mode
		if ctx.CmdDebug > 0 {
			commandLine = append(commandLine, "--debug")
			redactedCommandLine = append(redactedCommandLine, "--debug")
		}
		if ctx.EsBulkSize > 0 {
			commandLine = append(commandLine, "--bulk-size")
			commandLine = append(commandLine, strconv.Itoa(ctx.EsBulkSize))
			redactedCommandLine = append(redactedCommandLine, "--bulk-size")
			redactedCommandLine = append(redactedCommandLine, strconv.Itoa(ctx.EsBulkSize))
		}
		if ctx.ScrollSize > 0 {
			commandLine = append(commandLine, "--scroll-size")
			commandLine = append(commandLine, strconv.Itoa(ctx.ScrollSize))
			redactedCommandLine = append(redactedCommandLine, "--scroll-size")
			redactedCommandLine = append(redactedCommandLine, strconv.Itoa(ctx.ScrollSize))
		}
		if ctx.ScrollWait > 0 {
			commandLine = append(commandLine, "--scroll-wait")
			commandLine = append(commandLine, strconv.Itoa(ctx.ScrollWait))
			redactedCommandLine = append(redactedCommandLine, "--scroll-wait")
			redactedCommandLine = append(redactedCommandLine, strconv.Itoa(ctx.ScrollWait))
		}
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
		redactedCommandLine = append(
			redactedCommandLine,
			[]string{
				"--db-host",
				lib.Redacted,
				"--db-sortinghat",
				lib.Redacted,
				"--db-user",
				lib.Redacted,
				"--db-password",
				lib.Redacted,
			}...,
		)
		if strings.Contains(ds, "/") {
			ary := strings.Split(ds, "/")
			if len(ary) != 2 {
				result[3] = fmt.Sprintf("%s: %+v: %s", ds, tsk, lib.ErrorStrings[1])
				return
			}
			commandLine = append(commandLine, ary[0])
			commandLine = append(commandLine, "--category")
			commandLine = append(commandLine, ary[1])
			redactedCommandLine = append(redactedCommandLine, ary[0])
			redactedCommandLine = append(redactedCommandLine, "--category")
			redactedCommandLine = append(redactedCommandLine, ary[1])
			ds = ary[0]
		} else {
			commandLine = append(commandLine, ds)
			redactedCommandLine = append(redactedCommandLine, ds)
		}

		// Handle DS endpoint
		eps := massageEndpoint(tsk.Endpoint, ds)
		if len(eps) == 0 {
			result[3] = fmt.Sprintf("%s: %+v: %s", tsk.Endpoint, tsk, lib.ErrorStrings[2])
			return
		}
		for _, ep := range eps {
			commandLine = append(commandLine, ep)
			redactedCommandLine = append(redactedCommandLine, ep)
		}

		// Handle DS config options
		multiConfig, fail, keyAdded := massageConfig(ctx, &(tsk.Config), ds, idxSlug)
		if fail == true {
			result[3] = fmt.Sprintf("%+v: %s\n", tsk, lib.ErrorStrings[3])
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
			for _, val := range mcfg.RedactedValue {
				if val != "" {
					redactedCommandLine = append(redactedCommandLine, val)
				}
			}
		}
		// Handle DS project
		if tsk.ProjectP2O && tsk.Project != "" {
			commandLine = append(commandLine, "--project", tsk.Project)
			redactedCommandLine = append(redactedCommandLine, "--project", tsk.Project)
		}
		rcl := strings.Join(redactedCommandLine, " ")
		retries := 0
		dtStart := time.Now()
		for {
			if ctx.DryRun {
				if keyAdded {
					if sshKeyMtx != nil {
						sshKeyMtx.Lock()
					}
					gKeyMtx.Lock()
					fail := !makeCurrentSSHKey(ctx, idxSlug)
					if fail == true {
						gKeyMtx.Unlock()
						if sshKeyMtx != nil {
							sshKeyMtx.Unlock()
						}
						result[3] = fmt.Sprintf("%+v: %s\n", tsk, lib.ErrorStrings[5])
						return
					}
				}
				if ctx.DryRunSeconds > 0 {
					if ctx.DryRunSecondsRandom {
						time.Sleep(time.Duration(rand.Intn(ctx.DryRunSeconds*1000)) * time.Millisecond)
					} else {
						time.Sleep(time.Duration(ctx.DryRunSeconds) * time.Second)
					}
				}
				if keyAdded {
					gKeyMtx.Unlock()
					if sshKeyMtx != nil {
						sshKeyMtx.Unlock()
					}
				}
				if ctx.DryRunCodeRandom {
					rslt := rand.Intn(6)
					if rslt > 0 {
						result[3] = fmt.Sprintf("error: %d", rslt)
					}
				} else {
					if ctx.DryRunCode != 0 {
						result[3] = fmt.Sprintf("error: %d", ctx.DryRunCode)
					}
				}
				return
			}
			if keyAdded {
				if sshKeyMtx != nil {
					sshKeyMtx.Lock()
				}
				gKeyMtx.Lock()
				fail := !makeCurrentSSHKey(ctx, idxSlug)
				if fail == true {
					gKeyMtx.Unlock()
					if sshKeyMtx != nil {
						sshKeyMtx.Unlock()
					}
					result[3] = fmt.Sprintf("%+v: %s\n", tsk, lib.ErrorStrings[5])
					return
				}
			}
			if ctx.Debug > 0 {
				lib.Printf("External endpoint: %s\n", rcl)
			}
			var (
				err error
				str string
			)
			if !ctx.SkipP2O {
				str, err = lib.ExecCommand(ctx, commandLine, map[string]string{"PROJECT_SLUG": tsk.FxSlug}, nil)
			}
			if keyAdded {
				gKeyMtx.Unlock()
				if sshKeyMtx != nil {
					sshKeyMtx.Unlock()
				}
			}
			// str = strings.Replace(str, ctx.ElasticURL, lib.Redacted, -1)
			// p2o.py do not return error even if its backend execution fails
			// we need to capture STDERR and check if there was python exception there
			pyE := false
			strippedStr := str
			strLen := len(str)
			if strLen > ctx.StripErrorSize {
				strippedStr = str[0:ctx.StripErrorSize] + "\n(...)\n" + str[strLen-ctx.StripErrorSize:strLen]
			}
			if strings.Contains(str, lib.PyException) {
				pyE = true
				err = fmt.Errorf("%s", strippedStr)
			}
			if err == nil {
				if ctx.Debug > 0 {
					dtEnd := time.Now()
					lib.Printf("%+v: finished in %v, retries: %d\n", tsk, dtEnd.Sub(dtStart), retries)
				}
				break
			}
			if isTimeoutError(err) {
				dtEnd := time.Now()
				lib.Printf("Timeout error for %+v (took %v, tried %d times): %+v: %s\n", rcl, dtEnd.Sub(dtStart), retries, err, strippedStr)
				str += fmt.Sprintf(": %+v", err)
				result[3] = fmt.Sprintf("timeout: last retry took %v: %+v", dtEnd.Sub(dtStart), strippedStr)
				return
			}
			retries++
			if retries <= ctx.MaxRetry {
				time.Sleep(time.Duration(retries) * time.Second)
				continue
			}
			dtEnd := time.Now()
			if pyE {
				lib.Printf("Python exception for %+v (took %v, tried %d times): %+v\n", rcl, dtEnd.Sub(dtStart), retries, err)
			} else {
				lib.Printf("Error for %+v (took %v, tried %d times): %+v: %s\n", rcl, dtEnd.Sub(dtStart), retries, err, strippedStr)
				str += fmt.Sprintf(": %+v", err)
			}
			result[3] = fmt.Sprintf("last retry took %v: %+v", dtEnd.Sub(dtStart), strippedStr)
			return
		}
		return
	}
	processedEndpoints := 0
	allEndpoints := len(newTasks)
	allIndices := len(processedIndices)
	lastTime := time.Now()
	dtStart := lastTime
	prefix := "(***external***) "
	if thrN > 1 {
		lib.Printf("Now processing %d external indices enrichments (%d endpoints) using method MT%d version\n", allIndices, allEndpoints, thrN)
		gInfoExternal()
		ch := make(chan [4]string)
		nThreads := 0
		for _, tsk := range newTasks {
			go enrichExternal(ch, tsk)
			nThreads++
			if nThreads == thrN {
				ary := <-ch
				if ary[3] != "" {
					lib.Printf("WARNING: %s\n", strings.Join(ary[:], ":"))
				}
				nThreads--
				processedEndpoints++
				inf := prefix + strings.Join(ary[:3], ":")
				lib.ProgressInfo(processedEndpoints, allEndpoints, dtStart, &lastTime, time.Duration(1)*time.Minute, inf)
			}
		}
		for nThreads > 0 {
			ary := <-ch
			if ary[3] != "" {
				lib.Printf("WARNING: %s\n", strings.Join(ary[:], ":"))
			}
			nThreads--
			processedEndpoints++
			inf := prefix + strings.Join(ary[:3], ":") + ":" + fmt.Sprintf("wait for %d threads", nThreads)
			lib.ProgressInfo(processedEndpoints, allEndpoints, dtStart, &lastTime, time.Duration(1)*time.Minute, inf)
		}
	} else {
		lib.Printf("Now processing %d external indices enrichments (%d endpoints) using ST version\n", allIndices, allEndpoints)
		gInfoExternal()
		for _, tsk := range newTasks {
			ary := enrichExternal(nil, tsk)
			if ary[3] != "" {
				lib.Printf("WARNING: %s\n", strings.Join(ary[:], ":"))
			}
			processedEndpoints++
			inf := prefix + strings.Join(ary[:3], ":")
			lib.ProgressInfo(processedEndpoints, allEndpoints, dtStart, &lastTime, time.Duration(1)*time.Minute, inf)
		}
	}
	gInfoExternal()
	if !ctx.SkipP2O && !ctx.SkipEsData {
		updated := setLastRun(ctx, nil, lib.Bitergia, lib.External)
		if !updated {
			lib.Printf("failed to set last sync date for bitergia/external\n")
		}
	}
	en := time.Now()
	lib.Printf("Processed %d external indices (%d endpoints) took: %v\n", allIndices, allEndpoints, en.Sub(st))
}

func figureOutEndpoints(ctx *lib.Ctx, index, dataSource string) (endpoints, origins []string) {
	if ctx.DryRun && !ctx.DryRunAllowOrigins {
		return
	}
	//lib.Printf("figureOutEndpoints(%s, %s)\n", index, dataSource)
	//curl -H "Content-Type: application/json" URL/idx/_search -d'{"size":0,"aggs":{"origin":{"terms":{"field":"origin"}}}}'
	data := `{"size":0,"aggs":{"origin":{"terms":{"field":"origin","size":2147483647}}}}`
	payloadBytes := []byte(data)
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Get
	url := fmt.Sprintf("%s/%s/_search?size=1", ctx.ElasticURL, index)
	rurl := fmt.Sprintf("/%s/_search?size=1", index)
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("new request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("do request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s status:%d data:%s\n%s\n", method, rurl, resp.StatusCode, data, body)
		return
	}
	payload := lib.EsSearchResultPayload{}
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		body, err := ioutil.ReadAll(resp.Body)
		lib.Printf("JSON decode error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		lib.Printf("Body:%s\n", body)
		return
	}
	var m map[string]interface{}
	m, ok := payload.Aggregations.(map[string]interface{})
	if !ok {
		lib.Printf("Cannot read aggregations from %+v\n", payload)
		return
	}
	m, ok = m["origin"].(map[string]interface{})
	if !ok {
		lib.Printf("Cannot read origin from %+v\n", payload)
		return
	}
	buckets, ok := m["buckets"].([]interface{})
	if !ok {
		lib.Printf("Cannot read buckets from %+v\n", payload)
		return
	}
	for _, bucket := range buckets {
		b, ok := bucket.(map[string]interface{})
		if !ok {
			lib.Printf("Cannot read bucket from %+v\n", bucket)
			return
		}
		key, ok := b["key"]
		if !ok {
			lib.Printf("Origin %v do net have a key\n", b)
		}
		strKey, ok := key.(string)
		if !ok {
			lib.Printf("Origin key %v is not a string\n", key)
		}
		origins = append(origins, strKey)
	}
	if len(origins) == 0 {
		lib.Printf("WARNING: No origins found for %s\n", index)
	}
	ary := strings.Split(dataSource, "/")
	if len(ary) > 1 {
		dataSource = ary[0]
	}
	switch dataSource {
	case lib.Git, lib.Jira, lib.Bugzilla, lib.BugzillaRest, lib.Jenkins, lib.Gerrit, lib.Pipermail, lib.Confluence, lib.GitHub, lib.Discourse, lib.RocketChat:
		endpoints = origins
	case lib.MeetUp:
		// FIXME: meetup we don't really have meetup config, the only one we have in SDS is disabled for CNCF/Prometheus 'SF-Prometheus-Meetup-Group'
		// But there is one Bitergia index for meetup which has origin: 'https://meetup.com/' (perceval docs just say 'group'), so this may or may not work for meetup
		lib.Printf("WARNING: dataSource: %s, origins: %+v, meetup transformation 1:1 from origins to endpoint may be wrong\n", dataSource, origins)
		endpoints = origins
	case lib.Slack, lib.GroupsIO:
		// Slack: https://slack.com/C0417QHH7 --> C0417QHH7
		// Groups.io: https://groups.io/g/OPNFV+opnfv-tech-discuss -> OPNFV+opnfv-tech-discuss
		for _, origin := range origins {
			origin = strings.TrimSpace(origin)
			if strings.HasSuffix(origin, "/") {
				origin = origin[:len(origin)-1]
			}
			ary := strings.Split(origin, "/")
			origin = ary[len(ary)-1]
			endpoints = append(endpoints, origin)
		}
	case lib.DockerHub:
		// DockerHub: https://hub.docker.com/hyperledger/explorer-db -> "hyperledger explorer-db"
		for _, origin := range origins {
			origin = strings.TrimSpace(origin)
			if strings.HasSuffix(origin, "/") {
				origin = origin[:len(origin)-1]
			}
			ary := strings.Split(strings.TrimSpace(origin), "/")
			lAry := len(ary)
			if lAry >= 2 {
				endpoints = append(endpoints, ary[lAry-2]+" "+ary[lAry-1])
			}
		}
	default:
		lib.Printf("ERROR(not fatal): dataSource: %s, origins: %+v, don't know how to transform them to endpoints\n", dataSource, origins)
	}
	return
}

func figureOutDatasourceFromIndexName(index string) (dataSource string) {
	index = strings.ToLower(index)
	known := []string{
		lib.Git,
		lib.GitHub,
		lib.Confluence,
		lib.Gerrit,
		lib.Jira,
		lib.Slack,
		lib.GroupsIO,
		lib.Pipermail,
		lib.Discourse,
		lib.Jenkins,
		lib.DockerHub,
		lib.Bugzilla,
		lib.BugzillaRest,
		lib.MeetUp,
		lib.RocketChat,
	}
	sort.SliceStable(known, func(i, j int) bool {
		return len(known[i]) > len(known[j])
	})
	for _, ds := range known {
		if strings.Contains(index, strings.ToLower(ds)) {
			return ds
		}
	}
	return
}

func checkForSharedEndpoints(pfixtures *[]lib.Fixture) {
	fixtures := *pfixtures
	eps := make(map[[3]string][]string)
	for _, fixture := range fixtures {
		slug := fixture.Native.Slug
		for _, ds := range fixture.DataSources {
			cfgs := []string{}
			for _, cfg := range ds.Config {
				cfgs = append(cfgs, cfg.String())
			}
			sort.Strings(cfgs)
			cfg := strings.Join(cfgs, ",")
			cfg = strings.Replace(cfg, " ", "", -1)
			cfg = strings.Replace(cfg, "\t", "", -1)
			for _, ep := range ds.Endpoints {
				key := [3]string{ds.Slug, ep.Name, cfg}
				slugs := eps[key]
				slugs = append(slugs, slug)
				eps[key] = slugs
			}
		}
	}
	for ep, slugs := range eps {
		if len(slugs) == 1 {
			continue
		}
		lib.Printf("NOTICE: Endpoint (%s,%s) that can be split into shared, used in %+v\n", ep[0], ep[1], slugs)
	}
}

func giantLock(ctx *lib.Ctx, mtx string) {
	if ctx.Debug > 0 {
		lib.Printf("giantLock(%s)\n", mtx)
	}
	mtxIndex := lib.SDSMtx
	esQuery := fmt.Sprintf("mtx:\"%s\"", mtx)
	_, ok, found := searchByQuery(ctx, mtxIndex, esQuery)
	if !ok {
		lib.Fatalf("cannot lock ES mtx: %s", mtx)
	}
	if found {
		if ctx.Debug >= 0 {
			lib.Printf("%s is already locked\n", mtx)
		}
		return
	}
	if ctx.DryRun {
		if !ctx.DryRunAllowMtx {
			lib.Printf("Would lock ES mtx %s\n", mtx)
			return
		}
	}
	data := lib.EsMtxPayload{Mtx: mtx, Dt: time.Now()}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		lib.Fatalf("json marshall error: %+v for mtx: %s, data: %+v", err, mtx, data)
	}
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Post
	url := fmt.Sprintf("%s/%s/_doc?refresh=wait_for", ctx.ElasticURL, mtxIndex)
	rurl := fmt.Sprintf("/%s/_doc?refresh=wait_for", mtxIndex)
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Fatalf("new request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Fatalf("do request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 201 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Fatalf("ReadAll request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
		}
		lib.Fatalf("Method:%s url:%s status:%d data:%+v\n%s", method, rurl, resp.StatusCode, data, body)
	}
	if ctx.Debug >= 0 {
		lib.Printf("%s locked\n", mtx)
	}
}

func giantUnlock(ctx *lib.Ctx, mtx string) {
	if ctx.Debug > 0 {
		lib.Printf("giantUnlock(%s)\n", mtx)
	}
	mtxIndex := lib.SDSMtx
	esQuery := fmt.Sprintf("mtx:\"%s\"", mtx)
	_, ok, found := searchByQuery(ctx, mtxIndex, esQuery)
	if !ok {
		lib.Fatalf("cannot lock ES mtx: %s", mtx)
	}
	if !found {
		if ctx.Debug >= 0 {
			lib.Printf("%s wasn't locked\n", mtx)
		}
		return
	}
	if ctx.DryRun {
		if !ctx.DryRunAllowMtx {
			lib.Printf("Would unlock ES mtx %s\n", mtx)
			return
		}
	}
	trials := 0
	for {
		deleted := deleteByQuery(ctx, mtxIndex, esQuery)
		if deleted {
			if trials > 0 {
				lib.Printf("Unlocked %s mutex after %d/%d trials\n", mtx, trials, ctx.MaxDeleteTrials)
			}
			break
		}
		trials++
		if trials == ctx.MaxDeleteTrials {
			lib.Fatalf("Failed to unlock %s mutex, tried %d times\n", mtx, ctx.MaxDeleteTrials)
		}
		time.Sleep(time.Duration(10*trials) * time.Millisecond)
	}
	if ctx.Debug >= 0 {
		lib.Printf("%s unlocked\n", mtx)
	}
}

func giantWait(ctx *lib.Ctx, mtx, waitForState string) {
	if ctx.Debug > 0 {
		lib.Printf("giantWait(%s,%s)\n", mtx, waitForState)
	}
	mtxIndex := lib.SDSMtx
	esQuery := fmt.Sprintf("mtx:\"%s\"", mtx)
	n := 0
	for {
		_, ok, found := searchByQuery(ctx, mtxIndex, esQuery)
		if !ok {
			lib.Fatalf("cannot wait for ES mtx: %s to be %s", mtx, waitForState)
		}
		if ctx.MaxMtxWait > 0 && n > ctx.MaxMtxWait {
			if ctx.MaxMtxWaitFatal {
				lib.Fatalf("waited %d seconds for %s to be %s, exceeded %ds, this is fatal", n, mtx, waitForState, ctx.MaxMtxWait)
			} else {
				lib.Printf("WARNING: Waited %d seconds for %s to be %s, exceeded %ds, will proceed after next 30s\n", n, mtx, waitForState, ctx.MaxMtxWait)
				time.Sleep(time.Duration(30) * time.Second)
				if waitForState == lib.Unlocked {
					giantUnlock(ctx, mtx)
				}
				return
			}
		}
		if (found && waitForState == lib.Locked) || (!found && waitForState == lib.Unlocked) {
			if ctx.Debug >= 0 || n > 0 {
				lib.Printf("Waited %d seconds for %s to be %s...\n", n, mtx, waitForState)
			}
			return
		}
		// Wait 1s for next retry
		if (ctx.Debug > 0 || n >= 30) && n%30 == 0 {
			lib.Printf("Waiting for %s to be %s (already waited %ds)...\n", mtx, waitForState, n)
		}
		time.Sleep(time.Duration(1000) * time.Millisecond)
		n++
	}
}

func renameIndex(ctx *lib.Ctx, from, to string) {
	// Disable write to source index
	indexBlocksWrite := true
	data := lib.EsIndexSettingsPayload{Settings: lib.EsIndexSettings{IndexBlocksWrite: &indexBlocksWrite}}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		lib.Printf("JSON marshall error: %+v for snapshot rename %s to %s: %+v\n", err, from, to, data)
		return
	}
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Put
	url := fmt.Sprintf("%s/%s/_settings", ctx.ElasticURL, from)
	rurl := fmt.Sprintf("%s/_settings", from)
	if ctx.DryRun {
		if !ctx.DryRunAllowRename {
			lib.Printf("Would execute: method:%s url:%s data:%+v\n", method, os.ExpandEnv(rurl), data)
			return
		}
		lib.Printf("Dry run allowed rename index %s to %s\n", from, to)
	}
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s data:%+v\n", err, method, rurl, data)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s data:%+v\n", err, method, rurl, data)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s data:%+v\n", err, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s status:%d data:%+v\n%s\n", method, rurl, resp.StatusCode, data, body)
		return
	}
	// Clone source to dest
	data = lib.EsIndexSettingsPayload{Settings: lib.EsIndexSettings{IndexBlocksWrite: nil}}
	payloadBytes, err = json.Marshal(data)
	if err != nil {
		lib.Printf("JSON marshall error: %+v for snapshot rename %s to %s: %+v\n", err, from, to, data)
		return
	}
	payloadBody = bytes.NewReader(payloadBytes)
	method = lib.Put
	url = fmt.Sprintf("%s/%s/_clone/%s", ctx.ElasticURL, from, to)
	rurl = fmt.Sprintf("/%s/_clone/%s", from, to)
	req, err = http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s data:%+v\n", err, method, rurl, data)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s data:%+v\n", err, method, rurl, data)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s data:%+v\n", err, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s status:%d data:%+v\n%s\n", method, rurl, resp.StatusCode, data, body)
		return
	}
	// Wait for at least yeallow state on dest index
	method = lib.Get
	url = fmt.Sprintf("%s/_cluster/health/%s?wait_for_status=yellow&timeout=180s", ctx.ElasticURL, to)
	rurl = fmt.Sprintf("/_cluster/health/%s?wait_for_status=yellow&timeout=180s", to)
	req, err = http.NewRequest(method, os.ExpandEnv(url), nil)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		lib.Printf("Method:%s url:%s status:%d \n%s\n", method, rurl, resp.StatusCode, body)
		return
	}
	// Delete source index (it will become an alias to source (with some other additional index in the same alias)
	// if configured that way in fixtures
	method = lib.Delete
	url = fmt.Sprintf("%s/%s", ctx.ElasticURL, from)
	rurl = fmt.Sprintf("/%s", from)
	req, err = http.NewRequest(method, os.ExpandEnv(url), nil)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		lib.Printf("Method:%s url:%s status:%d\n%s\n", method, rurl, resp.StatusCode, body)
		return
	}
	lib.Printf("Renamed %s to %s\n", from, to)
}

// processIndexes - dropping unused indexes, renaming indexes that require this ('index_suffix' option), info about missing indexes
func processIndexes(ctx *lib.Ctx, pfixtures *[]lib.Fixture) (didRenames bool) {
	fixtures := *pfixtures
	if ctx.NodeIdx > 0 {
		lib.Printf("Skipping processing indexes, this only runs on 1st node\n")
		return
	}
	// after dropping (and possibly renaming) indices we're unlocking "rename" ES mutex
	if ctx.NodeNum > 1 {
		defer func() {
			lib.Printf("Waiting %ds for other node(s) to settle up\n", ctx.NodeSettleTime*ctx.NodeNum)
			time.Sleep(time.Duration(ctx.NodeSettleTime*ctx.NodeNum) * time.Second)
			for i := ctx.NodeNum - 1; i > 0; i-- {
				mtx := fmt.Sprintf("rename-node-%d", i)
				lib.Printf("Master wait for %s to be locked by node\n", mtx)
				giantWait(ctx, mtx, lib.Locked)
				lib.Printf("Master unlocking %s\n", mtx)
				giantUnlock(ctx, mtx)
				lib.Printf("Master unlocked %s\n", mtx)
			}
			lib.Printf("Master processing mutexes finished\n")
		}()
	}
	should := make(map[string]struct{})
	fromFull := make(map[string]string)
	toFull := make(map[string]string)
	for _, fixture := range fixtures {
		slug := fixture.Slug
		slug = strings.Replace(slug, "/", "-", -1)
		for _, ds := range fixture.DataSources {
			idxSlug := "sds-" + slug + "-" + ds.FullSlug
			idxSlug = strings.Replace(idxSlug, "/", "-", -1)
			should[idxSlug] = struct{}{}
			should[idxSlug+"-raw"] = struct{}{}
			if ds.Slug != ds.FullSlug {
				idx := "sds-" + slug + "-" + ds.Slug
				idx = strings.Replace(idx, "/", "-", -1)
				fromFull[idxSlug] = idx
				fromFull[idxSlug+"-raw"] = idx + "-raw"
				toFull[idx] = idxSlug
				toFull[idx+"-raw"] = idxSlug + "-raw"
			}
		}
	}
	method := lib.Get
	url := fmt.Sprintf("%s/_cat/indices?format=json", ctx.ElasticURL)
	rurl := "/_cat/indices?format=json"
	req, err := http.NewRequest(method, os.ExpandEnv(url), nil)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		lib.Printf("Method:%s url:%s status:%d\n%s\n", method, rurl, resp.StatusCode, body)
		return
	}
	indices := []lib.EsIndex{}
	err = json.NewDecoder(resp.Body).Decode(&indices)
	if err != nil {
		lib.Printf("JSON decode error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	got := make(map[string]struct{})
	for _, index := range indices {
		sIndex := index.Index
		if !strings.HasPrefix(sIndex, "sds-") {
			continue
		}
		got[sIndex] = struct{}{}
	}
	missing := []string{}
	extra := []string{}
	rename := make(map[string]string)
	for fullIndex := range should {
		_, ok := got[fullIndex]
		if !ok {
			index := fromFull[fullIndex]
			_, ok := got[index]
			if ok {
				rename[index] = fullIndex
			} else {
				missing = append(missing, fullIndex)
			}
		}
	}
	for index := range got {
		_, ok := should[index]
		if !ok {
			fullIndex, ok := rename[index]
			if !ok {
				extra = append(extra, index)
			} else {
				lib.Printf("NOTICE: Index %s will be renamed to %s\n", index, fullIndex)
			}
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 {
		lib.Printf("NOTICE: Missing indices (%d): %s\n", len(missing), strings.Join(missing, ", "))
	}
	if len(rename) > 0 {
		lib.Printf("Indices to rename:\n")
		for from, to := range rename {
			lib.Printf("%s -> %s\n", from, to)
		}
		renameFunc := func(ch chan struct{}, ctx *lib.Ctx, from, to string) {
			defer func() {
				if ch != nil {
					ch <- struct{}{}
				}
			}()
			renameIndex(ctx, from, to)
		}
		thrN := lib.GetThreadsNum(ctx)
		if thrN > 1 {
			lib.Printf("Now processing %d renames using method MT%d version\n", len(rename), thrN)
			ch := make(chan struct{})
			nThreads := 0
			for from, to := range rename {
				go renameFunc(ch, ctx, from, to)
				nThreads++
				if nThreads == thrN {
					<-ch
					nThreads--
				}
			}
			for nThreads > 0 {
				<-ch
				nThreads--
			}
		} else {
			lib.Printf("Now processing %d renames using ST version\n", len(rename))
			for from, to := range rename {
				renameFunc(nil, ctx, from, to)
			}
		}
		didRenames = true
	} else {
		lib.Printf("No indices to rename\n")
	}
	if len(extra) == 0 {
		lib.Printf("No indices to drop, environment clean\n")
		return
	}
	if partialRun(ctx) {
		return
	}
	lib.Printf("Indices to delete (%d): %s\n", len(extra), strings.Join(extra, ", "))
	method = lib.Delete
	extras := []string{}
	curr := ""
	maxSize := 0x800
	for _, ex := range extra {
		ex = lib.SafeString(ex)
		if curr == "" {
			curr = ex
		} else {
			if len(curr+","+ex) < maxSize {
				curr += "," + ex
			} else {
				extras = append(extras, curr)
				curr = ex
			}
		}
	}
	extras = append(extras, curr)
	for _, indices := range extras {
		lib.Printf("Deleting indices: %s\n", indices)
		url = fmt.Sprintf("%s/%s", ctx.ElasticURL, indices)
		rurl = fmt.Sprintf("/%s", indices)
		if ctx.DryRun {
			lib.Printf("Would execute: method:%s url:%s\n", method, os.ExpandEnv(rurl))
			return
		}
		req, err = http.NewRequest(method, os.ExpandEnv(url), nil)
		if err != nil {
			lib.Printf("New request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			lib.Printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		if resp.StatusCode != 200 {
			body, err := ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				lib.Printf("ReadAll request error: %+v for %s url: %s\n", err, method, rurl)
				return
			}
			lib.Printf("Method:%s url:%s status:%d\n%s\n", method, rurl, resp.StatusCode, body)
			return
		}
		_ = resp.Body.Close()
		lib.Printf("%d indices dropped\n", len(strings.Split(indices, ",")))
	}
	return
}

func dropUnusedAliases(ctx *lib.Ctx, pfixtures *[]lib.Fixture) {
	fixtures := *pfixtures
	if ctx.NodeIdx > 0 {
		lib.Printf("Skipping dropping unused aliases, this only runs on 1st node\n")
		return
	}
	should := make(map[string]struct{})
	for _, fixture := range fixtures {
		for _, alias := range fixture.Aliases {
			for _, to := range alias.To {
				should[strings.Replace(to, "/", "-", -1)] = struct{}{}
			}
		}
	}
	method := lib.Get
	url := fmt.Sprintf("%s/_cat/aliases?format=json", ctx.ElasticURL)
	rurl := "/_cat/aliases?format=json"
	req, err := http.NewRequest(method, os.ExpandEnv(url), nil)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		lib.Printf("Method:%s url:%s status:%d\n%s\n", method, rurl, resp.StatusCode, body)
		return
	}
	aliases := []lib.EsAlias{}
	err = json.NewDecoder(resp.Body).Decode(&aliases)
	if err != nil {
		lib.Printf("JSON decode error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	got := make(map[string]struct{})
	for _, alias := range aliases {
		sAlias := alias.Alias
		if !strings.HasPrefix(sAlias, "sds-") {
			continue
		}
		got[sAlias] = struct{}{}
	}
	missing := []string{}
	extra := []string{}
	for alias := range should {
		_, ok := got[alias]
		if !ok {
			missing = append(missing, alias)
		}
	}
	for alias := range got {
		_, ok := should[alias]
		if !ok {
			extra = append(extra, alias)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 {
		lib.Printf("NOTICE: Missing aliases %d: %s\n", len(missing), strings.Join(missing, ", "))
	}
	if len(extra) == 0 {
		lib.Printf("No aliases to drop, environment clean\n")
		return
	}
	if partialRun(ctx) {
		return
	}
	lib.Printf("Aliases to delete (%d): %s\n", len(extra), strings.Join(extra, ", "))
	method = lib.Delete
	extras := []string{}
	curr := ""
	maxSize := 0x800
	for _, ex := range extra {
		ex = lib.SafeString(ex)
		if curr == "" {
			curr = ex
		} else {
			if len(curr+","+ex) < maxSize {
				curr += "," + ex
			} else {
				extras = append(extras, curr)
				curr = ex
			}
		}
	}
	extras = append(extras, curr)
	for _, aliases := range extras {
		lib.Printf("Deleting aliases: %s\n", aliases)
		url = fmt.Sprintf("%s/_all/_alias/%s", ctx.ElasticURL, aliases)
		rurl = fmt.Sprintf("/_all/_alias/%s", aliases)
		if ctx.DryRun {
			lib.Printf("Would execute: method:%s url:%s\n", method, os.ExpandEnv(rurl))
			return
		}
		req, err = http.NewRequest(method, os.ExpandEnv(url), nil)
		if err != nil {
			lib.Printf("New request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			lib.Printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		if resp.StatusCode != 200 {
			body, err := ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				lib.Printf("ReadAll request error: %+v for %s url: %s\n", err, method, rurl)
				return
			}
			lib.Printf("Method:%s url:%s status:%d\n%s\n", method, rurl, resp.StatusCode, body)
			return
		}
		_ = resp.Body.Close()
		lib.Printf("%d aliases dropped\n", len(strings.Split(aliases, ",")))
	}
}

func processAlias(ch chan struct{}, ctx *lib.Ctx, pair [2]string, method string) {
	defer func() {
		if ch != nil {
			ch <- struct{}{}
		}
	}()
	var (
		url  string
		rurl string
	)
	if method == lib.Delete {
		if partialRun(ctx) {
			return
		}
		url = fmt.Sprintf("%s/_all/_alias/%s", ctx.ElasticURL, pair[1])
		rurl = fmt.Sprintf("/_all/_alias/%s", pair[1])
	} else {
		url = fmt.Sprintf("%s/%s/_alias/%s", ctx.ElasticURL, pair[0], pair[1])
		rurl = fmt.Sprintf("/%s/_alias/%s", pair[0], pair[1])
	}
	if ctx.DryRun {
		lib.Printf("DryRun: Method:%s url:%s\n", method, rurl)
		return
	}
	req, err := http.NewRequest(method, os.ExpandEnv(url), nil)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		lib.Printf("Method:%s url:%s status:%d\n%s\n", method, rurl, resp.StatusCode, body)
	}
	if ctx.Debug > 0 {
		lib.Printf("Success alias url: %s\n", rurl)
	}
}

func processAliases(ctx *lib.Ctx, pFixtures *[]lib.Fixture, method string) {
	st := time.Now()
	fixtures := *pFixtures
	pairs := [][2]string{}
	tom := make(map[string]struct{})
	for _, fixture := range fixtures {
		for _, alias := range fixture.Aliases {
			for _, to := range alias.To {
				_, ok := tom[to]
				if ok && method == lib.Delete {
					continue
				}
				pairs = append(pairs, [2]string{alias.From, to})
				tom[to] = struct{}{}
			}
		}
	}
	if ctx.Debug > 0 {
		lib.Printf("Aliases:\n%+v\n", pairs)
	}
	// Get number of CPUs available
	thrN := lib.GetThreadsNum(ctx)
	if thrN > 1 {
		lib.Printf("Now processing %d aliases using method %s MT%d version\n", len(pairs), method, thrN)
		ch := make(chan struct{})
		nThreads := 0
		for _, pair := range pairs {
			go processAlias(ch, ctx, pair, method)
			nThreads++
			if nThreads == thrN {
				<-ch
				nThreads--
			}
		}
		for nThreads > 0 {
			<-ch
			nThreads--
		}
	} else {
		lib.Printf("Now processing %d aliases using method %s ST version\n", len(pairs), method)
		for _, pair := range pairs {
			processAlias(nil, ctx, pair, method)
		}
	}
	en := time.Now()
	lib.Printf("Processed %d aliases using method %s took: %v\n", len(pairs), method, en.Sub(st))
}

func saveCSVInternal(ctx *lib.Ctx, tasks []lib.Task, when string, redacted bool) {
	var writer *csv.Writer
	csvFile := fmt.Sprintf("%s_%s_%d_%d.csv", ctx.CSVPrefix, when, ctx.NodeIdx, ctx.NodeNum)
	oFile, err := os.Create(csvFile)
	if err != nil {
		lib.Printf("CSV create error: %+v\n", err)
		return
	}
	defer func() { _ = oFile.Close() }()
	writer = csv.NewWriter(oFile)
	defer writer.Flush()
	hdr := lib.CSVHeader()
	err = writer.Write(hdr)
	if err != nil {
		lib.Printf("CSV write header error: %+v\n", err)
		return
	}
	gCSVMtx.Lock()
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].FxSlug == tasks[j].FxSlug {
			if tasks[i].DsFullSlug == tasks[j].DsFullSlug {
				return tasks[i].Endpoint < tasks[j].Endpoint
			}
			return tasks[i].DsFullSlug < tasks[j].DsFullSlug
		}
		return tasks[i].FxSlug < tasks[j].FxSlug
	})
	gCSVMtx.Unlock()
	for _, task := range tasks {
		f := task.ToCSV
		if !redacted {
			f = task.ToCSVNotRedacted
		}
		err = writer.Write(f())
		if err != nil {
			lib.Printf("CSV write row (%+v) error: %+v\n", task, err)
			return
		}
	}
	lib.Printf("CSV file %s written\n", csvFile)
}

func saveCSV(ctx *lib.Ctx, tasks []lib.Task, when string) {
	ch := make(chan struct{})
	go func() {
		saveCSVInternal(ctx, tasks, "redacted_"+when, true)
		ch <- struct{}{}
	}()
	saveCSVInternal(ctx, tasks, when, false)
	<-ch
}

func processTasks(ctx *lib.Ctx, ptasks *[]lib.Task, dss []string) error {
	tasks := *ptasks
	saveCSV(ctx, tasks, "init")
	thrN := lib.GetThreadsNum(ctx)
	tMtx := lib.TaskMtx{}
	if thrN > 1 {
		tMtx.SSHKeyMtx = &sync.Mutex{}
		tMtx.TaskOrderMtx = &sync.Mutex{}
		tMtx.SyncInfoMtx = &sync.Mutex{}
		tMtx.SyncFreqMtx = &sync.RWMutex{}
	}
	failed := [][2]int{}
	processed := 0
	all := len(tasks)
	mul := 1
	if !ctx.SkipData && !ctx.SkipAffs {
		mul = 2
		all *= mul
		if thrN > 1 {
			tMtx.OrderMtx = make(map[int]*sync.Mutex)
			for idx := range tasks {
				tmtx := &sync.Mutex{}
				tmtx.Lock()
				tMtx.OrderMtx[idx] = tmtx
			}
		}
	}
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
	mtx := &sync.RWMutex{}
	info := func(when string) {
		mtx.RLock()
		defer func() {
			mtx.RUnlock()
		}()
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
			sort.SliceStable(dursAry, func(i, j int) bool {
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
				sort.SliceStable(dursAry, func(i, j int) bool {
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
			allDs := data[0] * mul
			failedDs := data[1]
			processedDs := data[2]
			if failedDs > 0 || processedDs != allDs {
				lib.Printf("Data source: %s, Processed %d/%d (%.2f%%), failed: %d (%.2f%%)\n", ds, processedDs, allDs, (float64(processedDs)*100.0)/float64(allDs), failedDs, (float64(failedDs)*100.0)/float64(allDs))
				out = true
			}
		}
		for _, fx := range fxs {
			data := byFx[fx]
			allFx := data[0] * mul
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
		saveCSV(ctx, tasks, when)
	}
	go func() {
		for {
			sig := <-sigs
			info("signal")
			if gInfoExternal != nil {
				gInfoExternal()
			}
			if sig == syscall.SIGINT {
				lib.Printf("Exiting due to SIGINT\n")
				os.Exit(1)
			} else if sig == syscall.SIGALRM {
				lib.Printf("Timeout after %d seconds\n", ctx.TimeoutSeconds)
				lib.Printf("Ensuring aliases are created\n")
				gAliasesFunc()
				lib.Printf("Aliases processed after timeout\n")
				os.Exit(2)
			}
		}
	}()
	skipAry := strings.Split(ctx.SkipReenrich, ",")
	skipDS := make(map[string]struct{})
	for _, skipV := range skipAry {
		skipDS[strings.TrimSpace(skipV)] = struct{}{}
	}
	lastTime := time.Now()
	dtStart := lastTime
	modes := []bool{false, true}
	modesStr := []string{"data", "affs"}
	nThreads := 0
	skippedTasks := 0
	ch := make(chan lib.TaskResult)
	for modeIdx, affs := range modes {
		stTime := time.Now()
		lib.Printf("Affiliations mode: %+v\n", affs)
		if affs == false && ctx.SkipData {
			lib.Printf("Incremental data sync skipped\n")
			continue
		}
		if affs == true && ctx.SkipAffs {
			lib.Printf("Historical data affiliations sync skipped\n")
			continue
		}
		if thrN > 1 {
			if ctx.Debug >= 0 {
				lib.Printf("Processing %d tasks using MT%d version (affiliations mode: %+v)\n", len(tasks), thrN, affs)
			}
			for idx, task := range tasks {
				if taskFilteredOut(ctx, &task) {
					skippedTasks++
					processed++
					continue
				}
				_, skipped := skipDS[task.DsSlug]
				if affs && skipped {
					skippedTasks++
					processed++
					continue
				}
				mtx.Lock()
				processing[idx] = struct{}{}
				startTimes[idx] = time.Now()
				mtx.Unlock()
				go processTask(ch, ctx, idx, task, affs, &tMtx)
				nThreads++
				if nThreads == thrN {
					result := <-ch
					res := result.Code
					taffs := result.Affs
					tIdx := res[0]
					tasks[tIdx].CommandLine = result.CommandLine
					tasks[tIdx].RedactedCommandLine = result.RedactedCommandLine
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
					if res[1] > 0 {
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
					extraInf := tasks[tIdx].ShortString()
					if skippedTasks > 0 {
						extraInf += fmt.Sprintf(" (%d skipped)", skippedTasks)
					}
					lib.ProgressInfo(processed, all, dtStart, &lastTime, time.Duration(1)*time.Minute, extraInf)
					if res[1] < 0 {
						continue
					}
					if !taffs && tMtx.OrderMtx != nil {
						tMtx.TaskOrderMtx.Lock()
						tmtx, ok := tMtx.OrderMtx[tIdx]
						if !ok {
							tMtx.TaskOrderMtx.Unlock()
							lib.Fatalf("per task mutex map is defined, but no mutex for tIdx: %d", tIdx)
						}
						tmtx.Unlock()
						tMtx.OrderMtx[tIdx] = tmtx
						// lib.Printf("mtx %d unlocked (data task finished)\n", tIdx)
						tMtx.TaskOrderMtx.Unlock()
					}
					setSyncInfo(ctx, &tMtx, &result, false)
					if result.Err == nil && len(result.Projects) > 0 {
						setProject(ctx, result.Index, result.Projects)
					}
				}
			}
		} else {
			if ctx.Debug >= 0 {
				lib.Printf("Processing %d tasks using ST version\n", len(tasks))
			}
			for idx, task := range tasks {
				if taskFilteredOut(ctx, &task) {
					skippedTasks++
					processed++
					continue
				}
				_, skipped := skipDS[task.DsSlug]
				if affs && skipped {
					skippedTasks++
					processed++
					continue
				}
				processing[idx] = struct{}{}
				result := processTask(nil, ctx, idx, task, affs, &tMtx)
				res := result.Code
				tIdx := res[0]
				tasks[tIdx].CommandLine = result.CommandLine
				tasks[tIdx].RedactedCommandLine = result.RedactedCommandLine
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
				if res[1] > 0 {
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
				extraInf := tasks[tIdx].ShortString()
				if skippedTasks > 0 {
					extraInf += fmt.Sprintf(" (%d skipped)", skippedTasks)
				}
				lib.ProgressInfo(processed, all, dtStart, &lastTime, time.Duration(1)*time.Minute, extraInf)
				if res[1] < 0 {
					continue
				}
				setSyncInfo(ctx, nil, &result, false)
				if result.Err == nil && len(result.Projects) > 0 {
					setProject(ctx, result.Index, result.Projects)
				}
			}
		}
		enTime := time.Now()
		lib.Printf("Pass (affiliations: %+v) finished in %v (excluding pending %d threads)\n", affs, enTime.Sub(stTime), nThreads)
		info(modesStr[modeIdx])
	}
	if thrN > 1 {
		stTime := time.Now()
		lib.Printf("Final %d threads join\n", nThreads)
		earlyAliasesProcessed := false
		for nThreads > 0 {
			if !earlyAliasesProcessed {
				gAliasesFunc()
				earlyAliasesProcessed = true
			}
			result := <-ch
			res := result.Code
			taffs := result.Affs
			tIdx := res[0]
			tasks[tIdx].CommandLine = result.CommandLine
			tasks[tIdx].RedactedCommandLine = result.RedactedCommandLine
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
			if res[1] > 0 {
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
			extraInf := tasks[tIdx].ShortString()
			if skippedTasks > 0 {
				extraInf += fmt.Sprintf(" (%d skipped)", skippedTasks)
			}
			lib.ProgressInfo(processed, all, dtStart, &lastTime, time.Duration(1)*time.Minute, extraInf)
			if res[1] < 0 {
				continue
			}
			if !taffs && tMtx.OrderMtx != nil {
				tMtx.TaskOrderMtx.Lock()
				tmtx, ok := tMtx.OrderMtx[tIdx]
				if !ok {
					tMtx.TaskOrderMtx.Unlock()
					lib.Fatalf("per task mutex map is defined, but no mutex for tIdx (final threads join): %d", tIdx)
				}
				tmtx.Unlock()
				tMtx.OrderMtx[tIdx] = tmtx
				//lib.Printf("mtx %d unlocked (data task finished in final join)\n", tIdx)
				tMtx.TaskOrderMtx.Unlock()
			}
			setSyncInfo(ctx, &tMtx, &result, false)
			if result.Err == nil && len(result.Projects) > 0 {
				setProject(ctx, result.Index, result.Projects)
			}
		}
		enTime := time.Now()
		lib.Printf("Pass (threads join) finished in %v\n", enTime.Sub(stTime))
	}
	info("final")
	lib.Printf("Skipped tasks: %d\n", skippedTasks)
	return nil
}

func makeCurrentSSHKey(ctx *lib.Ctx, idxSlug string) bool {
	if ctx.DryRun && !ctx.DryRunAllowSSH {
		return true
	}
	home := os.Getenv("HOME")
	dir := home + "/.ssh"
	fnTo := dir + "/id_rsa"
	fnFrom := fnTo + "-" + idxSlug
	cmd := exec.Command("cp", fnFrom, fnTo)
	err := cmd.Run()
	if err != nil {
		lib.Printf("Failed command cp %s %s: %+v\n", fnFrom, fnTo, err)
		return false
	}
	if ctx.Debug >= 0 {
		lib.Printf("Set current SSH Key: %s\n", fnFrom)
	}
	return true
}

func addSSHPrivKey(ctx *lib.Ctx, key, idxSlug string) bool {
	if ctx.DryRun && !ctx.DryRunAllowSSH {
		return true
	}
	home := os.Getenv("HOME")
	dir := home + "/.ssh"
	cmd := exec.Command("mkdir", dir)
	_ = cmd.Run()
	fn := dir + "/id_rsa-" + idxSlug
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
		lib.Git:          {},
		lib.Confluence:   {},
		lib.Gerrit:       {},
		lib.Jira:         {},
		lib.Slack:        {},
		lib.GroupsIO:     {},
		lib.Pipermail:    {},
		lib.Discourse:    {},
		lib.Jenkins:      {},
		lib.DockerHub:    {},
		lib.Bugzilla:     {},
		lib.BugzillaRest: {},
		lib.MeetUp:       {},
		lib.RocketChat:   {},
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
			if lAry < 2 {
				return
			}
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
	} else if ds == lib.DockerHub || ds == lib.RocketChat {
		if strings.Contains(endpoint, " ") {
			ary := strings.Split(endpoint, " ")
			nAry := []string{}
			for _, e := range ary {
				if e != "" {
					nAry = append(nAry, e)
				}
			}
			lAry := len(nAry)
			if lAry < 2 {
				return
			}
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

func mergeTokens(ctx *lib.Ctx, inf string, tokens1, tokens2 []string) (tokens []string) {
	m := make(map[string]struct{})
	for _, token := range tokens1 {
		m[token] = struct{}{}
	}
	for _, token := range tokens2 {
		m[token] = struct{}{}
	}
	for token := range m {
		tokens = append(tokens, token)
	}
	if ctx.Debug > 0 {
		lib.Printf("%s: GitHub OAuth tokens merge: %d, %d --> %d\n", inf, len(tokens1), len(tokens2), len(tokens))
	}
	return
}

// massageConfig - this function makes sure that given config options are valid for a given data source
// it also ensures some essential options are enabled and eventually reformats config
func massageConfig(ctx *lib.Ctx, config *[]lib.Config, ds, idxSlug string) (c []lib.MultiConfig, fail, keyAdded bool) {
	m := make(map[string]struct{})
	if ds == lib.GitHub {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				redactedValue = lib.Redacted
				lib.AddRedacted(value, true)
			}
			m[name] = struct{}{}
			if name == lib.APIToken {
				vals := []string{}
				if strings.Contains(value, ",") {
					ary := strings.Split(value, ",")
					for _, key := range ary {
						key = strings.Replace(key, "[", "", -1)
						key = strings.Replace(key, "]", "", -1)
						lib.AddRedacted(key, true)
						vals = append(vals, key)
					}
				} else {
					vals = append(vals, value)
				}
				inf := idxSlug + "/" + ds
				c = append(c, lib.MultiConfig{Name: "-t", Value: mergeTokens(ctx, inf, vals, ctx.OAuthKeys), RedactedValue: []string{lib.Redacted}})
				// OAuthKeys
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
			}
		}
		_, ok := m["sleep-for-rate"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "sleep-for-rate", Value: []string{}, RedactedValue: []string{}})
		}
		_, ok = m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}, RedactedValue: []string{}})
		}
	} else if ds == lib.Git {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				redactedValue = lib.Redacted
				lib.AddRedacted(value, true)
			}
			m[name] = struct{}{}
			c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
		}
		if ctx.LatestItems {
			_, ok := m["latest-items"]
			if !ok {
				c = append(c, lib.MultiConfig{Name: "latest-items", Value: []string{}, RedactedValue: []string{}})
			}
		}
	} else if ds == lib.Confluence {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				redactedValue = lib.Redacted
				lib.AddRedacted(value, true)
			}
			m[name] = struct{}{}
			c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}, RedactedValue: []string{}})
		}
	} else if ds == lib.Gerrit {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				lib.AddRedacted(value, true)
				redactedValue = lib.Redacted
			}
			if name == "ssh-key" {
				fail = !addSSHPrivKey(ctx, value, idxSlug)
				if !fail {
					keyAdded = true
				}
				continue
			}
			m[name] = struct{}{}
			c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
		}
		_, ok := m["disable-host-key-check"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "disable-host-key-check", Value: []string{}, RedactedValue: []string{}})
		}
		_, ok = m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}, RedactedValue: []string{}})
		}
	} else if ds == lib.Jira {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				lib.AddRedacted(value, true)
				redactedValue = lib.Redacted
			}
			m[name] = struct{}{}
			if name == lib.BackendUser {
				c = append(c, lib.MultiConfig{Name: "-u", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else if name == lib.BackendPassword {
				c = append(c, lib.MultiConfig{Name: "-p", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}, RedactedValue: []string{}})
		}
		_, ok = m["no-ssl-verify"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-ssl-verify", Value: []string{}, RedactedValue: []string{}})
		}
	} else if ds == lib.Slack {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				lib.AddRedacted(value, true)
				redactedValue = lib.Redacted
			}
			m[name] = struct{}{}
			if name == lib.APIToken {
				c = append(c, lib.MultiConfig{Name: "-t", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}, RedactedValue: []string{}})
		}
	} else if ds == lib.GroupsIO {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				lib.AddRedacted(value, true)
				redactedValue = lib.Redacted
			}
			m[name] = struct{}{}
			if name == lib.Email {
				c = append(c, lib.MultiConfig{Name: "-e", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else if name == lib.Password {
				c = append(c, lib.MultiConfig{Name: "-p", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
			}
		}
		_, ok := m["no-ssl-verify"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-ssl-verify", Value: []string{}, RedactedValue: []string{}})
		}
	} else if ds == lib.Pipermail {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				redactedValue = lib.Redacted
				lib.AddRedacted(value, true)
			}
			m[name] = struct{}{}
			c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
		}
		_, ok := m["no-ssl-verify"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-ssl-verify", Value: []string{}, RedactedValue: []string{}})
		}
	} else if ds == lib.Discourse {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				lib.AddRedacted(value, true)
				redactedValue = lib.Redacted
			}
			m[name] = struct{}{}
			if name == lib.APIToken {
				if strings.Contains(value, ",") {
					ary := strings.Split(value, ",")
					randInitOnce.Do(func() {
						rand.Seed(time.Now().UnixNano())
					})
					idx := rand.Intn(len(ary))
					lib.AddRedacted(ary[idx], true)
					c = append(c, lib.MultiConfig{Name: "-t", Value: []string{ary[idx]}, RedactedValue: []string{lib.Redacted}})
				} else {
					c = append(c, lib.MultiConfig{Name: "-t", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
				}
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}, RedactedValue: []string{}})
		}
	} else if ds == lib.Jenkins {
		for _, cfg := range *config {
			name := cfg.Name
			if name == lib.FromDate {
				continue
			}
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				lib.AddRedacted(value, true)
				redactedValue = lib.Redacted
			}
			m[name] = struct{}{}
			if name == lib.APIToken {
				c = append(c, lib.MultiConfig{Name: "-t", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else if name == lib.BackendUser {
				c = append(c, lib.MultiConfig{Name: "-u", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}, RedactedValue: []string{}})
		}
	} else if ds == lib.DockerHub {
		for _, cfg := range *config {
			name := cfg.Name
			if name == lib.FromDate {
				continue
			}
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				redactedValue = lib.Redacted
				lib.AddRedacted(value, true)
			}
			m[name] = struct{}{}
			c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
			_, ok := m["no-archive"]
			if !ok {
				c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}, RedactedValue: []string{}})
			}
		}
	} else if ds == lib.Bugzilla {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				lib.AddRedacted(value, true)
				redactedValue = lib.Redacted
			}
			m[name] = struct{}{}
			if name == lib.BackendUser {
				c = append(c, lib.MultiConfig{Name: "-u", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else if name == lib.BackendPassword {
				c = append(c, lib.MultiConfig{Name: "-p", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}, RedactedValue: []string{}})
		}
	} else if ds == lib.BugzillaRest {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				lib.AddRedacted(value, true)
				redactedValue = lib.Redacted
			}
			m[name] = struct{}{}
			if name == lib.BackendUser {
				c = append(c, lib.MultiConfig{Name: "-u", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else if name == lib.BackendPassword {
				c = append(c, lib.MultiConfig{Name: "-p", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else if name == lib.APIToken {
				c = append(c, lib.MultiConfig{Name: "-t", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}, RedactedValue: []string{}})
		}
	} else if ds == lib.MeetUp {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				lib.AddRedacted(value, true)
				redactedValue = lib.Redacted
			}
			m[name] = struct{}{}
			if name == lib.APIToken {
				c = append(c, lib.MultiConfig{Name: "-t", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}, RedactedValue: []string{}})
		}
		_, ok = m["sleep-for-rate"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "sleep-for-rate", Value: []string{}, RedactedValue: []string{}})
		}
	} else if ds == lib.RocketChat {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				redactedValue = lib.Redacted
				lib.AddRedacted(value, true)
			}
			m[name] = struct{}{}
			if name == lib.APIToken {
				c = append(c, lib.MultiConfig{Name: "-t", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else if name == lib.UserID {
				c = append(c, lib.MultiConfig{Name: "-u", Value: []string{value}, RedactedValue: []string{lib.Redacted}})
			} else {
				c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
			}
		}
		_, ok := m["no-archive"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-archive", Value: []string{}, RedactedValue: []string{}})
		}
		_, ok = m["sleep-for-rate"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "sleep-for-rate", Value: []string{}, RedactedValue: []string{}})
		}
		_, ok = m["min-rate-to-sleep"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "min-rate-to-sleep", Value: []string{"100"}, RedactedValue: []string{"100"}})
		}
		_, ok = m["no-ssl-verify"]
		if !ok {
			c = append(c, lib.MultiConfig{Name: "no-ssl-verify", Value: []string{}, RedactedValue: []string{}})
		}
	} else {
		fail = true
	}
	return
}

//func massageDataSource(ds string) string {
//	return ds
//}

func searchByQueryFirstID(ctx *lib.Ctx, index, esQuery string) (id string) {
	data := lib.EsSearchPayload{Query: lib.EsSearchQuery{QueryString: lib.EsSearchQueryString{Query: esQuery}}}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		lib.Printf("JSON marshall error: %+v for search: %s query: %s\n", err, index, esQuery)
		return
	}
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Post
	url := fmt.Sprintf("%s/%s/_doc/_search?size=1", ctx.ElasticURL, index)
	rurl := fmt.Sprintf("/%s/_doc/_search?size=1", index)
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s, query: %s\n", err, method, rurl, esQuery)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s, query: %s\n", err, method, rurl, esQuery)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, query: %s\n", err, method, rurl, esQuery)
			return
		}
		lib.Printf("Method:%s url:%s status:%d query:%s\n%s\n", method, rurl, resp.StatusCode, esQuery, body)
		return
	}
	payload := lib.EsSearchResultPayload{}
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		body, err := ioutil.ReadAll(resp.Body)
		lib.Printf("JSON decode error: %+v for %s url: %s, query: %s\n", err, method, rurl, esQuery)
		lib.Printf("Body:%s\n", body)
		return
	}
	for _, hit := range payload.Hits.Hits {
		id = hit.ID
		break
	}
	return
}

func searchByQuery(ctx *lib.Ctx, index, esQuery string) (dt time.Time, ok, found bool) {
	data := lib.EsSearchPayload{Query: lib.EsSearchQuery{QueryString: lib.EsSearchQueryString{Query: esQuery}}}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		lib.Printf("JSON marshall error: %+v for search: %s query: %s\n", err, index, esQuery)
		return
	}
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Post
	url := fmt.Sprintf("%s/%s/_doc/_search?size=1000", ctx.ElasticURL, index)
	rurl := fmt.Sprintf("/%s/_doc/_search?size=1000", index)
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s, query: %s\n", err, method, rurl, esQuery)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s, query: %s\n", err, method, rurl, esQuery)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, query: %s\n", err, method, rurl, esQuery)
			return
		}
		lib.Printf("Method:%s url:%s status:%d query:%s\n%s\n", method, rurl, resp.StatusCode, esQuery, body)
		return
	}
	payload := lib.EsSearchResultPayload{}
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		body, err := ioutil.ReadAll(resp.Body)
		lib.Printf("JSON decode error: %+v for %s url: %s, query: %s\n", err, method, rurl, esQuery)
		lib.Printf("Body:%s\n", body)
		return
	}
	ok = true
	dts := []time.Time{}
	for _, hit := range payload.Hits.Hits {
		dts = append(dts, hit.Source.Dt)
	}
	if len(dts) > 0 {
		found = true
		sort.SliceStable(dts, func(i, j int) bool {
			return dts[i].After(dts[j])
		})
		dt = dts[0]
	}
	return
}

func deleteByQuery(ctx *lib.Ctx, index, esQuery string) (ok bool) {
	data := lib.EsSearchPayload{Query: lib.EsSearchQuery{QueryString: lib.EsSearchQueryString{Query: esQuery}}}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		lib.Printf("JSON marshall error: %+v for search: %s query: %s\n", err, index, esQuery)
		return
	}
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Post
	url := fmt.Sprintf("%s/%s/_delete_by_query?conflicts=proceed&refresh=true&timeout=20m", ctx.ElasticURL, index)
	rurl := fmt.Sprintf("/%s/_delete_by_query?conflicts=proceed&refresh=true&timeout=20m", index)
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s, query: %s\n", err, method, rurl, esQuery)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s, query: %s\n", err, method, rurl, esQuery)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		if resp.StatusCode == 409 {
			if ctx.Debug > 0 {
				lib.Printf("Delete by query failed: index=%s query=%s\n", index, esQuery)
			}
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, query: %s\n", err, method, rurl, esQuery)
			return
		}
		lib.Printf("Method:%s url:%s status:%d query:%s\n%s\n", method, rurl, resp.StatusCode, esQuery, body)
		return
	}
	ok = true
	return
}

func addLastRun(ctx *lib.Ctx, dataIndex, index, ep string) (ok bool) {
	data := lib.EsLastRunPayload{Index: index, Endpoint: ep, Type: "last_sync", Dt: time.Now()}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		lib.Printf("JSON marshall error: %+v for index: %s, endpoint: %s, data: %+v\n", err, index, ep, data)
		return
	}
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Post
	url := fmt.Sprintf("%s/%s/_doc?refresh=wait_for", ctx.ElasticURL, dataIndex)
	rurl := fmt.Sprintf("/%s/_doc?refresh=wait_for", dataIndex)
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s, data: %+v\n", err, method, rurl, data)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s, data: %+v\n", err, method, rurl, data)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 201 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, data: %+v\n", err, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s status:%d data:%+v\n%s\n", method, rurl, resp.StatusCode, data, body)
		return
	}
	ok = true
	return
}

func setLastRun(ctx *lib.Ctx, tMtx *lib.TaskMtx, index, ep string) bool {
	esQuery := fmt.Sprintf("index:\"%s\" AND endpoint:\"%s\" AND type:\"last_sync\"", index, ep)
	if tMtx != nil && tMtx.SyncFreqMtx != nil {
		tMtx.SyncFreqMtx.Lock()
		defer func() {
			tMtx.SyncFreqMtx.Unlock()
		}()
	}
	dt, ok, found := searchByQuery(ctx, "sdsdata", esQuery)
	if !ok {
		return false
	}
	if found {
		if ctx.Debug > 0 {
			lib.Printf("Previous sync recorded for %s/%s: %+v, deleting\n", index, ep, dt)
		}
		if ctx.DryRun {
			if !ctx.DryRunAllowFreq {
				lib.Printf("Would delete and then add last sync date via: %s\n", esQuery)
				return true
			}
			lib.Printf("Dry run allowed delete and then add last sync date via: %s\n", esQuery)
		}
		trials := 0
		for {
			deleted := deleteByQuery(ctx, "sdsdata", esQuery)
			if deleted {
				if trials > 0 {
					lib.Printf("Deleted sync record for %s/%s after %d/%d trials\n", index, ep, trials, ctx.MaxDeleteTrials)
				}
				break
			}
			trials++
			if trials == ctx.MaxDeleteTrials {
				lib.Printf("Failed to delete sync record for %s/%s, tried %d times\n", index, ep, ctx.MaxDeleteTrials)
				return false
			}
			if tMtx != nil {
				tMtx.SyncFreqMtx.Unlock()
			}
			time.Sleep(time.Duration(10*trials) * time.Millisecond)
			if tMtx != nil {
				tMtx.SyncFreqMtx.Lock()
			}
		}
	} else {
		if ctx.Debug > 0 {
			lib.Printf("No previous sync recorded for %s/%s\n", index, ep)
		}
	}
	if ctx.Debug > 0 {
		lib.Printf("Adding sync record for %s/%s\n", index, ep)
	}
	if ctx.DryRun {
		if !ctx.DryRunAllowFreq {
			lib.Printf("Would add last sync date for: %s/%s\n", index, ep)
			return true
		}
		lib.Printf("Dry run allowed add last sync date for: %s/%s\n", index, ep)
	}
	added := addLastRun(ctx, "sdsdata", index, ep)
	return added
}

func checkSyncFreq(ctx *lib.Ctx, tMtx *lib.TaskMtx, index, ep string, freq time.Duration) bool {
	esQuery := fmt.Sprintf("index:\"%s\" AND endpoint:\"%s\" AND type:\"last_sync\"", index, ep)
	if tMtx != nil && tMtx.SyncFreqMtx != nil {
		tMtx.SyncFreqMtx.RLock()
		defer func() {
			tMtx.SyncFreqMtx.RUnlock()
		}()
	}
	dt, ok, found := searchByQuery(ctx, "sdsdata", esQuery)
	if !ok {
		lib.Printf("Error getting last sync date, assuming all is OK and allowing sync\n")
		return true
	}
	if !found {
		if ctx.Debug > 0 {
			lib.Printf("No previous sync recorded for %s/%s, allowing sync\n", index, ep)
		}
		return true
	}
	ago := time.Now().Sub(dt)
	allowed := true
	if ago < freq {
		allowed = false
	}
	if ctx.Debug > 0 {
		if !allowed {
			lib.Printf("%s/%s Freq: %+v, Ago: %+v, allowed: %+v (wait %+v)\n", index, ep, freq, ago, allowed, freq-ago)
		} else {
			lib.Printf("%s/%s Freq: %+v, Ago: %+v, allowed: %+v\n", index, ep, freq, ago, allowed)
		}
	}
	return allowed
}

func jsonEscape(str string) string {
	b, _ := json.Marshal(str)
	return string(b[1 : len(b)-1])
}

func lastProjectDate(ctx *lib.Ctx, index, origin, must, mustNot string, silent bool) (epoch int64) {
	data := ""
	mustPartial := ""
	if must != "" {
		mustPartial = "," + must
	}
	optionalMustNot := ""
	if mustNot != "" {
		optionalMustNot = `,"must_not":[` + mustNot + "]"
	}
	if origin == "" {
		data = fmt.Sprintf(
			`{"query":{"bool":{"must":[{"exists":{"field":"project"}}%s]%s}},"sort":{"project_ts":"desc"}}`,
			mustPartial,
			optionalMustNot,
		)
	} else {
		data = fmt.Sprintf(
			`{"query":{"bool":{"must":[{"exists":{"field":"project"}},{"term":{"origin":"%s"}}%s]%s}},"sort":{"project_ts":"desc"}}`,
			jsonEscape(origin),
			mustPartial,
			optionalMustNot,
		)
	}
	payloadBytes := []byte(data)
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Post
	url := fmt.Sprintf("%s/%s/_doc/_search?size=1", ctx.ElasticURL, index)
	rurl := fmt.Sprintf("/%s/_doc/_search?size=1", index)
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		if silent {
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s status:%d data:%s\n%s\n", method, rurl, resp.StatusCode, data, body)
		return
	}
	payload := lib.EsSearchResultPayload{}
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		body, err := ioutil.ReadAll(resp.Body)
		lib.Printf("JSON decode error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		lib.Printf("Body:%s\n", body)
		return
	}
	if len(payload.Hits.Hits) == 0 {
		return
	}
	epoch = payload.Hits.Hits[0].Source.ProjectTS
	return
}

func setSyncInfo(ctx *lib.Ctx, tMtx *lib.TaskMtx, result *lib.TaskResult, before bool) {
	if ctx.SkipSyncInfo || (ctx.DryRun && !ctx.DryRunAllowSyncInfo) || (!ctx.DryRun && ctx.SkipP2O) {
		return
	}
	if tMtx != nil && tMtx.SyncInfoMtx != nil {
		tMtx.SyncInfoMtx.Lock()
		defer func() {
			tMtx.SyncInfoMtx.Unlock()
		}()
	}
	// sdssyncinfo
	// before   -> field modificator: attempt/success/error
	// result:
	// Affs     -> field modificator: data_sync/enrich
	// Err      -> *error
	// Index    -> index
	// Endpoint -> endpoint
	affs := result.Affs
	esIndex := "sdssyncinfo"
	now := time.Now()
	errStr := ""
	if result.Err != nil {
		errStr = lib.FilterRedacted(result.Err.Error())
	}
	cl := result.CommandLine
	rcl := result.RedactedCommandLine
	esQuery := fmt.Sprintf("index:\"%s\" AND endpoint:\"%s\"", result.Index, result.Endpoint)
	id := searchByQueryFirstID(ctx, esIndex, esQuery)
	var (
		err          error
		payloadBytes []byte
	)
	if id != "" {
		// Update record
		data := `{"doc":{`
		data += fmt.Sprintf(`"dt":"%s",`, now.Format(time.RFC3339Nano))
		if affs {
			data += fmt.Sprintf(`"enrich_command_line":"%s",`, jsonEscape(cl))
			data += fmt.Sprintf(`"enrich_redacted_command_line":"%s",`, jsonEscape(rcl))
		} else {
			data += fmt.Sprintf(`"data_sync_command_line":"%s",`, jsonEscape(cl))
			data += fmt.Sprintf(`"data_sync_redacted_command_line":"%s",`, jsonEscape(rcl))
		}
		if before {
			if affs {
				data += fmt.Sprintf(`"enrich_attempt_dt":"%s",`, now.Format(time.RFC3339Nano))
			} else {
				data += fmt.Sprintf(`"data_sync_attempt_dt":"%s",`, now.Format(time.RFC3339Nano))
			}
		} else {
			if affs {
				if result.Err == nil {
					data += `"enrich_error":null,`
					data += `"enrich_error_dt":null,`
					data += fmt.Sprintf(`"enrich_success_dt":"%s",`, now.Format(time.RFC3339Nano))
				} else {
					data += fmt.Sprintf(`"enrich_error":"%s",`, jsonEscape(errStr))
					data += fmt.Sprintf(`"enrich_error_dt":"%s",`, now.Format(time.RFC3339Nano))
				}
			} else {
				if result.Err == nil {
					data += `"data_sync_error":null,`
					data += `"data_sync_error_dt":null,`
					data += fmt.Sprintf(`"data_sync_success_dt":"%s",`, now.Format(time.RFC3339Nano))
				} else {
					data += fmt.Sprintf(`"data_sync_error":"%s",`, jsonEscape(errStr))
					data += fmt.Sprintf(`"data_sync_error_dt":"%s",`, now.Format(time.RFC3339Nano))
				}
			}
		}
		data = data[:len(data)-1] + "}}"
		payloadBytes = []byte(data)
	} else {
		// New record
		data := lib.EsSyncInfoPayload{Index: result.Index, Endpoint: result.Endpoint, Dt: time.Now()}
		if affs {
			data.EnrichCL = &cl
			data.EnrichRCL = &rcl
		} else {
			data.DataSyncCL = &cl
			data.DataSyncRCL = &rcl
		}
		if before {
			if affs {
				data.EnrichAttemptDt = &now
			} else {
				data.DataSyncAttemptDt = &now
			}
		} else {
			if affs {
				if result.Err == nil {
					data.EnrichSuccessDt = &now
				} else {
					data.EnrichError = &errStr
					data.EnrichErrorDt = &now
				}
			} else {
				if result.Err == nil {
					data.DataSyncSuccessDt = &now
				} else {
					data.DataSyncError = &errStr
					data.DataSyncErrorDt = &now
				}
			}
		}
		payloadBytes, err = json.Marshal(data)
		if err != nil {
			lib.Printf("JSON marshall error: %+v for index: %s, endpoint: %s, data: %+v\n", err, result.Index, result.Endpoint, data)
			return
		}
	}
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Post
	var (
		url          string
		rurl         string
		expectedCode int
	)
	if id == "" {
		url = fmt.Sprintf("%s/%s/_doc?refresh=wait_for", ctx.ElasticURL, esIndex)
		rurl = fmt.Sprintf("/%s/_doc?refresh=wait_for", esIndex)
		expectedCode = 201
	} else {
		url = fmt.Sprintf("%s/%s/_update/%s?refresh=wait_for&retry_on_conflict=5", ctx.ElasticURL, esIndex, id)
		rurl = fmt.Sprintf("/%s/_update/%s?refresh=wait_for&retry_on_conflict=5", esIndex, id)
		expectedCode = 200
	}
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		data := string(payloadBytes)
		lib.Printf("New request error: %+v for %s url: %s, data: %+v\n", err, method, rurl, data)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		data := string(payloadBytes)
		lib.Printf("Do request error: %+v for %s url: %s, data: %+v\n", err, method, rurl, data)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != expectedCode {
		data := string(payloadBytes)
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, data: %+v\n", err, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s status:%d data:%+v\n%s\n", method, rurl, resp.StatusCode, data, body)
		return
	}
}

func getConditionJSON(conds []lib.ColumnCondition, origin string) (s string) {
	for _, cond := range conds {
		val := cond.Value
		if val == "{{endpoint}}" {
			val = origin
		}
		s += fmt.Sprintf(`{"regexp":{"%s":{"value":"%s", "flags":"ALL"}}},`, jsonEscape(cond.Column), jsonEscape(val))
	}
	l := len(s)
	if l > 0 {
		s = s[0 : l-1]
	}
	return
}

func setProject(ctx *lib.Ctx, index string, projects []lib.EndpointProject) {
	if ctx.SkipProject || (ctx.DryRun && !ctx.DryRunAllowProject) {
		return
	}
	if len(projects) == 0 {
		lib.Printf("No projects configuration specified for index %s\n", index)
	}
	for _, conf := range projects {
		project := conf.Name
		origin := conf.Origin
		if project == "" || origin == "" {
			continue
		}
		if ctx.Debug > 0 {
			lib.Printf("Setting project %+v origin (index '%s')\n", conf, index)
		}
		lastEpoch := int64(0)
		must := getConditionJSON(conf.Must, origin)
		mustPartial := ""
		if must != "" {
			mustPartial = "," + must
		}
		mustNot := getConditionJSON(conf.MustNot, origin)
		optionalMustNot := ""
		mustNotPartial := ""
		if mustNot != "" {
			optionalMustNot = `,"must_not":[` + mustNot + "]"
			mustNotPartial = "," + mustNot
		}
		if !ctx.SkipProjectTS {
			lastEpoch = lastProjectDate(ctx, index, origin, must, mustNot, true)
		}
		projectEpoch := time.Now().Unix()
		var err error
		payloadBytes := []byte{}
		data := ""
		var projectVal string
		if project == "(null)" {
			projectVal = "null"
		} else {
			projectVal = `\"` + jsonEscape(project) + `\"`
		}
		if lastEpoch == 0 {
			data = fmt.Sprintf(
				`{"script":{"inline":"ctx._source.project=%s;ctx._source.project_ts=%d;"},"query":{"bool":{"must":[{"term":{"origin":"%s"}}%s]%s}}}`,
				projectVal,
				projectEpoch,
				jsonEscape(origin),
				mustPartial,
				optionalMustNot,
			)
			payloadBytes = []byte(data)
		} else {
			data = fmt.Sprintf(
				`{"script":{"inline":"ctx._source.project=%s;ctx._source.project_ts=%d;"},"query":{"bool":{"must_not":[{"range":{"project_ts":{"lte":%d}}}%s],"must":[{"term":{"origin":"%s"}}%s]}}}`,
				projectVal,
				projectEpoch,
				lastEpoch,
				mustNotPartial,
				jsonEscape(origin),
				mustPartial,
			)
			payloadBytes = []byte(data)
		}
		payloadBody := bytes.NewReader(payloadBytes)
		method := lib.Post
		url := fmt.Sprintf("%s/%s/_update_by_query?conflicts=proceed&refresh=true&timeout=20m", ctx.ElasticURL, index)
		rurl := fmt.Sprintf("/%s/_update_by_query?conflicts=proceed&refresh=true&timeout=20m", index)
		req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
		if err != nil {
			lib.Printf("new request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lib.Printf("do request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode != 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				lib.Printf("ReadAll request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
				return
			}
			lib.Printf("Method:%s url:%s status:%d data:%+v\n%s", method, rurl, resp.StatusCode, data, body)
			return
		}
		payload := lib.EsUpdateByQueryPayload{}
		err = json.NewDecoder(resp.Body).Decode(&payload)
		if err != nil {
			body, err2 := ioutil.ReadAll(resp.Body)
			if err2 != nil {
				lib.Printf("ReadAll request error when parsing response: %+v/%+v for %s url: %s, data: %+v", err, err2, method, rurl, data)
				return
			}
			lib.Printf("Method:%s url:%s status:%d data:%+v err:%+v\n%s", method, rurl, resp.StatusCode, data, err, body)
			return
		}
		if ctx.DryRun || ctx.Debug > 0 {
			lib.Printf("Set project '%s'/%d on '%s'/%d origin (index '%s', config %+v): updated: %d\n", project, projectEpoch, origin, lastEpoch, index, conf, payload.Updated)
		} else {
			lib.PrintLogf("Set project '%s'/%d on '%s'/%d origin (index '%s', config %+v): updated: %d\n", project, projectEpoch, origin, lastEpoch, index, conf, payload.Updated)
		}
	}
}

func lastDataDate(ctx *lib.Ctx, index, must, mustNot string, silent bool) (epoch time.Time) {
	mustPartial := ""
	if must != "" {
		mustPartial = "," + must
	}
	optionalMustNot := ""
	if mustNot != "" {
		optionalMustNot = `,"must_not":[` + mustNot + "]"
	}
	data := fmt.Sprintf(
		`{"query":{"bool":{"must":[{"exists":{"field":"date"}}%s]%s}},"sort":{"date":"desc"}}`,
		mustPartial,
		optionalMustNot,
	)
	if ctx.Debug > 0 {
		lib.Printf("lastDataDate query: %s:%s\n", index, data)
	}
	payloadBytes := []byte(data)
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Post
	url := fmt.Sprintf("%s/%s/_doc/_search?size=1", ctx.ElasticURL, index)
	rurl := fmt.Sprintf("/%s/_doc/_search?size=1", index)
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		if silent {
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s status:%d data:%s\n%s\n", method, rurl, resp.StatusCode, data, body)
		return
	}
	payload := lib.EsSearchResultPayload{}
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		body, err := ioutil.ReadAll(resp.Body)
		lib.Printf("JSON decode error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		lib.Printf("Body:%s\n", body)
		return
	}
	if len(payload.Hits.Hits) == 0 {
		return
	}
	epoch = payload.Hits.Hits[0].Source.Date
	return
}

func bulkJSONData(ctx *lib.Ctx, index string, payloadBytes []byte) (err error) {
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Post
	url := fmt.Sprintf("%s/%s/_bulk?refresh=wait_for", ctx.ElasticURL, index)
	rurl := fmt.Sprintf("/%s/_bulk?refresh=wait_for", index)
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("new request error: %+v for %s url: %s, data: %+v", err, method, rurl, string(payloadBytes))
		return
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("do request error: %+v for %s url: %s, payload: %s\n", err, method, rurl, string(payloadBytes))
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, payload: %s\n", err, method, rurl, string(payloadBytes))
			return
		}
		err = fmt.Errorf("Method:%s url:%s status:%d data:%+v\n%s", method, rurl, resp.StatusCode, string(payloadBytes), body)
		return
	}
	esResult := lib.EsBulkResult{}
	err = json.Unmarshal(body, &esResult)
	if err != nil {
		lib.Printf("Bulk result unmarshal error: %+v", err)
		return
	}
	for i, item := range esResult.Items {
		if item.Index.Status != 201 {
			err = fmt.Errorf("Failed to create #%d item, status %d, error %+v\n", i, item.Index.Status, item.Index.Error)
			return
		}
	}
	return
}

func processJSON(ctx *lib.Ctx, index string, bulkNum, lineNum int, payloadBytes []byte) (err error) {
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Post
	url := fmt.Sprintf("%s/%s/_doc", ctx.ElasticURL, index)
	rurl := fmt.Sprintf("/%s/_doc", index)
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s, payload: %s\n", err, method, rurl, string(payloadBytes))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s, payload: %s\n", err, method, rurl, string(payloadBytes))
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 201 {
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, payload: %s\n", err, method, rurl, string(payloadBytes))
			return
		}
		err = fmt.Errorf("Method:%s url:%s status:%d data:%+v\n%s", method, rurl, resp.StatusCode, string(payloadBytes), body)
		return
	}
	return
}

func bulkCopy(ctx *lib.Ctx, bulkNum int, index string, jsons [][]byte) (err error) {
	if ctx.Debug > 0 {
		lib.Printf("Saving #%d bulk %d JSONs\n", bulkNum, len(jsons))
	}
	hdr := []byte("{\"index\":{\"_index\":\"" + index + "\"}}\n")
	payloads := []byte{}
	newLine := []byte("\n")
	for _, doc := range jsons {
		payloads = append(payloads, hdr...)
		payloads = append(payloads, doc...)
		payloads = append(payloads, newLine...)
	}
	nItems := len(jsons)
	er := bulkJSONData(ctx, index, payloads)
	ers := []error{}
	if er != nil {
		lib.Printf("Warning: critical bulk failure, fallback to line by line mode for bucket %d (%d lines): %+v\n", bulkNum, nItems, er)
		for i, doc := range jsons {
			err = processJSON(ctx, index, bulkNum, i, doc)
			if err != nil {
				lib.Printf("bulk #%d, line %d/%d, error: %+v\n", bulkNum, i, nItems, err)
				ers = append(ers, err)
			}
		}
	}
	if len(ers) > 0 {
		err = fmt.Errorf("bulk #%d, docuemnts: %d, errors: %d, last: %+v", bulkNum, nItems, len(ers), ers[len(ers)-1])
	}
	return
}

func copyMapping(ctx *lib.Ctx, pattern, index string) (err error) {
	// Get mapping(s) from pattern
	rurl := "/" + pattern + "/_mapping"
	url := ctx.ElasticURL + rurl
	method := lib.Get
	req, err := http.NewRequest(method, os.ExpandEnv(url), nil)
	if err != nil {
		lib.Printf("new request error: %+v for %s url: %s", err, method, rurl)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("do request error: %+v for %s url: %s", err, method, rurl)
		return
	}
	if resp.StatusCode != 200 {
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s", err, method, rurl)
			return
		}
		lib.Printf("Method:%s url:%s status:%d\n%s", method, rurl, resp.StatusCode, body)
		return
	}
	var (
		result interface{}
		field  interface{}
	)
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		lib.Printf("JSON decode error: %+v for %s url: %s\n", err, method, rurl)
		lib.Printf("Body:%s\n", body)
		return
	}
	_ = resp.Body.Close()
	// Attempt to create index
	rurl = "/" + index
	url = ctx.ElasticURL + rurl
	method = lib.Put
	req, err = http.NewRequest(method, os.ExpandEnv(url), nil)
	if err != nil {
		lib.Printf("new request error: %+v for %s url: %s", err, method, rurl)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("do request error: %+v for %s url: %s", err, method, rurl)
		return
	}
	if resp.StatusCode != 200 && resp.StatusCode != 400 {
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s", err, method, rurl)
			return
		}
		lib.Printf("Method:%s url:%s status:%d\n%s", method, rurl, resp.StatusCode, body)
		return
	}
	if ctx.Debug >= 0 {
		if resp.StatusCode == 200 {
			lib.Printf("copy_from: created new index: %s\n", index)
		} else if resp.StatusCode == 400 {
			lib.Printf("copy_from: index %s already exists\n", index)
		}
	}
	_ = resp.Body.Close()
	root, ok := result.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("parse json root error")
		return
	}
	// Iterate mapping(s) from pattern
	i := 0
	nPatternIndices := len(root)
	mapping := make(map[string]map[string]interface{})
	mapping["properties"] = make(map[string]interface{})
	for patternIndex := range root {
		i++
		field = root[patternIndex]
		if err != nil {
			return
		}
		item, ok := field.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("parse json item error")
			return
		}
		mappings, ok := item["mappings"]
		if !ok {
			err = fmt.Errorf("parse json item mappings error")
			return
		}
		item, ok = mappings.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("parse json item 2 error")
			return
		}
		properties, ok := item["properties"]
		if !ok {
			err = fmt.Errorf("parse json item properties error")
			return
		}
		items, ok := properties.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("parse json items error")
			return
		}
		if ctx.Debug > 0 {
			lib.Printf("Pattern index %d/%d: %s\n", i, nPatternIndices, patternIndex)
		}
		j := 0
		nCols := len(items)
		for col, data := range items {
			j++
			if ctx.Debug > 1 {
				lib.Printf("Column %d/%d: %s: %+v\n", j, nCols, col, data)
			}
			// First column def across all pattern indices win
			_, ok := mapping["properties"][col]
			if !ok {
				mapping["properties"][col] = data
			}
		}
	}
	// Final mapping write
	var jsonBytes []byte
	jsonBytes, err = json.Marshal(mapping)
	if err != nil {
		return
	}
	data := string(jsonBytes)
	// jsonBytes = []byte(`{"properties":{"ancestors_links":{"type":"keyword"}}}`)
	payloadBody := bytes.NewReader(jsonBytes)
	rurl = "/" + index + "/_mapping"
	url = ctx.ElasticURL + rurl
	method = lib.Put
	req, err = http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("new request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("do request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
		return
	}
	if resp.StatusCode != 200 && resp.StatusCode != 400 {
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s status:%d data:%+v\n%s", method, rurl, resp.StatusCode, data, body)
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode == 400 {
		thrN := lib.GetThreadsNum(ctx)
		if thrN > 8 {
			thrN = 8
		}
		lib.Printf("copy_from: Put entire mapping at once failed, fallback to column by column mode (%d threads)\n", thrN)
		rurl = "/" + index + "/_mapping"
		url = ctx.ElasticURL + rurl
		method = lib.Put
		// This will be used in parallel thread when putting all columns at once fails
		putColumn := func(ch chan error, col string, def interface{}) (err error) {
			if ch != nil {
				defer func() {
					ch <- err
				}()
			}
			mp := make(map[string]map[string]interface{})
			mp["properties"] = make(map[string]interface{})
			mp["properties"][col] = def
			var jsonBytes []byte
			jsonBytes, err = json.Marshal(mp)
			if err != nil {
				return
			}
			data := string(jsonBytes)
			payloadBody := bytes.NewReader(jsonBytes)
			var req *http.Request
			req, err = http.NewRequest(method, os.ExpandEnv(url), payloadBody)
			if err != nil {
				lib.Printf("new request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			var resp *http.Response
			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				lib.Printf("do request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
				return
			}
			if resp.StatusCode != 200 {
				var body []byte
				body, err = ioutil.ReadAll(resp.Body)
				_ = resp.Body.Close()
				if err != nil {
					lib.Printf("ReadAll request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
					return
				}
				err = fmt.Errorf("method:%s url:%s status:%d data:%+v\n%s", method, rurl, resp.StatusCode, data, body)
				return
			}
			_ = resp.Body.Close()
			return
		}
		ers := 0
		if thrN > 1 {
			ch := make(chan error)
			nThreads := 0
			for col, data := range mapping["properties"] {
				go func() {
					_ = putColumn(ch, col, data)
				}()
				nThreads++
				if nThreads == thrN {
					err := <-ch
					if err != nil {
						lib.Printf("Column mapping error: %+v\n", err)
						ers++
					}
					nThreads--
				}
			}
			for nThreads > 0 {
				err := <-ch
				if err != nil {
					lib.Printf("Column mapping error: %+v\n", err)
					ers++
				}
				nThreads--
			}
		} else {
			for col, data := range mapping["properties"] {
				err := putColumn(nil, col, data)
				if err != nil {
					lib.Printf("Column mapping error: %+v\n", err)
					ers++
				}
			}
		}
		if ers > 0 {
			lib.Printf("Failed %d/%d columns\n", ers, len(mapping["properties"]))
		}
	}
	if ctx.Debug >= 0 {
		lib.Printf("%s -> %s: copied %d mappings (%d columns)\n", pattern, index, i, len(mapping["properties"]))
	}
	return
}

func handleCopyFrom(ctx *lib.Ctx, index string, task *lib.Task) (err error) {
	if ctx.SkipCopyFrom || (ctx.DryRun && !ctx.DryRunAllowCopyFrom) {
		return
	}
	scrollSize := 500
	scrollTime := "10m"
	bulkSize := 500
	conf := task.CopyFrom
	origin := mapOrigin(task.Endpoint, task.DsSlug)
	if ctx.Debug > 0 {
		lib.Printf("%s:%s: copy config: %+v\n", index, origin, conf)
	}
	must := getConditionJSON(conf.Must, origin)
	mustPartial := ""
	if must != "" {
		mustPartial = "," + must
	}
	mustNot := getConditionJSON(conf.MustNot, origin)
	optionalMustNot := ""
	mustNotPartial := ""
	if mustNot != "" {
		optionalMustNot = `,"must_not":[` + mustNot + "]"
		mustNotPartial = "," + mustNot
	}
	// There can already be an index alias under index name, we need to drop it
	method := lib.Delete
	url := fmt.Sprintf("%s/_all/_alias/%s", ctx.ElasticURL, index)
	rurl := "/_all/_alias/" + index
	req, err := http.NewRequest(method, os.ExpandEnv(url), nil)
	if err != nil {
		lib.Printf("New request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	_ = resp.Body.Close()
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	if resp.StatusCode == 200 {
		lib.Printf("Dropped conflicting alias: %s\n", index)
	}
	pattern := conf.Pattern
	err = copyMapping(ctx, pattern, index)
	if err != nil {
		lib.Printf("copyMapping(%s,%s): %v\n", pattern, index, err)
	}
	// Now check last date on index (not alias) if present
	lastDate := lastDataDate(ctx, index, must, mustNot, true)
	if ctx.Debug > 0 {
		lib.Printf("lastDataDate: %+v\n", lastDate)
	}
	payloadBytes := []byte{}
	data := ""
	if lastDate.IsZero() {
		data = fmt.Sprintf(
			`{"size":%d,"query":{"bool":{"must":[{"exists":{"field":"date"}}%s]%s}}}`,
			scrollSize,
			mustPartial,
			optionalMustNot,
		)
		payloadBytes = []byte(data)
	} else {
		millis := lastDate.UnixNano() / 1000000
		// millis -= 36000000
		data = fmt.Sprintf(
			`{"size":%d,"query":{"bool":{"must_not":[{"range":{"date":{"lte":%d,"format":"epoch_millis"}}}%s],"must":[{"exists":{"field":"date"}}%s]}}}`,
			scrollSize,
			millis,
			mustNotPartial,
			mustPartial,
		)
		payloadBytes = []byte(data)
	}
	payloadBody := bytes.NewReader(payloadBytes)
	method = lib.Post
	if ctx.Debug > 0 {
		lib.Printf("handleCopyFrom query: %s:%s\n", pattern, data)
	}
	url = fmt.Sprintf("%s/%s/_search?scroll=%s", ctx.ElasticURL, pattern, scrollTime)
	rurl = fmt.Sprintf("/%s/_search?scroll=%s", pattern, scrollTime)
	req, err = http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("new request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("do request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
		return
	}
	if resp.StatusCode != 200 {
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s status:%d data:%+v\n%s", method, rurl, resp.StatusCode, data, body)
		return
	}
	payload := lib.EsSearchScrollPayload{}
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		lib.Printf("JSON decode error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		lib.Printf("Body:%s\n", body)
		return
	}
	_ = resp.Body.Close()
	jsons := [][]byte{}
	nJSONs := 0
	bulks := 0
	scrollID := payload.ScrollID
	docs := 0
	for {
		data = fmt.Sprintf(`{"scroll":"%s", "scroll_id":"%s"}`, scrollTime, scrollID)
		payloadBytes = []byte(data)
		payloadBody = bytes.NewReader(payloadBytes)
		url = ctx.ElasticURL + lib.SearchScroll
		rurl = lib.SearchScroll
		method := lib.Get
		req, err = http.NewRequest(method, os.ExpandEnv(url), payloadBody)
		if err != nil {
			lib.Printf("new request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			lib.Printf("do request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
			return
		}
		if resp.StatusCode != 200 {
			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				lib.Printf("ReadAll request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
				return
			}
			lib.Printf("Method:%s url:%s status:%d data:%+v\n%s", method, rurl, resp.StatusCode, data, body)
			return
		}
		var (
			result interface{}
			field  interface{}
		)
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()
			lib.Printf("JSON decode error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
			lib.Printf("Body:%s\n", body)
			return
		}
		_ = resp.Body.Close()
		root, ok := result.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("parse json root error")
			return
		}
		field, ok = root["_scroll_id"]
		if !ok {
			err = fmt.Errorf("parse json field _scroll_id error")
			return
		}
		scrollID, ok = field.(string)
		if !ok {
			err = fmt.Errorf("parse json scrollID error")
			return
		}
		field, ok = root["hits"]
		if !ok {
			err = fmt.Errorf("parse json field hits error")
			return
		}
		hits, ok := field.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("parse json hits error")
			return
		}
		field, ok = hits["hits"]
		if !ok {
			err = fmt.Errorf("parse json field hits2 error")
			return
		}
		hits2, ok := field.([]interface{})
		if !ok {
			err = fmt.Errorf("parse json hits2 error")
			return
		}
		nHits := len(hits2)
		if ctx.Debug > 0 {
			lib.Printf("%s -> %s: Fetched %d documents\n", pattern, index, nHits)
		}
		if nHits == 0 {
			break
		}
		docs += nHits
		for i, item := range hits2 {
			root, ok := item.(map[string]interface{})
			if !ok {
				err = fmt.Errorf("parse json #%d item root error %+v", i, item)
				return
			}
			field, ok = root["_source"]
			if !ok {
				err = fmt.Errorf("parse json #%d item field _source error %+v", i, item)
				return
			}
			doc, ok := field.(map[string]interface{})
			if !ok {
				err = fmt.Errorf("parse json #%d item doc error: %+v", i, item)
				return
			}
			var jsonBytes []byte
			jsonBytes, err = json.Marshal(doc)
			if err != nil {
				return
			}
			jsons = append(jsons, jsonBytes)
			nJSONs++
			if nJSONs == bulkSize {
				err = bulkCopy(ctx, bulks, index, jsons)
				if err != nil {
					return
				}
				jsons = [][]byte{}
				nJSONs = 0
				bulks++
			}
		}
	}
	if nJSONs > 0 {
		err = bulkCopy(ctx, bulks, index, jsons)
		if err != nil {
			return
		}
		bulks++
	}
	if ctx.Debug > 0 {
		lib.Printf("Releasing scroll_id %s\n", scrollID)
	}
	data = fmt.Sprintf(`{"scroll_id":"%s"}`, scrollID)
	payloadBytes = []byte(data)
	payloadBody = bytes.NewReader(payloadBytes)
	url = ctx.ElasticURL + lib.SearchScroll
	rurl = lib.SearchScroll
	method = lib.Delete
	req, err = http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		lib.Printf("new request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		lib.Printf("do request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
		return
	}
	if resp.StatusCode != 200 {
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s status:%d data:%+v\n%s", method, rurl, resp.StatusCode, data, body)
		return
	}
	_ = resp.Body.Close()
	lib.Printf("copy_from: %s -> %s: saved %d bulks (%d documents)\n", pattern, index, bulks, docs)
	return
}

// mapOrigin maps fixture's origin to ES "origin" column, depending on data-source type
func mapOrigin(origin, ds string) string {
	switch ds {
	case lib.RocketChat:
		return strings.Replace(strings.Replace(strings.TrimSpace(origin), "  ", " ", -1), " ", "/", -1)
	case lib.DockerHub:
		return "https://hub.docker.com/" + strings.Replace(strings.Replace(strings.TrimSpace(origin), "  ", " ", -1), " ", "/", -1)
	default:
		return origin
	}
}

func setTaskResultProjects(result *lib.TaskResult, task *lib.Task) {
	if task.Project == "" {
		for _, project := range task.Projects {
			ep := lib.EndpointProject{Name: project.Name, Origin: mapOrigin(project.Origin, task.DsSlug)}
			for _, cond := range project.Must {
				ep.Must = append(ep.Must, lib.ColumnCondition{Column: cond.Column, Value: cond.Value})
			}
			for _, cond := range project.MustNot {
				ep.MustNot = append(ep.MustNot, lib.ColumnCondition{Column: cond.Column, Value: cond.Value})
			}
			result.Projects = append(result.Projects, ep)
		}
	} else {
		result.Projects = append(
			result.Projects,
			lib.EndpointProject{
				Name:   task.Project,
				Origin: mapOrigin(task.Endpoint, task.DsSlug),
			},
		)
	}
}

func taskFilteredOut(ctx *lib.Ctx, tsk *lib.Task) bool {
	idxSlug := "sds-" + tsk.FxSlug + "-" + tsk.DsFullSlug
	idxSlug = strings.Replace(idxSlug, "/", "-", -1)
	task := idxSlug + ":" + tsk.Endpoint
	if (ctx.TasksRE != nil && !ctx.TasksRE.MatchString(task)) || (ctx.TasksSkipRE != nil && ctx.TasksSkipRE.MatchString(task)) {
		if ctx.Debug > 0 {
			lib.Printf("Task %s filtered out due to RE (match,skip) = (%+v,%+v)\n", task, ctx.TasksRE, ctx.TasksSkipRE)
		}
		return true
	}
	return false
}

func processTask(ch chan lib.TaskResult, ctx *lib.Ctx, idx int, task lib.Task, affs bool, tMtx *lib.TaskMtx) (result lib.TaskResult) {
	// Ensure to unlock thread when finishing
	defer func() {
		// Synchronize go routine
		if ch != nil {
			ch <- result
		}
	}()
	if ctx.Debug > 1 {
		lib.Printf("Processing (affs: %+v): %s\n", affs, task)
	}
	result.Code[0] = idx
	result.Affs = affs
	if !affs && !task.ProjectP2O && (task.Project != "" || len(task.Projects) > 0) {
		setTaskResultProjects(&result, &task)
	}
	// Handle DS slug
	ds := task.DsSlug
	fds := task.DsFullSlug
	idxSlug := "sds-" + task.FxSlug + "-" + fds
	idxSlug = strings.Replace(idxSlug, "/", "-", -1)
	result.Index = idxSlug
	result.Endpoint = task.Endpoint
	// Filter out by task / task skip RE
	if taskFilteredOut(ctx, &task) {
		result.Code[1] = -1
		return
	}
	// Handle copy from another index slug
	if task.CopyFrom.Pattern != "" {
		err := handleCopyFrom(ctx, idxSlug, &task)
		if err != nil {
			result.Code[1] = 7
			result.Err = err
			return
		}
		if len(result.Projects) > 0 {
			setProject(ctx, idxSlug, result.Projects)
		}
		result.Code[1] = -2
		return
	}
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
	redactedCommandLine := make([]string, len(commandLine))
	copy(redactedCommandLine, commandLine)
	redactedCommandLine[len(redactedCommandLine)-1] = lib.Redacted
	if affs {
		refresh := []string{"--only-enrich", "--refresh-identities", "--no_incremental"}
		commandLine = append(commandLine, refresh...)
		redactedCommandLine = append(redactedCommandLine, refresh...)
	}
	// This only enables p2o.py -g flag (so only subcommand is executed with debug mode)
	if !ctx.Silent {
		commandLine = append(commandLine, "-g")
		redactedCommandLine = append(redactedCommandLine, "-g")
	}
	// This enabled debug mode on the p2o.py subcommand als also makes ExecCommand() call run in debug mode
	if ctx.CmdDebug > 0 {
		commandLine = append(commandLine, "--debug")
		redactedCommandLine = append(redactedCommandLine, "--debug")
	}
	if ctx.EsBulkSize > 0 {
		commandLine = append(commandLine, "--bulk-size")
		commandLine = append(commandLine, strconv.Itoa(ctx.EsBulkSize))
		redactedCommandLine = append(redactedCommandLine, "--bulk-size")
		redactedCommandLine = append(redactedCommandLine, strconv.Itoa(ctx.EsBulkSize))
	}
	if ctx.ScrollSize > 0 {
		commandLine = append(commandLine, "--scroll-size")
		commandLine = append(commandLine, strconv.Itoa(ctx.ScrollSize))
		redactedCommandLine = append(redactedCommandLine, "--scroll-size")
		redactedCommandLine = append(redactedCommandLine, strconv.Itoa(ctx.ScrollSize))
	}
	if ctx.ScrollWait > 0 {
		commandLine = append(commandLine, "--scroll-wait")
		commandLine = append(commandLine, strconv.Itoa(ctx.ScrollWait))
		redactedCommandLine = append(redactedCommandLine, "--scroll-wait")
		redactedCommandLine = append(redactedCommandLine, strconv.Itoa(ctx.ScrollWait))
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
		redactedCommandLine = append(
			redactedCommandLine,
			[]string{
				"--db-host",
				lib.Redacted,
				"--db-sortinghat",
				lib.Redacted,
				"--db-user",
				lib.Redacted,
				"--db-password",
				lib.Redacted,
			}...,
		)
	}
	if strings.Contains(ds, "/") {
		ary := strings.Split(ds, "/")
		if len(ary) != 2 {
			lib.Printf("%s: %+v: %s\n", ds, task, lib.ErrorStrings[1])
			result.Code[1] = 1
			return
		}
		commandLine = append(commandLine, ary[0])
		commandLine = append(commandLine, "--category")
		commandLine = append(commandLine, ary[1])
		redactedCommandLine = append(redactedCommandLine, ary[0])
		redactedCommandLine = append(redactedCommandLine, "--category")
		redactedCommandLine = append(redactedCommandLine, ary[1])
		ds = ary[0]
	} else {
		commandLine = append(commandLine, ds)
		redactedCommandLine = append(redactedCommandLine, ds)
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
		redactedCommandLine = append(redactedCommandLine, ep)
	}
	sEp := strings.Join(eps, " ")
	if !ctx.SkipEsData && !ctx.SkipCheckFreq {
		var nilDur time.Duration
		if task.MaxFreq != nilDur {
			freqOK := checkSyncFreq(ctx, tMtx, idxSlug, sEp, task.MaxFreq)
			if !freqOK {
				return
			}
		}
	}
	// Handle DS config options
	multiConfig, fail, keyAdded := massageConfig(ctx, &(task.Config), ds, idxSlug)
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
		for _, val := range mcfg.RedactedValue {
			if val != "" {
				redactedCommandLine = append(redactedCommandLine, val)
			}
		}
	}
	// Handle DS project
	if task.ProjectP2O && task.Project != "" {
		commandLine = append(commandLine, "--project", task.Project)
		redactedCommandLine = append(redactedCommandLine, "--project", task.Project)
	}
	result.CommandLine = strings.Join(commandLine, " ")
	result.RedactedCommandLine = strings.Join(redactedCommandLine, " ")
	if affs && tMtx.OrderMtx != nil {
		tMtx.TaskOrderMtx.Lock()
		tmtx, ok := tMtx.OrderMtx[idx]
		if !ok {
			tMtx.TaskOrderMtx.Unlock()
			lib.Fatalf("per task mutex map is defined, but no mutex for idx: %d", idx)
		}
		tMtx.TaskOrderMtx.Unlock()
		// Ensure that data sync task is finished before attempting to run historical affiliations
		st := time.Now()
		// lib.Printf("wait for mtx %d\n", idx)
		// we need to wait for data sync, then we lock reenrich but unlock immediatelly because no next step will process that task
		// in worst case it is possible that we must wait for data-sync until timeout (10h) and then reenrich can take 10h too
		// and be killed by another timeout, so a nasty endpoint can trigger 2 * task timeout wait time
		tmtx.Lock()
		tmtx.Unlock()
		// lib.Printf("mtx %d passed (affs task)\n", idx)
		tMtx.TaskOrderMtx.Lock()
		tMtx.OrderMtx[idx] = tmtx
		tMtx.TaskOrderMtx.Unlock()
		en := time.Now()
		took := en.Sub(st)
		if took > time.Duration(10)*time.Minute {
			lib.Printf("Waited for data sync on %d/%+v mutex: %v\n", idx, task, en.Sub(st))
		}
	}
	retries := 0
	dtStart := time.Now()
	for {
		if ctx.DryRun {
			if keyAdded {
				if tMtx.SSHKeyMtx != nil {
					tMtx.SSHKeyMtx.Lock()
				}
				gKeyMtx.Lock()
				fail := !makeCurrentSSHKey(ctx, idxSlug)
				if fail == true {
					gKeyMtx.Unlock()
					if tMtx.SSHKeyMtx != nil {
						tMtx.SSHKeyMtx.Unlock()
					}
					lib.Printf("%+v: %s\n", task, lib.ErrorStrings[5])
					result.Code[1] = 5
					return
				}
			}
			if ctx.DryRunAllowSyncInfo {
				setSyncInfo(ctx, tMtx, &result, true)
			}
			if ctx.DryRunSeconds > 0 {
				if ctx.DryRunSecondsRandom {
					time.Sleep(time.Duration(rand.Intn(ctx.DryRunSeconds*1000)) * time.Millisecond)
				} else {
					time.Sleep(time.Duration(ctx.DryRunSeconds) * time.Second)
				}
			}
			if keyAdded {
				gKeyMtx.Unlock()
				if tMtx.SSHKeyMtx != nil {
					tMtx.SSHKeyMtx.Unlock()
				}
			}
			if ctx.DryRunCodeRandom {
				rslt := rand.Intn(6)
				result.Code[1] = rslt
				if rslt > 0 {
					result.Err = fmt.Errorf("error: %d", rslt)
					result.Retries = rand.Intn(ctx.MaxRetry)
				}
			} else {
				result.Code[1] = ctx.DryRunCode
				if ctx.DryRunCode != 0 {
					result.Err = fmt.Errorf("error: %d", ctx.DryRunCode)
					result.Retries = rand.Intn(ctx.MaxRetry)
				}
			}
			if !ctx.SkipP2O && !ctx.SkipEsData && !affs {
				_ = setLastRun(ctx, tMtx, idxSlug, sEp)
			}
			return
		}
		if keyAdded {
			st := time.Now()
			if tMtx.SSHKeyMtx != nil {
				tMtx.SSHKeyMtx.Lock()
			}
			gKeyMtx.Lock()
			fail := !makeCurrentSSHKey(ctx, idxSlug)
			if fail == true {
				gKeyMtx.Unlock()
				if tMtx.SSHKeyMtx != nil {
					tMtx.SSHKeyMtx.Unlock()
				}
				lib.Printf("%+v: %s\n", task, lib.ErrorStrings[5])
				result.Code[1] = 5
				return
			}
			en := time.Now()
			took := en.Sub(st)
			if took > time.Duration(10)*time.Minute {
				lib.Printf("Waited for ssh key on %d/%+v mutex: %v\n", idx, task, en.Sub(st))
			}
		}
		setSyncInfo(ctx, tMtx, &result, true)
		var (
			err error
			str string
		)
		if !ctx.SkipP2O {
			str, err = lib.ExecCommand(ctx, commandLine, map[string]string{"PROJECT_SLUG": task.FxSlug}, &task.Timeout)
		}
		if keyAdded {
			gKeyMtx.Unlock()
			if tMtx.SSHKeyMtx != nil {
				tMtx.SSHKeyMtx.Unlock()
			}
		}
		// str = strings.Replace(str, ctx.ElasticURL, lib.Redacted, -1)
		// p2o.py do not return error even if its backend execution fails
		// we need to capture STDERR and check if there was python exception there
		pyE := false
		strippedStr := str
		strLen := len(str)
		if strLen > ctx.StripErrorSize {
			strippedStr = str[0:ctx.StripErrorSize] + "\n(...)\n" + str[strLen-ctx.StripErrorSize:strLen]
		}
		if strings.Contains(str, lib.PyException) {
			pyE = true
			err = fmt.Errorf("%s", strippedStr)
		}
		if err == nil {
			if ctx.Debug > 0 {
				dtEnd := time.Now()
				lib.Printf("%+v: finished in %v, retries: %d\n", task, dtEnd.Sub(dtStart), retries)
			}
			break
		}
		if isTimeoutError(err) {
			dtEnd := time.Now()
			lib.Printf("Timeout error for %+v (took %v, tried %d times): %+v: %s\n", redactedCommandLine, dtEnd.Sub(dtStart), retries, err, strippedStr)
			str += fmt.Sprintf(": %+v", err)
			result.Code[1] = 6
			result.Err = fmt.Errorf("timeout: last retry took %v: %+v", dtEnd.Sub(dtStart), strippedStr)
			result.Retries = retries
			return
		}
		retries++
		if retries <= ctx.MaxRetry {
			time.Sleep(time.Duration(retries) * time.Second)
			continue
		}
		dtEnd := time.Now()
		if pyE {
			lib.Printf("Python exception for %+v (took %v, tried %d times): %+v\n", redactedCommandLine, dtEnd.Sub(dtStart), retries, err)
		} else {
			lib.Printf("Error for %+v (took %v, tried %d times): %+v: %s\n", redactedCommandLine, dtEnd.Sub(dtStart), retries, err, strippedStr)
			str += fmt.Sprintf(": %+v", err)
		}
		result.Code[1] = 4
		result.Err = fmt.Errorf("last retry took %v: %+v", dtEnd.Sub(dtStart), strippedStr)
		result.Retries = retries
		return
	}
	if !ctx.SkipP2O && !ctx.SkipEsData && !affs {
		updated := setLastRun(ctx, tMtx, idxSlug, sEp)
		if !updated {
			lib.Printf("failed to set last sync date for %s/%s\n", idxSlug, sEp)
		}
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

func getToken(ctx *lib.Ctx) (err error) {
	if ctx.Auth0URL == "" || ctx.Auth0ClientID == "" || ctx.Auth0ClientSecret == "" || ctx.Auth0Audience == "" {
		err = fmt.Errorf("Cannot obtain auth0 bearer token - all auth0 parameters must be set")
		return
	}
	data := fmt.Sprintf(
		`{"grant_type":"client_credentials","client_id":"%s","client_secret":"%s","audience":"%s","scope":"access:api"}`,
		jsonEscape(ctx.Auth0ClientID),
		jsonEscape(ctx.Auth0ClientSecret),
		jsonEscape(ctx.Auth0Audience),
	)
	payloadBytes := []byte(data)
	payloadBody := bytes.NewReader(payloadBytes)
	method := http.MethodPost
	rurl := "/oauth/token"
	url := ctx.Auth0URL + rurl
	req, e := http.NewRequest(method, url, payloadBody)
	if e != nil {
		err = fmt.Errorf("new request error: %+v for %s url: %s", e, method, rurl)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		err = fmt.Errorf("do request error: %+v for %s url: %s", e, method, rurl)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		body, e := ioutil.ReadAll(resp.Body)
		if e != nil {
			err = fmt.Errorf("ReadAll non-ok request error: %+v for %s url: %s", e, method, rurl)
			return
		}
		err = fmt.Errorf("Method:%s url:%s status:%d\n%s", method, rurl, resp.StatusCode, body)
		return
	}
	var rdata struct {
		Token string `json:"access_token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&rdata)
	if err != nil {
		return
	}
	if rdata.Token == "" {
		err = fmt.Errorf("empty token retuned")
		return
	}
	lib.AddRedacted(rdata.Token, false)
	gToken = "Bearer " + rdata.Token
	lib.Printf("Generated new token(%d) [%s]\n", len(gToken), gToken)
	return
}

func executeAPICall(ctx *lib.Ctx, path string) (err error) {
	if ctx.AffiliationAPIURL == "" {
		err = fmt.Errorf("Cannot execute DA affiliation API calls, no API URL specified")
		return
	}
	if gToken == "" {
		gToken = os.Getenv("JWT_TOKEN")
	}
	if gToken == "" {
		lib.Printf("Obtaining API token\n")
		err = getToken(ctx)
		if err != nil {
			return
		}
	}
	method := http.MethodPut
	rurl := path
	url := ctx.AffiliationAPIURL + rurl
	for i := 0; i < 2; i++ {
		req, e := http.NewRequest(method, url, nil)
		if e != nil {
			err = fmt.Errorf("new request error: %+v for %s url: %s", e, method, rurl)
			return
		}
		req.Header.Set("Authorization", gToken)
		resp, e := http.DefaultClient.Do(req)
		if e != nil {
			err = fmt.Errorf("do request error: %+v for %s url: %s", e, method, rurl)
			return
		}
		if i == 0 && resp.StatusCode == 401 {
			_ = resp.Body.Close()
			lib.Printf("Token is invalid, trying to generate another one\n")
			err = getToken(ctx)
			if err != nil {
				return
			}
			continue
		}
		if resp.StatusCode != 200 {
			body, e := ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if e != nil {
				err = fmt.Errorf("ReadAll non-ok request error: %+v for %s url: %s", e, method, rurl)
				return
			}
			err = fmt.Errorf("Method:%s url:%s status:%d\n%s", method, rurl, resp.StatusCode, body)
			return
		}
		var rdata struct {
			Text string `json:"text"`
		}
		err = json.NewDecoder(resp.Body).Decode(&rdata)
		_ = resp.Body.Close()
		if err != nil {
			return
		}
		lib.Printf("%s\n", rdata.Text)
		break
	}
	return
}

func hideEmails(ctx *lib.Ctx) (err error) {
	if ctx.SkipHideEmails || ctx.OnlyP2O || (ctx.DryRun && !ctx.DryRunAllowHideEmails) {
		return
	}
	err = executeAPICall(ctx, "/v1/affiliation/hide_emails")
	return
}

func mergeAll(ctx *lib.Ctx) (err error) {
	if ctx.SkipMerge || ctx.OnlyP2O || (ctx.DryRun && !ctx.DryRunAllowMerge) {
		return
	}
	err = executeAPICall(ctx, "/v1/affiliation/merge_all")
	return
}

func mapOrgNames(ctx *lib.Ctx) (err error) {
	if ctx.SkipOrgMap || ctx.OnlyP2O || (ctx.DryRun && !ctx.DryRunAllowOrgMap) {
		return
	}
	err = executeAPICall(ctx, "/v1/affiliation/map_org_names")
	return
}

func detAffRange(ctx *lib.Ctx) (err error) {
	if !ctx.RunDetAffRange || ctx.OnlyP2O || (ctx.DryRun && !ctx.DryRunAllowDetAffRange) {
		return
	}
	err = executeAPICall(ctx, "/v1/affiliation/det_aff_range")
	return
}

func main() {
	var ctx lib.Ctx
	dtStart := time.Now()
	ctx.Init()
	if ctx.DryRun {
		lib.Printf("Running in dry-run mode\n")
	}
	if ctx.OnlyValidate {
		validateFixtureFiles(&ctx, getFixtures(&ctx))
	} else {
		err := ensureGrimoireStackAvail(&ctx)
		if err != nil {
			lib.Fatalf("Grimoire stack not available: %+v\n", err)
		}
		go finishAfterTimeout(ctx)
		processFixtureFiles(&ctx, getFixtures(&ctx))
		err = hideEmails(&ctx)
		if err != nil {
			lib.Printf("Hide emails result: %+v\n", err)
		}
		err = mergeAll(&ctx)
		if err != nil {
			lib.Printf("Merge profiles result: %+v\n", err)
		}
		err = mapOrgNames(&ctx)
		if err != nil {
			lib.Printf("Map organization names result: %+v\n", err)
		}
		err = detAffRange(&ctx)
		if err != nil {
			lib.Printf("Detect affiliations ranges result: %+v\n", err)
		}
		dtEnd := time.Now()
		lib.Printf("Sync time: %v\n", dtEnd.Sub(dtStart))
	}
}
