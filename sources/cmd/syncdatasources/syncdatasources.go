package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"math"
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

	lib "github.com/LF-Engineering/sync-data-sources/sources"
	"github.com/google/go-github/v38/github" // with go mod enabled

	// "github.com/google/go-github/github" // with go mod disabled

	jsoniter "github.com/json-iterator/go"
	yaml "gopkg.in/yaml.v2"
)

var (
	randInitOnce      sync.Once
	gInfoExternal     func()
	gAliasesFunc      func()
	gAliasesMtx       *sync.Mutex
	gCSVMtx           *sync.Mutex
	gRateMtx          *sync.Mutex
	gToken            string
	gHint             int
	noDropPattern     = regexp.MustCompile(`^(.+-f-.+|.+-earned_media|.+-dads-.+|.+-slack|.+-da-ds-gha-.+|.+-social_media|.+-last-action-date-cache|.+-flat-.+|.+-flat)$`)
	notMissingPattern = regexp.MustCompile(`^.+-github-pull_request.*$`)
	emailRegex        = regexp.MustCompile("^[][a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	// if a given source is not in dadsTasks - it only supports legacy p2o then
	// if entry is true - all endpoints using this DS will use the new dads command
	// if entry is false only items marked via 'dads: true' fixture option will use the new dads command
	// Currently we just have jira, groupsio, git, gerrit, confluence, rocketchat which must be enabled per-project in fixture files
	dadsTasks = map[string]bool{
		lib.Jira:         false,
		lib.GroupsIO:     true,
		lib.Git:          true,
		lib.GitHub:       true,
		lib.Gerrit:       true,
		lib.Confluence:   true,
		lib.RocketChat:   false,
		lib.DockerHub:    true,
		lib.Bugzilla:     true,
		lib.BugzillaRest: true,
		lib.Jenkins:      true,
		lib.GoogleGroups: true,
		lib.Pipermail:    false, // we should enable da-ds
	}
	// dadsEnvDefaults - default da-ds settings (can be overwritten in fixture files)
	dadsEnvDefaults = map[string]map[string]string{
		lib.Jira: {
			// "DA_JIRA_LEGACY_UUID":   "1",
			"DA_JIRA_CATEGORY":      "issue",
			"DA_JIRA_NCPUS":         "16",
			"DA_JIRA_DEBUG":         "1",
			"DA_JIRA_RETRY":         "3",
			"DA_JIRA_NO_SSL_VERIFY": "1",
			"DA_JIRA_MULTI_ORIGIN":  "1",
		},
		lib.GroupsIO: {
			//"DA_GROUPSIO_LEGACY_UUID":   "1",
			"DA_GROUPSIO_CATEGORY":      "message",
			"DA_GROUPSIO_NCPUS":         "8",
			"DA_GROUPSIO_DEBUG":         "1",
			"DA_GROUPSIO_RETRY":         "4",
			"DA_GROUPSIO_NO_SSL_VERIFY": "1",
			"DA_GROUPSIO_MULTI_ORIGIN":  "1",
			"DA_GROUPSIO_SAVE_ARCHIVES": "false",
		},
		lib.Git: {
			///"DA_GIT_LEGACY_UUID":      "1",
			"DA_GIT_CATEGORY": "commit",
			"DA_GIT_NCPUS":    "8",
			"DA_GIT_DEBUG":    "1",
			"DA_GIT_RETRY":    "4",
			//"DA_GIT_PAIR_PROGRAMMING": "false",
		},
		lib.GitHub: {
			///"DA_GITHUB_LEGACY_UUID":   "1",
			// "DA_GITHUB_CATEGORY":      "repository || issue || pull_request",
			"DA_GITHUB_NCPUS": "8",
			"DA_GITHUB_DEBUG": "1",
			"DA_GITHUB_RETRY": "3",
		},
		lib.Gerrit: {
			//"DA_GERRIT_LEGACY_UUID":            "1",
			"DA_GERRIT_CATEGORY":               "review",
			"DA_GERRIT_NCPUS":                  "8",
			"DA_GERRIT_DEBUG":                  "1",
			"DA_GERRIT_RETRY":                  "4",
			"DA_GERRIT_NO_SSL_VERIFY":          "1",
			"DA_GERRIT_DISABLE_HOST_KEY_CHECK": "1",
		},
		lib.Confluence: {
			//"DA_CONFLUENCE_LEGACY_UUID":   "1",
			"DA_CONFLUENCE_CATEGORY":      "historical content",
			"DA_CONFLUENCE_NCPUS":         "16",
			"DA_CONFLUENCE_DEBUG":         "1",
			"DA_CONFLUENCE_RETRY":         "4",
			"DA_CONFLUENCE_MULTI_ORIGIN":  "1",
			"DA_CONFLUENCE_NO_SSL_VERIFY": "1",
		},
		lib.RocketChat: {
			//"DA_ROCKETCHAT_LEGACY_UUID":   "1",
			"DA_ROCKETCHAT_CATEGORY":      "message",
			"DA_ROCKETCHAT_NCPUS":         "6",
			"DA_ROCKETCHAT_DEBUG":         "1",
			"DA_ROCKETCHAT_RETRY":         "4",
			"DA_ROCKETCHAT_NO_SSL_VERIFY": "1",
			"DA_ROCKETCHAT_WAIT_RATE":     "1",
		},
		lib.DockerHub: {
			"DA_DOCKERHUB_HTTP_TIMEOUT":   "60s",
			"DA_DOCKERHUB_NO_INCREMENTAL": "1",
		},
		lib.Jenkins: {
			"DA_JENKINS_HTTP_TIMEOUT":   "60s",
			"DA_JENKINS_NO_INCREMENTAL": "1",
		},
		lib.GoogleGroups: {
			"DA_GOOGLEGROUPS_HTTP_TIMEOUT":   "60s",
			"DA_GOOGLEGROUPS_NO_INCREMENTAL": "1",
		},
		lib.Pipermail: {
			"DA_PIPERMAIL_HTTP_TIMEOUT":   "60s",
			"DA_PIPERMAIL_NO_INCREMENTAL": "1",
		},
	}
)

const (
	cOrigin = "sds"
)

func ensureGrimoireStackAvail(ctx *lib.Ctx) error {
	if ctx.Debug > 0 {
		lib.Printf("Checking grimoire stack availability\n")
	}
	ctx.ExecOutput = true
	home := os.Getenv("HOME")
	dir := home + "/.perceval"
	cmd := exec.Command("mkdir", dir)
	_ = cmd.Run()
	defer func() {
		ctx.ExecOutput = false
	}()
	/*
		dtStart := time.Now()
		info := ""
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
	*/
	return nil
}

func validateFixtureFiles(ctx *lib.Ctx, fixtureFiles []string) {
	// Connect to GitHub
	gctx, gcs := lib.GHClient(ctx)
	gHint = -1

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

func isLowercaseEndpointsNeeded(dsSlug string) bool {
	ary := strings.Split(dsSlug, "/")
	typ := strings.TrimSpace(ary[0])
	if typ == "git" || typ == "github" || typ == "gerrit" {
		return true
	}
	return false
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
	lowerCaseNeeded := isLowercaseEndpointsNeeded(dataSource.Slug)
	ste := make(map[string]lib.Endpoint)
	for _, endpoint := range dataSource.Endpoints {
		name := endpoint.Name
		if lowerCaseNeeded {
			name = strings.ToLower(name)
		}
		endpoint2, ok := ste[name]
		if ok {
			if ctx.Debug == 0 {
				lib.Fatalf("Duplicate name %s in the fixture %s, data source: %s\n", name, fixture.Slug, dataSource.Slug)
			} else {
				lib.Fatalf("Duplicate name %s in the fixture %s datasource %s, endpoints: %+v and %+v, data source: %s, fixture: %+v\n", name, fixture.Slug, dataSource.Slug, endpoint, endpoint2, dataSource, fixture)
			}
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
	if !fixture.AllowEmpty && nEndpoints == 0 && nAliases == 0 {
		lib.Fatalf("Fixture file %s must have at least one endpoint defined in 'endpoints'/'projects' key or at least one alias/view defined in 'aliases' key (or you can set 'allow_empty: true' fixture flag)\n", fixtureFile)
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
	// We consider TasksRE, TasksSkipRE and TasksExtraSkipRE safe - they only decide run or not run at the final task level
	if ctx.FixturesRE != nil || ctx.DatasourcesRE != nil || ctx.ProjectsRE != nil || ctx.EndpointsRE != nil || ctx.FixturesSkipRE != nil || ctx.DatasourcesSkipRE != nil || ctx.ProjectsSkipRE != nil || ctx.EndpointsSkipRE != nil {
		return true
	}
	return false
}

func filterFixture(ctx *lib.Ctx, fixture *lib.Fixture) (drop bool) {
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
		lib.Printf("%s has %d after filter\n", fixture.Fn, n)
	}
	if len(fixture.Aliases) > 0 {
		if n == 0 {
			lib.Printf("%s contains only aliases\n", fixture.Fn)
		}
		return false
	}
	return n == 0
}

func getGitHubClients(gctx context.Context, gc []*github.Client, ds *lib.DataSource) (context.Context, []*github.Client, string) {
	dst := ds.Slug
	ary := strings.Split(dst, "/")
	if len(ary) > 1 {
		dst = ary[0]
	}
	if dst != lib.GitHub && dst != lib.Git {
		return gctx, gc, ""
	}
	for _, cfg := range ds.Config {
		name := cfg.Name
		if name != lib.APIToken {
			continue
		}
		_, ok := cfg.Flags["no_default_tokens"]
		if !ok {
			continue
		}
		value := cfg.Value
		keys := make(map[string]struct{})
		if strings.Contains(value, ",") || strings.Contains(value, "[") || strings.Contains(value, "]") {
			ary := strings.Split(value, ",")
			for _, key := range ary {
				key = strings.Replace(key, "[", "", -1)
				key = strings.Replace(key, "]", "", -1)
				keys[key] = struct{}{}
			}
		} else {
			keys[value] = struct{}{}
		}
		if gRateMtx != nil {
			gRateMtx.Lock()
		}
		gHint = -1
		if gRateMtx != nil {
			gRateMtx.Unlock()
		}
		suff := ""
		for key := range keys {
			suff += key
		}
		rgctx, rgc := lib.GHClientForKeys(keys)
		// fmt.Printf("New GH clients for %v (suff=%s)\n", keys, suff)
		return rgctx, rgc, suff
	}
	return gctx, gc, ""
}

func handleDatasourceSettings(ctx *lib.Ctx, fixtureSlug string, ds *lib.DataSource) {
	if ds.Settings == nil {
		return
	}
	idx := "sds-" + fixtureSlug + "-" + ds.Slug + ds.IndexSuffix
	idx = strings.Replace(idx, "/", "-", -1)
	indices := [2]string{idx, idx + "-raw"}
	for _, index := range indices {
		lib.Printf("Applying %+v settings to '%s'\n", *ds.Settings, index)
		payloadBytes, err := jsoniter.Marshal(*ds.Settings)
		if err != nil {
			lib.Fatalf("json marshall error: %+v, data: %+v", err, *ds.Settings)
		}
		data := `{"settings":` + string(payloadBytes) + `}`
		payloadBytes = []byte(data)
		payloadBody := bytes.NewReader(payloadBytes)
		method := lib.Put
		url := fmt.Sprintf("%s/%s", ctx.ElasticURL, index)
		rurl := fmt.Sprintf("/%s", index)
		req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
		if err != nil {
			lib.Fatalf("new request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lib.Fatalf("do request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
		}
		if resp.StatusCode != 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				lib.Fatalf("ReadAll request error: %+v for %s url: %s, data: %+v", err, method, rurl, data)
			}
			_ = resp.Body.Close()
			if resp.StatusCode == 400 && strings.Contains(string(body), "resource_already_exists_exception") {
				lib.Printf("Index '%s' already exists, failed to apply %+v settings, continuying\n", index, data)
				continue
			}
			lib.Fatalf("Method:%s url:%s status:%d data:%+v\n%s", method, rurl, resp.StatusCode, data, body)
		}
		lib.Printf("Index '%s' created with %+v settings\n", index, data)
	}
}

func postprocessFixture(igctx context.Context, igc []*github.Client, ctx *lib.Ctx, fixture *lib.Fixture) {
	cache := make(map[string][]string)
	for i, dataSource := range fixture.DataSources {
		gctx, gc, cacheSuff := getGitHubClients(igctx, igc, &dataSource)
		handleDatasourceSettings(ctx, fixture.Slug, &dataSource)
		for _, projectData := range dataSource.Projects {
			project := projectData.Name
			projectP2O := projectData.P2O
			projectNoOrigin := projectData.NoOrigin
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
				projNoOrigin := projectNoOrigin
				if rawEndpoint.ProjectP2O != nil {
					projP2O = rawEndpoint.ProjectP2O
				}
				if rawEndpoint.ProjectNoOrigin != nil {
					projNoOrigin = rawEndpoint.ProjectNoOrigin
				}
				name := rawEndpoint.Name
				if projP2O != nil && *projP2O {
					name += ":::" + proj
				}
				fixture.DataSources[i].RawEndpoints = append(
					fixture.DataSources[i].RawEndpoints,
					lib.RawEndpoint{
						Name:              name,
						Project:           proj,
						ProjectP2O:        projP2O,
						ProjectNoOrigin:   projNoOrigin,
						Flags:             rawEndpoint.Flags,
						Skip:              rawEndpoint.Skip,
						Only:              rawEndpoint.Only,
						Timeout:           rawEndpoint.Timeout,
						CopyFrom:          rawEndpoint.CopyFrom,
						AffiliationSource: rawEndpoint.AffiliationSource,
						PairProgramming:   rawEndpoint.PairProgramming,
						Groups:            rawEndpoint.Groups[:],
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
			pno := false
			if rawEndpoint.ProjectNoOrigin != nil {
				pno = *(rawEndpoint.ProjectNoOrigin)
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
						Name:              name,
						Project:           rawEndpoint.Project,
						ProjectP2O:        p2o,
						ProjectNoOrigin:   pno,
						Projects:          rawEndpoint.Projects,
						Timeout:           tmout,
						CopyFrom:          rawEndpoint.CopyFrom,
						AffiliationSource: rawEndpoint.AffiliationSource,
						PairProgramming:   rawEndpoint.PairProgramming,
						Groups:            rawEndpoint.Groups[:],
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
			for k, group := range rawEndpoint.Groups {
				for _, skip := range group.Skip {
					skipRE, err := regexp.Compile(skip)
					lib.FatalOnError(err)
					group.SkipREs = append(group.SkipREs, skipRE)
				}
				fixture.DataSources[i].RawEndpoints[j].Groups[k].SkipREs = group.SkipREs
				for _, only := range group.Only {
					onlyRE, err := regexp.Compile(only)
					lib.FatalOnError(err)
					group.OnlyREs = append(group.OnlyREs, onlyRE)
				}
				fixture.DataSources[i].RawEndpoints[j].Groups[k].OnlyREs = group.OnlyREs
			}
			handleNoData := func() {
				fixture.DataSources[i].Endpoints = append(fixture.DataSources[i].Endpoints, lib.Endpoint{Name: name, Dummy: true})
			}
			handleRate := func() (aHint int, canCache bool) {
				// fmt.Printf("handle rate called: %s\n", fixture.Native)
				h, _, rem, wait := lib.GetRateLimits(gctx, ctx, gc, true)
				for {
					lib.Printf("Checking token %d %+v %+v\n", h, rem, wait)
					if rem[h] <= 100 {
						lib.Printf("All GH API tokens are overloaded, maximum points %d, waiting %+v\n", rem[h], wait[h])
						time.Sleep(time.Duration(1) * time.Second)
						time.Sleep(wait[h])
						h, _, rem, wait = lib.GetRateLimits(gctx, ctx, gc, true)
						continue
					}
					if rem[h] >= 2500 {
						canCache = true
					}
					break
				}
				aHint = h
				lib.Printf("Found usable token %d/%d/%v, cache enabled: %v\n", aHint, rem[h], wait[h], canCache)
				return
			}
			isAbuse := func(e error) bool {
				if e == nil {
					return false
				}
				errStr := e.Error()
				return strings.Contains(errStr, "403 You have triggered an abuse detection mechanism") || strings.Contains(errStr, "403 API rate limit")
			}
			switch epType {
			case "slack_bot_channels":
				token := ""
				_, ok := rawEndpoint.Flags["is_token"]
				if ok {
					token = rawEndpoint.Name
				} else {
					for _, cfg := range dataSource.Config {
						if cfg.Name == lib.APIToken {
							token = cfg.Value
							break
						}
					}
				}
				if token == "" {
					lib.Printf("Error getting slack token\n")
					continue
				}
				rtoken := token[len(token)-len(token)/4:]
				lib.AddRedacted(token, true)
				ids, ok1 := cache[epType+"i"+token]
				channels, ok2 := cache[epType+"c"+token]
				if !ok1 || !ok2 {
					var err error
					ids, channels, err = lib.GetSlackBotUsersConversation(ctx, token)
					if err != nil {
						lib.Printf("Error getting slack conversations list for: %s: error: %+v\n", rtoken, err)
						continue
					}
					cache[epType+"i"+token] = ids
					cache[epType+"c"+token] = channels
				}
				if ctx.Debug > 0 {
					lib.Printf("Slack %s ids: %+v, channels: %+v\n", rtoken, ids, channels)
				}
				for idx, id := range ids {
					channel := channels[idx]
					included1, state1 := lib.EndpointIncluded(ctx, &rawEndpoint, id)
					included2, state2 := lib.EndpointIncluded(ctx, &rawEndpoint, channel)
					// If neither id nor channel were included by 'only' condition
					// And id or channel were excluded by 'skip condition - then skip that endpoint
					if state1 != 1 && state2 != 1 && (!included1 || !included2) {
						if ctx.Debug > 0 {
							lib.Printf("Skipped slack((%d,%v),(%d,%v)): %s, id:%s, channel:%s\n", state1, included1, state2, included2, rtoken, id, channel)
						}
						continue
					}
					if p2o && rawEndpoint.Project != "" {
						id += ":::" + rawEndpoint.Project
					}
					if ctx.Debug > 0 {
						lib.Printf("Added slack((%d,%v),(%d,%v)): %s, id:%s, channel:%s\n", state1, included1, state2, included2, rtoken, id, channel)
					}
					fixture.DataSources[i].Endpoints = append(
						fixture.DataSources[i].Endpoints,
						lib.Endpoint{
							Name:              id,
							Project:           rawEndpoint.Project,
							ProjectP2O:        p2o,
							ProjectNoOrigin:   pno,
							Projects:          rawEndpoint.Projects,
							Timeout:           tmout,
							CopyFrom:          rawEndpoint.CopyFrom,
							AffiliationSource: rawEndpoint.AffiliationSource,
							PairProgramming:   rawEndpoint.PairProgramming,
							Groups:            rawEndpoint.Groups[:],
						},
					)
				}
				if len(ids) == 0 {
					handleNoData()
				}
			case "gerrit_org":
				gerrit := strings.TrimSpace(rawEndpoint.Name)
				projects, ok1 := cache[epType+"p"+gerrit]
				repos, ok2 := cache[epType+"r"+gerrit]
				if !ok1 || !ok2 {
					var err error
					projects, repos, err = lib.GetGerritRepos(ctx, gerrit)
					if err != nil {
						lib.Printf("Error getting gerrit repos list for: %s: error: %+v\n", gerrit, err)
						continue
					}
					cache[epType+"p"+gerrit] = projects
					cache[epType+"r"+gerrit] = repos
				}
				if ctx.Debug > 0 {
					lib.Printf("Gerrit %s repos: %+v, projects: %+v\n", gerrit, repos, projects)
				}
				for idx, repo := range repos {
					included, _ := lib.EndpointIncluded(ctx, &rawEndpoint, repo)
					if !included {
						continue
					}
					if p2o && rawEndpoint.Project != "" {
						repo += ":::" + rawEndpoint.Project
					}
					gPrj := projects[idx]
					prj := rawEndpoint.Project
					if prj == "" && gPrj != "" {
						prj = gPrj
					}
					if ctx.Debug > 0 {
						lib.Printf("gerrit: %s, project: %s, repo: %s\n", gerrit, prj, repo)
					}
					// fmt.Printf("\"%s\",\"%s\",\"%s\"\n", gerrit, prj, repo)
					fixture.DataSources[i].Endpoints = append(
						fixture.DataSources[i].Endpoints,
						lib.Endpoint{
							Name:              repo,
							Project:           prj,
							ProjectP2O:        p2o,
							ProjectNoOrigin:   pno,
							Projects:          rawEndpoint.Projects,
							Timeout:           tmout,
							CopyFrom:          rawEndpoint.CopyFrom,
							AffiliationSource: rawEndpoint.AffiliationSource,
							PairProgramming:   rawEndpoint.PairProgramming,
							Groups:            rawEndpoint.Groups[:],
						},
					)
				}
				if len(repos) == 0 {
					handleNoData()
				}
			case "dockerhub_org":
				dockerhubOwner := strings.TrimSpace(rawEndpoint.Name)
				repos, ok := cache[epType+dockerhubOwner]
				if !ok {
					var err error
					repos, err = lib.GetDockerHubRepos(ctx, dockerhubOwner)
					if err != nil {
						lib.Printf("Error getting dockerhub repos list for: %s: error: %+v\n", dockerhubOwner, err)
						continue
					}
					cache[epType+dockerhubOwner] = repos
				}

				for _, repo := range repos {
					included, _ := lib.EndpointIncluded(ctx, &rawEndpoint, repo)
					if !included {
						continue
					}
					if p2o && rawEndpoint.Project != "" {
						repo += ":::" + rawEndpoint.Project
					}

					prj := rawEndpoint.Project

					if ctx.Debug > 0 {
						lib.Printf("Dockerhub Owner: %s, repo: %s\n", dockerhubOwner, repo)
					}

					fixture.DataSources[i].Endpoints = append(
						fixture.DataSources[i].Endpoints,
						lib.Endpoint{
							Name:              repo,
							Project:           prj,
							ProjectP2O:        p2o,
							ProjectNoOrigin:   pno,
							Projects:          rawEndpoint.Projects,
							Timeout:           tmout,
							CopyFrom:          rawEndpoint.CopyFrom,
							AffiliationSource: rawEndpoint.AffiliationSource,
							PairProgramming:   rawEndpoint.PairProgramming,
							Groups:            rawEndpoint.Groups[:],
						},
					)
				}
				if len(repos) == 0 {
					handleNoData()
				}
			case "rocketchat_server":
				srv := strings.TrimSpace(rawEndpoint.Name)
				channels, ok := cache[epType+srv]
				if !ok {
					token := ""
					uid := ""
					for _, cfg := range dataSource.Config {
						if cfg.Name == lib.APIToken {
							token = cfg.Value
							lib.AddRedacted(token, true)
						} else if cfg.Name == "user-id" {
							uid = cfg.Value
							lib.AddRedacted(uid, true)
						}
						if uid != "" && token != "" {
							break
						}
					}
					if uid == "" || token == "" {
						lib.Printf("Error getting rocket chat uid or token: (%s,%s)\n", uid, token)
						continue
					}
					var err error
					channels, err = lib.GetRocketChatChannels(ctx, srv, token, uid)
					if err != nil {
						lib.Printf("Error getting channels list for rocketchat server: %s: error: %+v\n", srv, err)
						continue
					}
					cache[epType+srv] = channels
				}
				if ctx.Debug > 0 {
					lib.Printf("RocketChat srv %s channels: %+v\n", srv, channels)
				}
				for _, channel := range channels {
					included, _ := lib.EndpointIncluded(ctx, &rawEndpoint, channel)
					if !included {
						continue
					}
					name := srv + " " + channel
					if p2o && rawEndpoint.Project != "" {
						name += ":::" + rawEndpoint.Project
					}
					fixture.DataSources[i].Endpoints = append(
						fixture.DataSources[i].Endpoints,
						lib.Endpoint{
							Name:              name,
							Project:           rawEndpoint.Project,
							ProjectP2O:        p2o,
							ProjectNoOrigin:   pno,
							Projects:          rawEndpoint.Projects,
							Timeout:           tmout,
							CopyFrom:          rawEndpoint.CopyFrom,
							AffiliationSource: rawEndpoint.AffiliationSource,
							PairProgramming:   rawEndpoint.PairProgramming,
							Groups:            rawEndpoint.Groups[:],
						},
					)
				}
				if len(channels) == 0 {
					handleNoData()
				}
			case lib.GitHubOrg:
				includeForksStr, includeForks := rawEndpoint.Flags["include_forks"]
				if includeForks {
					includeForks = lib.StringToBool(includeForksStr)
				}
				// fmt.Printf("github_org called for %v\n", fixture.Native)
				var aHint int
				if gRateMtx != nil {
					gRateMtx.Lock()
				}
				if gHint < 0 {
					var canCache bool
					aHint, canCache = handleRate()
					if canCache {
						gHint = aHint
					}
				} else {
					aHint = gHint
				}
				if gRateMtx != nil {
					gRateMtx.Unlock()
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
				cacheKey := epType + org + cacheSuff
				repos, ok := cache[cacheKey]
				if !ok {
					if ctx.Debug > 0 {
						lib.Printf("Repositories.ListByOrg(%s)\n", org)
					}
					opt := &github.RepositoryListByOrgOptions{Type: "public"} // can also use "all"
					opt.PerPage = 100
					repos = []string{}
					retried := false
					for {
						// fmt.Printf("non-cache call %s %v (%d tokens)\n", org, fixture.Native, len(gc))
						repositories, response, err := gc[aHint].Repositories.ListByOrg(gctx, org, opt)
						if err != nil && !retried {
							lib.Printf("Error getting repositories list for org: %s: response: %+v, error: %+v, retrying rate (hint %d, opt %+v)\n", org, response, err, aHint, opt)
							if isAbuse(err) {
								sleepFor := 30 + rand.Intn(30)
								lib.Printf("GitHub detected abuse, waiting for %ds\n", sleepFor)
								time.Sleep(time.Duration(sleepFor) * time.Second)
								aHint, _ = handleRate()
							} else {
								aHint, _ = handleRate()
								retried = true
							}
							continue
						}
						if err != nil {
							lib.Printf("Error getting repositories list for org: %s: response: %+v, error: %+v\n", org, response, err)
							break
						}
						for _, repo := range repositories {
							if repo.Name == nil {
								continue
							}
							if !includeForks && repo.Fork != nil && *repo.Fork {
								if ctx.Debug > 0 {
									lib.Printf("Skipping fork: org:%s, repo:%s\n", org, *repo.Name)
								}
								continue
							}
							name := root + "/" + org + "/" + *(repo.Name)
							repos = append(repos, name)
						}
						if response.NextPage == 0 {
							break
						}
						opt.Page = response.NextPage
					}
					cache[cacheKey] = repos
				}
				if ctx.Debug > 0 {
					lib.Printf("Org %s repos: %+v\n", org, repos)
				}
				for _, repo := range repos {
					included, _ := lib.EndpointIncluded(ctx, &rawEndpoint, repo)
					if !included {
						continue
					}
					name := repo
					if p2o && rawEndpoint.Project != "" {
						name += ":::" + rawEndpoint.Project
					}
					fixture.DataSources[i].Endpoints = append(
						fixture.DataSources[i].Endpoints,
						lib.Endpoint{
							Name:              name,
							Project:           rawEndpoint.Project,
							ProjectP2O:        p2o,
							ProjectNoOrigin:   pno,
							Projects:          rawEndpoint.Projects,
							Timeout:           tmout,
							CopyFrom:          rawEndpoint.CopyFrom,
							AffiliationSource: rawEndpoint.AffiliationSource,
							PairProgramming:   rawEndpoint.PairProgramming,
							Groups:            rawEndpoint.Groups[:],
						},
					)
				}
				if len(repos) == 0 {
					handleNoData()
				}
			case lib.GitHubUser:
				includeForksStr, includeForks := rawEndpoint.Flags["include_forks"]
				if includeForks {
					includeForks = lib.StringToBool(includeForksStr)
				}
				var aHint int
				if gRateMtx != nil {
					gRateMtx.Lock()
				}
				if gHint < 0 {
					var canCache bool
					aHint, canCache = handleRate()
					if canCache {
						gHint = aHint
					}
				} else {
					aHint = gHint
				}
				if gRateMtx != nil {
					gRateMtx.Unlock()
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
				cacheKey := epType + user + cacheSuff
				repos, ok := cache[cacheKey]
				if !ok {
					if ctx.Debug > 0 {
						lib.Printf("Repositories.List(%s)\n", user)
					}
					opt := &github.RepositoryListOptions{Type: "public"}
					opt.PerPage = 100
					repos = []string{}
					retried := false
					for {
						repositories, response, err := gc[aHint].Repositories.List(gctx, user, opt)
						if err != nil && !retried {
							lib.Printf("Error getting repositories list for user: %s: response: %+v, error: %+v, retrying rate\n", user, response, err)
							if isAbuse(err) {
								sleepFor := 30 + rand.Intn(30)
								lib.Printf("GitHub detected abuse, waiting for %ds\n", sleepFor)
								time.Sleep(time.Duration(sleepFor) * time.Second)
								aHint, _ = handleRate()
							} else {
								aHint, _ = handleRate()
								retried = true
							}
							continue
						}
						if err != nil {
							lib.Printf("Error getting repositories list for user: %s: response: %+v, error: %+v\n", user, response, err)
							break
						}
						for _, repo := range repositories {
							if repo.Name == nil {
								continue
							}
							if !includeForks && repo.Fork != nil && *repo.Fork {
								lib.Printf("Skipping fork: user:%s, repo:%s\n", user, *repo.Name)
								continue
							}
							name := root + "/" + user + "/" + *(repo.Name)
							repos = append(repos, name)
						}
						if response.NextPage == 0 {
							break
						}
						opt.Page = response.NextPage
					}
					cache[cacheKey] = repos
				}
				if ctx.Debug > 0 {
					lib.Printf("User %s repos: %+v\n", user, repos)
				}
				for _, repo := range repos {
					included, _ := lib.EndpointIncluded(ctx, &rawEndpoint, repo)
					if !included {
						continue
					}
					name := repo
					if p2o && rawEndpoint.Project != "" {
						name += ":::" + rawEndpoint.Project
					}
					fixture.DataSources[i].Endpoints = append(
						fixture.DataSources[i].Endpoints,
						lib.Endpoint{
							Name:              name,
							Project:           rawEndpoint.Project,
							ProjectP2O:        p2o,
							ProjectNoOrigin:   pno,
							Projects:          rawEndpoint.Projects,
							Timeout:           tmout,
							CopyFrom:          rawEndpoint.CopyFrom,
							AffiliationSource: rawEndpoint.AffiliationSource,
							PairProgramming:   rawEndpoint.PairProgramming,
							Groups:            rawEndpoint.Groups[:],
						},
					)
				}
				if len(repos) == 0 {
					handleNoData()
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
						Name:              name,
						Project:           rawEndpoint.Project,
						ProjectP2O:        p2o,
						ProjectNoOrigin:   pno,
						Projects:          rawEndpoint.Projects,
						Timeout:           tmout,
						CopyFrom:          rawEndpoint.CopyFrom,
						AffiliationSource: rawEndpoint.AffiliationSource,
						PairProgramming:   rawEndpoint.PairProgramming,
						Groups:            rawEndpoint.Groups[:],
					},
				)
				continue
			}
		}
	}
	for ai, alias := range fixture.Aliases {
		var idxSlug string
		if strings.HasPrefix(alias.From, "bitergia-") || strings.HasPrefix(alias.From, "pattern:") || strings.HasPrefix(alias.From, "postprocess-") {
			idxSlug = alias.From
		} else {
			idxSlug = "sds-" + alias.From
		}
		if !strings.HasPrefix(alias.From, "pattern:") {
			idxSlug = strings.Replace(idxSlug, "/", "-", -1)
		}
		fixture.Aliases[ai].From = idxSlug
		for ti, to := range alias.To {
			idxSlug := ""
			if strings.HasPrefix(to, "postprocess") {
				idxSlug = "postprocess-sds-" + strings.TrimPrefix(to, "postprocess/")
			} else {
				idxSlug = "sds-" + to
			}
			idxSlug = strings.Replace(idxSlug, "/", "-", -1)
			fixture.Aliases[ai].To[ti] = idxSlug

		}
		for vi, v := range alias.Views {
			idxSlug := "sds-" + v.Name
			idxSlug = strings.Replace(idxSlug, "/", "-", -1)
			fixture.Aliases[ai].Views[vi].Name = idxSlug
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
	if filterFixture(ctx, &fixture) {
		fixture.Disabled = true
		return
	}
	if ctx.Debug > 0 {
		lib.Printf("Post-processed and filtered %s fixture: %+v\n", fixtureFile, fixture)
	}
	validateFixture(ctx, &fixture, fixtureFile)
	return
}

// todo: move it from here
func getFlagByName(name string, flags []lib.Config) string {
	for _, flag := range flags {
		if flag.Name == name {
			return flag.Value
		}
	}
	return ""
}

func generateFoundationFAliases(ctx *lib.Ctx, pfixtures *[]lib.Fixture) {
	// - skip */shared, */common ? - NO? just include only datasources where there are at least one endpoints
	// - earned_media ? - NO problem? - it always has no endpoints
	// - skip DS es without endpoints? - YES?
	// - detect alias2alias cases
	// - The LF foundation-f - lf? - Assumed lf-f
	// - maintain (entire f-f - add/delete/update), subproject (add/delete/update) - YES
	// - dockerhub sds- -> postprocess-sds- - YES
	// - github/pull_request -> github/issue - YES
	// - index sufixes (possibly different)
	if ctx.OnlyP2O || ctx.SkipFAliases || (ctx.DryRun && !ctx.DryRunAllowFAliases) {
		lib.Printf("Skipping f-aliases generation\n")
		return
	}
	fixtures := *pfixtures
	// The Linux Foundation foundation slug
	lff := "lf"
	// DockerHub postprocessed index prefix (possibly others in the future)
	ppPrefix := "postprocess-"
	// Foundation-f aliases will look like aliasprefix-founation-f-datasource
	aliasPrefix := "sds-"
	// aliasPrefix := "sds-"
	// It will ook for data in dataprefix-foundation-project-datasurce-index-prefix
	dataPrefix := "sds-"
	maxThreads := 12
	// m[foundation][project][ds] = [full_ds]
	m := map[string]map[string]map[string]string{}
	adsm := map[string]struct{}{}
	var (
		f    string
		p    string
		fpds string
	)
	for _, fixture := range fixtures {
		slug := fixture.Slug
		ary := strings.Split(slug, "/")
		if len(ary) < 2 {
			f = lff
			p = strings.TrimSpace(ary[0])
		} else {
			f = strings.TrimSpace(ary[0])
			p = strings.TrimSpace(ary[1])
		}
		_, ok := m[f]
		if !ok {
			m[f] = map[string]map[string]string{}
		}
		_, ok = m[f][p]
		if !ok {
			m[f][p] = map[string]string{}
		}
		for _, ds := range fixture.DataSources {
			s := strings.Replace(ds.Slug, "/", "-", -1)
			fs := ds.FullSlug
			if s == "github-pull_request" {
				s = "github-issue"
				fs = strings.Replace(fs, "github-pull_request", "github-issue", -1)
			}
			if len(ds.Endpoints) > 0 {
				m[f][p][s] = fs
				adsm[strings.Replace(s, "/", "-", -1)] = struct{}{}
			}
		}
	}
	ads := []string{}
	for ds := range adsm {
		ads = append(ads, ds)
	}
	sort.Slice(ads, func(i, j int) bool {
		return len(ads[i]) > len(ads[j])
	})
	lib.Printf("Foundation-f all data source types: %+v\n", ads)
	for _, fixture := range fixtures {
		slug := fixture.Slug
		ary := strings.Split(slug, "/")
		if len(ary) < 2 {
			f = lff
			p = strings.TrimSpace(ary[0])
		} else {
			f = strings.TrimSpace(ary[0])
			p = strings.TrimSpace(ary[1])
		}
		for _, alias := range fixture.Aliases {
			for _, to := range alias.To {
				if to == "" || strings.HasSuffix(to, "-raw") {
					continue
				}
				an := strings.Replace(to, "/", "-", -1)
				found := false
				for _, ds := range ads {
					if strings.Contains(an, ds) && (f == lff || strings.Contains(an, f)) {
						m[f][p][ds] = "!" + an
						if ctx.Debug > 0 {
							lib.Printf("Foundation-f deduced %s,%s,%s -> %s from %s\n", f, p, ds, an, to)
						}
						found = true
						break
					}
				}
				if !found {
					lib.Printf("Foundation-f cannot deduce data source type from alias name %s in %s/%s, skipping\n", to, f, p)
				}
			}
		}
	}
	ks := []string{}
	config := map[string][]string{}
	dst := map[string]struct{}{}
	src := map[string]struct{}{}
	for f, ps := range m {
		if len(ps) == 0 {
			lib.Printf("foundation: %s has no projects\n", f)
			continue
		}
		dss := map[string]struct{}{}
		for _, d := range ps {
			for s := range d {
				dss[s] = struct{}{}
			}
		}
		if len(dss) == 0 {
			lib.Printf("foundation: %s has no data sources\n", f)
			continue
		}
		for ds := range dss {
			aliases := []string{}
			ppAliases := []string{}
			for p, d := range ps {
				fs, ok := d[ds]
				if !ok {
					continue
				}
				if strings.HasPrefix(fs, "!") {
					fs = fs[1:]
					if strings.HasPrefix(fs, dataPrefix) {
						fpds = fs
					} else {
						fpds = dataPrefix + fs
					}
				} else {
					if f == lff {
						fpds = dataPrefix + p + "-" + fs
					} else {
						fpds = dataPrefix + f + "-" + p + "-" + fs
					}
				}
				aliases = append(aliases, fpds)
				src[fpds] = struct{}{}
				if ds == "dockerhub" {
					fpds = "postprocess-" + fpds
					ppAliases = append(ppAliases, fpds)
					src[fpds] = struct{}{}
				}
			}
			nAliases := len(aliases)
			if nAliases == 0 {
				lib.Printf("foundation: %s, data source: %s has no data sources\n", f, ds)
				continue
			}
			if nAliases > 1 {
				sort.Strings(aliases)
			}
			k := aliasPrefix + f + "-f-" + ds
			ks = append(ks, k)
			config[k] = aliases
			dst[k] = struct{}{}
			// postprocess- aliases only for dockerhub)
			nPPAliases := len(ppAliases)
			if nPPAliases == 0 {
				continue
			}
			if nPPAliases > 1 {
				sort.Strings(ppAliases)
			}
			k = ppPrefix + aliasPrefix + f + "-f-" + ds
			ks = append(ks, k)
			config[k] = ppAliases
			dst[k] = struct{}{}
		}
	}
	nKs := len(ks)
	if nKs > 1 {
		sort.Strings(ks)
	}
	for _, k := range ks {
		aliases, ok := config[k]
		if !ok {
			lib.Printf("key not found: %s, should not happen\n", k)
			continue
		}
		if ctx.Debug > 0 {
			lib.Printf("Foundation-f alias/indices %s: %+v\n", k, aliases)
		}
	}
	lib.Printf("Foundation-f aliasing needs %d aliases that point to %d indices\n", len(dst), len(src))
	gotI := make(map[string]struct{})
	gotA := make(map[string]struct{})
	checkIndicesAndAliases := func() (err error) {
		missing := []string{}
		extra := []string{}
		method := lib.Get
		url := fmt.Sprintf("%s/_cat/indices?format=json", ctx.ElasticURL)
		rurl := "/_cat/indices?format=json"
		var req *http.Request
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			lib.Printf("New request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		var resp *http.Response
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			lib.Printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode != 200 {
			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				lib.Printf("ReadAll request error: %+v for %s url: %s\n", err, method, rurl)
				return
			}
			lib.Printf("Method:%s url:%s status:%d\n%s\n", method, rurl, resp.StatusCode, body)
			return
		}
		indices := []lib.EsIndex{}
		err = jsoniter.NewDecoder(resp.Body).Decode(&indices)
		if err != nil {
			lib.Printf("JSON decode error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		for _, index := range indices {
			sIndex := index.Index
			if strings.HasPrefix(sIndex, ppPrefix) {
				sIndex = sIndex[len(ppPrefix):]
			}
			if !strings.HasPrefix(sIndex, dataPrefix) && !strings.HasPrefix(sIndex, aliasPrefix) {
				continue
			}
			gotI[index.Index] = struct{}{}
		}
		method = lib.Get
		url = fmt.Sprintf("%s/_cat/aliases?format=json", ctx.ElasticURL)
		rurl = "/_cat/aliases?format=json"
		req, err = http.NewRequest(method, url, nil)
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
			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				lib.Printf("ReadAll request error: %+v for %s url: %s\n", err, method, rurl)
				return
			}
			lib.Printf("Method:%s url:%s status:%d\n%s\n", method, rurl, resp.StatusCode, body)
			return
		}
		aliases := []lib.EsAlias{}
		err = jsoniter.NewDecoder(resp.Body).Decode(&aliases)
		if err != nil {
			lib.Printf("JSON decode error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		for _, alias := range aliases {
			sAlias := alias.Alias
			if strings.HasPrefix(sAlias, ppPrefix) {
				sAlias = sAlias[len(ppPrefix):]
			}
			if !strings.HasPrefix(sAlias, dataPrefix) && !strings.HasPrefix(sAlias, aliasPrefix) {
				continue
			}
			gotA[alias.Alias] = struct{}{}
		}
		lib.Printf("Detected %d indices and %d aliases with matching data/index prefix\n", len(gotI), len(gotA))
		for alias := range dst {
			_, ok := gotA[alias]
			if !ok {
				// Note: Skip PRs
				if !notMissingPattern.MatchString(alias) {
					missing = append(missing, alias)
				}
			}
		}
		for alias := range gotA {
			if !strings.Contains(alias, "-f-") || !strings.HasPrefix(alias, aliasPrefix) {
				continue
			}
			_, ok := dst[alias]
			if !ok {
				extra = append(extra, alias)
			}
		}
		sort.Strings(missing)
		sort.Strings(extra)
		if len(missing) > 0 {
			lib.Printf("Foundation-f missing the following aliases %d: %s\n", len(missing), strings.Join(missing, ", "))
		}
		if len(extra) == 0 {
			lib.Printf("No foundation-f aliases to drop, environment clean\n")
			return
		}
		lib.Printf("Foundation-f aliases that should probably be dropped (%d): %s\n", len(extra), strings.Join(extra, ", "))
		return
	}
	err := checkIndicesAndAliases()
	if err != nil {
		lib.Printf("WARNING: maintain foundation-f indices retrned error: %+v, continuying anyway\n", err)
	}
	map2SortedString := func(mp map[string]struct{}) string {
		out := []string{}
		for m := range mp {
			out = append(out, m)
		}
		if len(out) > 1 {
			sort.Strings(out)
		}
		return strings.Join(out, ", ")
	}
	getAliasItems := func(alias string) (items map[string]struct{}, err error) {
		method := lib.Get
		url := fmt.Sprintf("%s/_cat/aliases/%s?format=json", ctx.ElasticURL, alias)
		rurl := "/_cat/aliases/" + alias + "?format=json"
		var req *http.Request
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			lib.Printf("New request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		var resp *http.Response
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			lib.Printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode != 200 {
			// There is no alias yet, no problem, so it has no items
			if resp.StatusCode == 404 {
				err = nil
				return
			}
			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				lib.Printf("ReadAll request error: %+v for %s url: %s\n", err, method, rurl)
				return
			}
			lib.Printf("Method:%s url:%s status:%d\n%s\n", method, rurl, resp.StatusCode, body)
			return
		}
		itms := []lib.EsAlias{}
		err = jsoniter.NewDecoder(resp.Body).Decode(&itms)
		if err != nil {
			lib.Printf("JSON decode error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		items = make(map[string]struct{})
		for _, itm := range itms {
			sIndex := itm.Index
			items[sIndex] = struct{}{}
		}
		return
	}
	processFFAlias := func(ch chan struct{}, alias string, itemsAry []string) {
		if ch != nil {
			defer func() {
				ch <- struct{}{}
			}()
		}
		got, err := getAliasItems(alias)
		if err != nil {
			lib.Printf("Foundation-f failed to get alias indices for %s: %v, continuying\n", alias, err)
		}
		if ctx.Debug > 0 && len(got) > 0 {
			fmt.Printf("Foundation-f current alias %s points to %+v\n", alias, got)
		}
		items := map[string]struct{}{}
		for _, item := range itemsAry {
			_, isAlias := gotA[item]
			_, isIndex := gotI[item]
			if !isAlias && !isIndex {
				if ctx.Debug > 0 {
					lib.Printf("Foundation-f %s item %s is neither an index or an alias, skipping\n", alias, item)
				}
				continue
			}
			if isAlias {
				gotItems, err := getAliasItems(item)
				if err != nil {
					lib.Printf("Foundation-f failed to get alias %s item %s indices: %v, continuying\n", alias, item, err)
				}
				if ctx.Debug > 0 {
					lib.Printf("Foundation-f %s item %s is an alias that points to %s\n", alias, item, map2SortedString(gotItems))
				}
				for itm := range gotItems {
					items[itm] = struct{}{}
				}
				continue
			}
			items[item] = struct{}{}
		}
		if ctx.Debug > 0 && len(itemsAry) != len(items) {
			fmt.Printf("Foundation-f after processing items of %s, number of items changed from %d to %d\n", alias, len(itemsAry), len(items))
		}
		// Note: Consider skipping PRs
		missing := []string{}
		extra := []string{}
		for alias := range items {
			_, ok := got[alias]
			if !ok {
				missing = append(missing, alias)
			}
		}
		for alias := range got {
			_, ok := items[alias]
			if !ok {
				extra = append(extra, alias)
			}
		}
		sort.Strings(missing)
		sort.Strings(extra)
		nMissing := len(missing)
		nExtra := len(extra)
		if nMissing > 0 {
			lib.Printf("Foundation-f alias %s missing %d indices: %s\n", alias, len(missing), strings.Join(missing, ", "))
		}
		if nExtra > 0 {
			lib.Printf("Foundation-f alias %s has extra %d indices: %s\n", alias, len(extra), strings.Join(extra, ", "))
		}
		if ctx.Debug > 0 {
			lib.Printf("===== %s =====>\ngot:   %s\nneeds: %s\nmiss:  %s\nextra: %s\n", alias, map2SortedString(got), map2SortedString(items), strings.Join(missing, ", "), strings.Join(extra, ", "))
		}
		if nMissing == 0 && nExtra == 0 {
			return
		}
		payload := `{"actions":[`
		for _, index := range extra {
			payload += `{"remove":{"index":"` + index + `","alias":"` + alias + `"}},`
		}
		for _, index := range missing {
			payload += `{"add":{"index":"` + index + `","alias":"` + alias + `"}},`
		}
		payload = payload[:len(payload)-1] + `]}`
		method := lib.Post
		url := fmt.Sprintf("%s/_aliases", ctx.ElasticURL)
		rurl := "/_aliases"
		payloadBytes := []byte(payload)
		payloadBody := bytes.NewReader(payloadBytes)
		req, err := http.NewRequest(method, url, payloadBody)
		if err != nil {
			lib.Printf("New request error: %+v for %s url: %s, payload: %s\n", err, method, rurl, payload)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lib.Printf("Do request error: %+v for %s url: %s, payload: %s\n", err, method, rurl, payload)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode != 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				lib.Printf("ReadAll request error: %+v for %s url: %s, payload: %s\n", err, method, rurl, payload)
				return
			}
			lib.Printf("Method:%s url:%s payload:%s status:%d\n%s\n", method, rurl, payload, resp.StatusCode, body)
		}
		if ctx.Debug > 0 {
			lib.Printf("Foundation-f: alias configuration changed: %s\n", payload)
		}
		return
	}
	thrN := lib.GetThreadsNum(ctx)
	if thrN > maxThreads {
		thrN = maxThreads
	}
	lib.Printf("%d foundation-f aliases to process using %d threads\n", len(config), thrN)
	if thrN > 1 {
		ch := make(chan struct{})
		nThreads := 0
		for alias, items := range config {
			go processFFAlias(ch, alias, items[:])
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
		for alias, items := range config {
			processFFAlias(nil, alias, items[:])
		}
	}
	lib.Printf("Finished generatig foundation-f aliases\n")
}

func calculateGroups(ctx *lib.Ctx, name string, groupsConfigs []lib.GroupConfig) (groups []string) {
	def := ""
	for _, group := range groupsConfigs {
		if lib.GroupIncluded(ctx, &group, name) {
			if group.Default && group.Name != "" {
				def = group.Name
			}
			if group.Default {
				continue
			}
			if group.Name != "" {
				groups = append(groups, group.Name)
			}
			if group.Self {
				groups = append(groups, name)
			}
		}
	}
	if def != "" && len(groups) == 0 {
		groups = append(groups, def)
	}
	return
}

func processFixtureFiles(ctx *lib.Ctx, fixtureFiles []string) {
	// Connect to GitHub
	gctx, gcs := lib.GHClient(ctx)
	gHint = -1
	// Get number of CPUs available
	thrN := lib.GetThreadsNum(ctx)
	fixtures := []lib.Fixture{}
	if thrN > 1 {
		if ctx.Debug > 0 {
			lib.Printf("Now processing %d fixture files using MT%d version\n", len(fixtureFiles), thrN)
		}
		ch := make(chan lib.Fixture)
		nThreads := 0
		gRateMtx = &sync.Mutex{}
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
	// Foundation-f aliases
	generateFoundationFAliases(ctx, &fixtures)
	// IMPL
	/*
		if 1 == 1 {
			os.Exit(1)
		}
	*/
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
				affiliationSource := fixture.Slug
				if fixture.Native.AffiliationSource != "" {
					affiliationSource = fixture.Native.AffiliationSource
				}
				if endpoint.AffiliationSource != "" {
					affiliationSource = endpoint.AffiliationSource
				}
				if ctx.Debug > 0 && affiliationSource != fixture.Slug {
					lib.Printf(
						"Using non-default '%s' affiliation source for '%s' endpoint (default would be '%s')\n",
						affiliationSource,
						name,
						fixture.Slug,
					)
				}

				flags := make(map[string]string, 0)
				switch dataSource.Slug {
				case lib.GoogleGroups:
					flags = map[string]string{
						"--googlegroups-do-fetch":    getFlagByName("dofetch", dataSource.Config),
						"--googlegroups-do-enrich":   getFlagByName("doenrich", dataSource.Config),
						"--googlegroups-groupname":   endpoint.Name,
						"--googlegroups-slug":        fixture.Native.Slug,
						"--googlegroups-project":     endpoint.Project,
						"--googlegroups-fetch-size":  getFlagByName("fetchsize", dataSource.Config),
						"--googlegroups-enrich-size": getFlagByName("enrichsize", dataSource.Config),
					}
				case lib.Pipermail:
					flags = map[string]string{
						"--pipermail-do-fetch":    getFlagByName("dofetch", dataSource.Config),
						"--pipermail-do-enrich":   getFlagByName("doenrich", dataSource.Config),
						"--pipermail-origin":      endpoint.Name,
						"--pipermail-slug":        fixture.Native.Slug,
						"--pipermail-project":     endpoint.Project,
						"--pipermail-fetch-size":  getFlagByName("fetchsize", dataSource.Config),
						"--pipermail-enrich-size": getFlagByName("enrichsize", dataSource.Config),
					}
				default:
					flags = map[string]string{
						"--bugzilla-origin":      name,
						"--bugzilla-do-fetch":    getFlagByName("dofetch", dataSource.Config),
						"--bugzilla-do-enrich":   getFlagByName("doenrich", dataSource.Config),
						"--bugzilla-project":     endpoint.Project,
						"--bugzilla-fetch-size":  getFlagByName("fetchsize", dataSource.Config),
						"--bugzilla-enrich-size": getFlagByName("enrichsize", dataSource.Config),
					}
				}
				tasks = append(
					tasks,
					lib.Task{
						Project:           endpoint.Project,
						ProjectP2O:        endpoint.ProjectP2O,
						ProjectNoOrigin:   endpoint.ProjectNoOrigin,
						Projects:          endpoint.Projects,
						Timeout:           endpoint.Timeout,
						CopyFrom:          endpoint.CopyFrom,
						PairProgramming:   endpoint.PairProgramming,
						Endpoint:          name,
						Config:            dataSource.Config,
						DsSlug:            dataSource.Slug,
						DsFullSlug:        dataSource.FullSlug,
						FxSlug:            fixture.Slug,
						FxFn:              fixture.Fn,
						MaxFreq:           dataSource.MaxFreq,
						AffiliationSource: affiliationSource,
						Groups:            calculateGroups(ctx, name, endpoint.Groups),
						Dummy:             endpoint.Dummy,
						Flags:             flags,
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
	// Most important work
	rslt := processTasks(ctx, &tasks, dss)
	if !ctx.OnlyP2O {
		gAliasesFunc()
		generateFoundationFAliases(ctx, &fixtures)
		processFixturesMetadata(ctx, &fixtures)
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

func checkProjectSlug(task *lib.Task) {
	// Warning when running for foundation-f project slug.
	if strings.HasSuffix(task.AffiliationSource, "-f") && !strings.Contains(task.AffiliationSource, "/") {
		lib.Printf("SDS WARNING: running on foundation-f level detected: task affiliation source is %s, for task: %s\n", task.AffiliationSource, task.String())
	}
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
	if ctx.SkipExternal {
		lib.Printf("Skip External is set, skipping enriching external indices\n")
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
	noEnrich := make(map[string]struct{})
	rename := func(arg string) string {
		arg = strings.Replace(arg, "/", "-", -1)
		if !strings.HasPrefix(arg, "sds-") {
			arg = "sds-" + arg
		}
		return arg
	}
	for _, fixture := range fixtures {
		for _, aliasFrom := range fixture.Aliases {
			if !strings.HasPrefix(aliasFrom.From, "bitergia-") {
				continue
			}
			lib.Printf("Enrich external indices: candidate: %s\n", aliasFrom.From)
			if aliasFrom.NoEnrich {
				noEnrich[aliasFrom.From] = struct{}{}
			}
			if len(aliasFrom.Dedup) > 0 {
				for _, dedup := range aliasFrom.Dedup {
					if !strings.HasSuffix(dedup, "-raw") {
						dedup = rename(dedup)
						ary, ok := manualEnrich[dedup]
						if !ok {
							manualEnrich[dedup] = []string{aliasFrom.From}
						} else {
							ary = append(ary, aliasFrom.From)
							manualEnrich[dedup] = ary
						}
					}
				}
				continue
			}
			for _, aliasTo := range aliasFrom.To {
				if !strings.HasSuffix(aliasTo, "-raw") {
					aliasTo = rename(aliasTo)
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
	if ctx.Debug > 0 {
		lib.Printf("Enrich external indices: list: %+v\n", manualEnrich)
	}
	indexToTask := make(map[string]lib.Task)
	dataSourceToTask := make(map[string]lib.Task)
	dsToCategory := make(map[string]map[string]struct{})
	for i, task := range tasks {
		idxSlug := "sds-" + task.FxSlug + "-" + task.DsFullSlug
		idxSlug = strings.Replace(idxSlug, "/", "-", -1)
		if ctx.Debug > 1 {
			lib.Printf("Enrich external indices: %v -> %s\n", task, idxSlug)
		}
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
	if ctx.Debug > 1 {
		lib.Printf("Enrich external indices: indexToTask: %+v\n", indexToTask)
	}
	newTasks := []lib.Task{}
	processedIndices := make(map[string]struct{})
	for sdsIndex, bitergiaIndices := range manualEnrich {
		sdsTask, ok := indexToTask[sdsIndex]
		if ctx.Debug > 0 {
			lib.Printf("Enrich external indices: %s -> %+v -> %v\n", sdsIndex, bitergiaIndices, ok)
		}
		if !ok {
			lib.Printf("WARNING: External index/indices have no corresponding configuration in SDS: %+v\n", bitergiaIndices)
			if ctx.SkipSH || ctx.SkipAffs {
				continue
			}
			for _, bitergiaIndex := range bitergiaIndices {
				_, ok := noEnrich[bitergiaIndex]
				if ok {
					lib.Printf("Enrich external indices: '%s' has enrichment disabled, skipping\n", bitergiaIndex)
					continue
				}
				_, ok = processedIndices[bitergiaIndex]
				if ok {
					lib.Printf("Enrich external indices: '%s' was already processed\n", bitergiaIndex)
					continue
				}
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
					if ctx.Debug > 0 {
						lib.Printf("Enrich external indices: %s/%s: adding %d artificial tasks (random task %+v, endpoints %+v)\n", bitergiaIndex, ds, len(endpoints), randomSdsTask, endpoints)
					}
					for _, endpoint := range endpoints {
						tsk := lib.Task{
							Project:           "",
							ProjectP2O:        false,
							ProjectNoOrigin:   false,
							Endpoint:          endpoint,
							Config:            randomSdsTask.Config,
							DsSlug:            ds,
							DsFullSlug:        randomSdsTask.DsFullSlug,
							FxSlug:            "random:" + randomSdsTask.FxSlug,
							AffiliationSource: randomSdsTask.FxSlug,
							FxFn:              "random:" + randomSdsTask.FxFn,
							MaxFreq:           randomSdsTask.MaxFreq,
							ExternalIndex:     bitergiaIndex,
						}
						if ctx.Debug > 1 {
							lib.Printf("Enrich external indices: task based on random sds task: %+v\n", tsk)
						}
						newTasks = append(newTasks, tsk)
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
				sdsIndex := "sds-" + sdsTask.FxSlug + "-" + sdsTask.DsFullSlug
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
			if ctx.SkipSH || ctx.SkipAffs {
				continue
			}
			_, ok = noEnrich[bitergiaIndex]
			if ok {
				lib.Printf("Enrich external indices: '%s' has enrichment disabled (but not deduplication)\n", bitergiaIndex)
				continue
			}
			_, ok := processedIndices[bitergiaIndex]
			if ok {
				lib.Printf("Enrich external indices: '%s' was already processed (skipping enrichment and deduplication)\n", bitergiaIndex)
				continue
			}
			if ctx.Debug > 0 {
				lib.Printf("Enrich external indices: %s/%s: adding %d external tasks (cloned task %+v, endpoints %+v)\n", bitergiaIndex, sdsTask.DsFullSlug, len(endpoints), sdsTask, endpoints)
			}
			for _, endpoint := range endpoints {
				tsk := lib.Task{
					Project:           "",
					ProjectP2O:        false,
					ProjectNoOrigin:   false,
					Endpoint:          endpoint,
					Config:            sdsTask.Config,
					DsSlug:            sdsTask.DsSlug,
					DsFullSlug:        sdsTask.DsFullSlug,
					FxSlug:            sdsTask.FxSlug,
					AffiliationSource: sdsTask.AffiliationSource,
					FxFn:              sdsTask.FxFn,
					MaxFreq:           sdsTask.MaxFreq,
					ExternalIndex:     bitergiaIndex,
				}
				if ctx.Debug > 1 {
					lib.Printf("Enrich external indices: external task: %+v\n", tsk)
				}
				newTasks = append(newTasks, tsk)
				processedIndices[bitergiaIndex] = struct{}{}
			}
		}
	}
	// Actual processing
	thrN := lib.GetThreadsNum(ctx)
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
	lib.Printf("Enrich external indices: external enrichment tasks: %d\n", len(newTasks))
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
			lib.Printf("External indices: %s\n", line)
		}
	}
	enrichExternal := func(ch chan [4]string, tsk lib.Task) (result [4]string) {
		defer func() {
			updateInfo(false, result)
			if ch != nil {
				ch <- result
			}
		}()
		dads := isDADS(&tsk)
		result[0] = tsk.ExternalIndex
		result[1] = tsk.DsSlug
		result[2] = tsk.Endpoint
		updateInfo(true, result)
		ds := tsk.DsSlug
		idxSlug := tsk.ExternalIndex
		mainEnv := make(map[string]string)
		var (
			commandLine []string
			envPrefix   string
		)
		if dads {
			commandLine = []string{"dads"}
			envPrefix = "DA_" + strings.ToUpper(strings.Split(tsk.DsSlug, "/")[0]) + "_"
			mainEnv[envPrefix+"ENRICH"] = "1"
			mainEnv[envPrefix+"RICH_INDEX"] = idxSlug
			mainEnv[envPrefix+"NO_RAW"] = "1"
			mainEnv[envPrefix+"REFRESH_AFFS"] = "1"
			mainEnv[envPrefix+"FORCE_FULL"] = "1"
			mainEnv[envPrefix+"ES_URL"] = ctx.ElasticURL
			mainEnv[envPrefix+"AFFILIATION_API_URL"] = ctx.AffiliationAPIURL
			mainEnv["AUTH0_DATA"] = ctx.Auth0Data
		} else {
			commandLine = []string{
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
		}
		redactedCommandLine := make([]string, len(commandLine))
		copy(redactedCommandLine, commandLine)
		if dads {
			if tsk.PairProgramming {
				mainEnv[envPrefix+"PAIR_PROGRAMMING"] = "1"
			}
			switch ctx.CmdDebug {
			case 0:
			case 1, 2:
				mainEnv[envPrefix+"DEBUG"] = "1"
			default:
				if ctx.CmdDebug > 0 {
					mainEnv[envPrefix+"DEBUG"] = strconv.Itoa(ctx.CmdDebug - 1)
				}
			}
			if ctx.EsBulkSize > 0 {
				mainEnv[envPrefix+"ES_BULK_SIZE"] = strconv.Itoa(ctx.EsBulkSize)
			}
			if ctx.ScrollSize > 0 {
				mainEnv[envPrefix+"ES_SCROLL_SIZE"] = strconv.Itoa(ctx.ScrollSize)
			}
			if ctx.ScrollWait > 0 {
				mainEnv[envPrefix+"ES_SCROLL_WAIT"] = strconv.Itoa(ctx.ScrollWait) + "s"
			}
			if !ctx.SkipSH {
				mainEnv[envPrefix+"DB_HOST"] = ctx.ShHost
				mainEnv[envPrefix+"DB_NAME"] = ctx.ShDB
				mainEnv[envPrefix+"DB_USER"] = ctx.ShUser
				mainEnv[envPrefix+"DB_PASS"] = ctx.ShPass
				if ctx.ShPort != "" {
					mainEnv[envPrefix+"DB_PORT"] = ctx.ShPort
				}
			}
			if strings.Contains(ds, "/") {
				ary := strings.Split(ds, "/")
				if len(ary) != 2 {
					result[3] = fmt.Sprintf("%s: %+v: %s\n", ds, tsk, lib.ErrorStrings[1])
					return
				}
				mainEnv[envPrefix+"CATEGORY"] = ary[1]
				ds = ary[0]
			}
		} else {
			redactedCommandLine[len(redactedCommandLine)-1] = lib.Redacted
			if tsk.PairProgramming {
				commandLine = append(commandLine, "--pair-programming")
				redactedCommandLine = append(redactedCommandLine, "--pair-programming")
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
		}

		// Handle DS endpoint
		eps, epEnv := massageEndpoint(tsk.Endpoint, ds, dads, idxSlug, tsk.Project)
		if len(eps) == 0 {
			result[3] = fmt.Sprintf("%s: %+v: %s", tsk.Endpoint, tsk, lib.ErrorStrings[2])
			return
		}
		if dads {
			for k, v := range epEnv {
				mainEnv[k] = v
			}
		} else {
			for _, ep := range eps {
				commandLine = append(commandLine, ep)
				redactedCommandLine = append(redactedCommandLine, ep)
			}
		}

		// Handle DS config options
		multiConfig, cfgEnv, fail := massageConfig(ctx, &(tsk.Config), ds, idxSlug)
		if fail == true {
			result[3] = fmt.Sprintf("%+v: %s\n", tsk, lib.ErrorStrings[3])
			return
		}
		if dads {
			for k, v := range cfgEnv {
				mainEnv[k] = v
			}
			// Handle DS project
			if tsk.Project != "" {
				mainEnv[envPrefix+"PROJECT"] = tsk.Project
			}
			if tsk.ProjectP2O {
				mainEnv[envPrefix+"PROJECT_FILTER"] = "1"
			}
		} else {
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
		}
		mainEnv["PROJECT_SLUG"] = tsk.AffiliationSource
		mainEnv[envPrefix+"PROJECT_SLUG"] = tsk.AffiliationSource
		mainEnv["GROUPS"] = makeTaskGroupsEnv(&tsk)
		checkProjectSlug(&tsk)
		rcl := strings.Join(redactedCommandLine, " ")
		redactedEnv := lib.FilterRedacted(fmt.Sprintf("%s", sortEnv(mainEnv)))
		retries := 0
		dtStart := time.Now()
		for {
			if ctx.DryRun {
				if ctx.DryRunSeconds > 0 {
					if ctx.DryRunSecondsRandom {
						time.Sleep(time.Duration(rand.Intn(ctx.DryRunSeconds*1000)) * time.Millisecond)
					} else {
						time.Sleep(time.Duration(ctx.DryRunSeconds) * time.Second)
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
			if ctx.Debug > 0 {
				lib.Printf("External endpoint: %s %s\n", redactedEnv, rcl)
			}
			var (
				err error
				str string
			)
			if !ctx.SkipP2O {
				str, err = lib.ExecCommand(ctx, commandLine, mainEnv, nil)
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
			if strings.Contains(str, lib.DadsException) {
				pyE = true
				err = fmt.Errorf("%s %s", redactedEnv, strippedStr)
			}
			if strings.Contains(str, lib.DadsWarning) {
				lib.Printf("Command error for %s %+v: %s\n", redactedEnv, rcl, strippedStr)
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
				lib.Printf("Timeout error for %s %+v (took %v, tried %d times): %+v: %s\n", redactedEnv, rcl, dtEnd.Sub(dtStart), retries, err, strippedStr)
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
				lib.Printf("Command error for %s %+v (took %v, tried %d times): %+v\n", redactedEnv, rcl, dtEnd.Sub(dtStart), retries, err)
			} else {
				lib.Printf("Error for %s %+v (took %v, tried %d times): %+v: %s\n", redactedEnv, rcl, dtEnd.Sub(dtStart), retries, err, strippedStr)
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
	//defer func() { lib.Printf("figureOutEndpoints(%s, %s) -> (%v,%v)\n", index, dataSource, endpoints, origins) }()
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
	err = jsoniter.NewDecoder(resp.Body).Decode(&payload)
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
	payloadBytes, err := jsoniter.Marshal(data)
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
	payloadBytes, err := jsoniter.Marshal(data)
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
	payloadBytes, err = jsoniter.Marshal(data)
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
			if ds.Slug == "earned_media" {
				continue
			}
			// Skip configured but empty data sources
			if len(ds.Endpoints) == 0 && len(ds.Projects) == 0 {
				continue
			}
			idxSlug := "sds-" + slug + "-" + ds.FullSlug
			idxSlug = strings.Replace(idxSlug, "/", "-", -1)
			if idxSlug == "sds-" {
				lib.Printf("WARNING: empty index generated for fixture %s, datasource: %+v\n", fixture.Slug, ds.Slug)
			}
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
		for _, alias := range fixture.Aliases {
			idxSlug := alias.From
			if strings.HasPrefix(alias.From, "pattern:") || strings.HasPrefix(alias.From, "bitergia-") {
				continue
			}
			if idxSlug == "sds-" {
				lib.Printf("WARNING: empty index generated for fixture %s, alias: %+v\n", fixture.Slug, alias)
			}
			should[idxSlug] = struct{}{}
			// should[idxSlug+"-raw"] = struct{}{}
		}
	}
	if ctx.Debug > 1 {
		lib.Printf("should: %+v\n", should)
		lib.Printf("fromFull: %+v\n", fromFull)
		lib.Printf("toFull: %+v\n", toFull)
	}
	if ctx.Debug == 1 {
		lib.Printf("should have indices: %+v\n", should)
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
	err = jsoniter.NewDecoder(resp.Body).Decode(&indices)
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
				// Note: Skip PRs
				if !notMissingPattern.MatchString(fullIndex) {
					missing = append(missing, fullIndex)
				}
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
	if ctx.Debug > 1 {
		lib.Printf("got: %+v\n", got)
		lib.Printf("missing: %+v\n", missing)
		lib.Printf("extra: %+v\n", extra)
	}
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
	newExtra := []string{}
	for _, idx := range extra {
		if noDropPattern.MatchString(idx) {
			continue
		}
		newExtra = append(newExtra, idx)
	}
	extra = newExtra
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
		url = fmt.Sprintf("%s/%s", ctx.ElasticURL, indices)
		rurl = fmt.Sprintf("/%s", indices)
		if ctx.DryRun {
			lib.Printf("Would execute: method:%s url:%s\n", method, os.ExpandEnv(rurl))
			continue
		}
		if ctx.NoIndexDrop {
			lib.Printf("WARNING: Need to delete indices: %s\n", indices)
			lib.Printf("Would execute: method:%s url:%s\n", method, os.ExpandEnv(rurl))
			continue
		}
		lib.Printf("Deleting indices: %s\n", indices)
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
			for _, view := range alias.Views {
				should[strings.Replace(view.Name, "/", "-", -1)] = struct{}{}
			}
		}
	}
	if ctx.Debug > 0 {
		lib.Printf("should have aliases: %+v\n", should)
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
	err = jsoniter.NewDecoder(resp.Body).Decode(&aliases)
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
			// Note: Skip PRs
			if !notMissingPattern.MatchString(alias) {
				missing = append(missing, alias)
			}
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
	newExtra := []string{}
	for _, idx := range extra {
		if noDropPattern.MatchString(idx) {
			continue
		}
		newExtra = append(newExtra, idx)
	}
	extra = newExtra
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
			continue
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
		from := pair[0]
		if strings.HasPrefix(from, "pattern:") {
			from = from[8:]
		}
		url = fmt.Sprintf("%s/%s/_alias/%s", ctx.ElasticURL, from, pair[1])
		rurl = fmt.Sprintf("/%s/_alias/%s", from, pair[1])
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

func processAliasView(ch chan struct{}, ctx *lib.Ctx, index string, view lib.AliasView) {
	defer func() {
		if ch != nil {
			ch <- struct{}{}
		}
	}()
	if strings.HasPrefix(index, "pattern:") {
		index = index[8:]
	}
	// API: POST /_aliases '{"actions":[{"add":{"index":"sds-lfn-onap-git-for-merge","alias":"test-lg","filter":{"term":{"project":"CLI"}}}}]}'
	method := lib.Post
	url := fmt.Sprintf("%s/_aliases", ctx.ElasticURL)
	rurl := "/_aliases"
	if ctx.DryRun {
		lib.Printf("DryRun: Method:%s url:%s\n", method, rurl)
		return
	}
	lib.Printf("View '%s' -> '%s' filter %+v\n", index, view.Name, view.Filter)
	payloadBytes, err := jsoniter.Marshal(view.Filter)
	if err != nil {
		lib.Fatalf("json marshall error: %+v, data: %+v", err, view.Filter)
	}
	data := fmt.Sprintf(`{"actions":[{"add":{"index":"%s","alias":"%s","filter":%s}}]}`, index, view.Name, string(payloadBytes))
	payloadBytes = []byte(data)
	payloadBody := bytes.NewReader(payloadBytes)
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
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lib.Printf("ReadAll request error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s data:%s status:%d\n%s\n", method, rurl, data, resp.StatusCode, body)
	}
}

func processAliases(ctx *lib.Ctx, pFixtures *[]lib.Fixture, method string) {
	st := time.Now()
	fixtures := *pFixtures
	pairs := [][2]string{}
	tom := make(map[string]struct{})
	type view struct {
		from string
		view lib.AliasView
	}
	views := []view{}
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
			for _, v := range alias.Views {
				if method == lib.Delete {
					_, ok := tom[v.Name]
					if ok {
						continue
					}
					pairs = append(pairs, [2]string{alias.From, v.Name})
					tom[v.Name] = struct{}{}
					continue
				}
				views = append(views, view{from: alias.From, view: v})
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
		lib.Printf("Now processing %d aliases views using MT%d version\n", len(views), thrN)
		for _, v := range views {
			go processAliasView(ch, ctx, v.from, v.view)
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
		lib.Printf("Now processing %d aliases views using ST version\n", len(views))
		for _, v := range views {
			processAliasView(nil, ctx, v.from, v.view)
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
					lib.Printf("\n\n %+v \n\n", tasks[durs[dur]].ShortString())
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
	var enrichCallsMtx *sync.Mutex
	enrichCalls := make(map[string]struct{})
	var addEnrichCall func(tr *lib.TaskResult)
	if ctx.SkipEnrichDS || (ctx.DryRun && !ctx.DryRunAllowEnrichDS) {
		addEnrichCall = func(tr *lib.TaskResult) {
		}
	} else {
		addEnrichCall = func(tr *lib.TaskResult) {
			// TODO: only support dockerhub for now.
			if tr.Ds != "dockerhub" {
				return
			}
			if enrichCallsMtx != nil {
				enrichCallsMtx.Lock()
			}
			enrichCalls[tr.Fx+"@"+tr.Ds] = struct{}{}
			if enrichCallsMtx != nil {
				enrichCallsMtx.Unlock()
			}
		}
	}
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
			enrichCallsMtx = &sync.Mutex{}
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
					tasks[tIdx].Env = result.Env
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
					if res[1] < 0 {
						continue
					}
					setSyncInfo(ctx, &tMtx, &result, false)
					if result.Err == nil && len(result.Projects) > 0 {
						setProject(ctx, result.Index, result.Projects)
					}
					addEnrichCall(&result)
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
				tasks[tIdx].Env = result.Env
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
				addEnrichCall(&result)
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
			tasks[tIdx].Env = result.Env
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
			if res[1] < 0 {
				continue
			}
			setSyncInfo(ctx, &tMtx, &result, false)
			if result.Err == nil && len(result.Projects) > 0 {
				setProject(ctx, result.Index, result.Projects)
			}
			addEnrichCall(&result)
		}
		enTime := time.Now()
		lib.Printf("Pass (threads join) finished in %v\n", enTime.Sub(stTime))
	}
	if len(enrichCalls) > 0 {
		enrich := func(ch chan struct{}, d string) {
			defer func() {
				if ch != nil {
					ch <- struct{}{}
				}
			}()
			ary := strings.Split(d, "@")
			metricsEnrich(ctx, ary[0], ary[1])
		}
		if thrN > 1 {
			if ctx.Debug > 0 {
				lib.Printf("Now post-processing %d metrics enrichments using MT%d version\n", len(enrichCalls), thrN)
			}
			ch := make(chan struct{})
			nThreads := 0
			for d := range enrichCalls {
				go enrich(ch, d)
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
			if ctx.Debug > 0 {
				lib.Printf("Now post-processing %d metrics enrichments using ST version\n", len(enrichCalls))
			}
			for d := range enrichCalls {
				enrich(nil, d)
			}
		}
	}
	info("final")
	lib.Printf("Skipped tasks: %d\n", skippedTasks)
	return nil
}

func addSSHPrivKey(ctx *lib.Ctx, key, idxSlug string) bool {
	if ctx.DryRun && !ctx.DryRunAllowSSH {
		return true
	}
	home := os.Getenv("HOME")
	dir := home + "/.ssh-" + idxSlug
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
func massageEndpoint(endpoint string, ds string, dads bool, idxSlug string, project string) (e []string, env map[string]string) {
	defer func() {
		env = p2oEndpoint2dadsEndpoint(e, ds, dads, idxSlug, project)
	}()
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
		lib.GoogleGroups: {},
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
	} else if ds == lib.GoogleGroups {
		// google groups endpoint should be a valid email address.
		if len(endpoint) < 3 || len(endpoint) > 254 {
			return
		}
		if ok := emailRegex.MatchString(endpoint); ok {
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
func massageConfig(ctx *lib.Ctx, config *[]lib.Config, ds, idxSlug string) (c []lib.MultiConfig, env map[string]string, fail bool) {
	defer func() {
		c, env = p2oConfig2dadsConfig(c, ds)
	}()
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
				if strings.Contains(value, ",") || strings.Contains(value, "[") || strings.Contains(value, "]") {
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
				_, ok := cfg.Flags["no_default_tokens"]
				if ok {
					c = append(c, lib.MultiConfig{Name: "-t", Value: vals, RedactedValue: []string{lib.Redacted}})
				} else {
					if ctx.DynamicOAuth {
						var dynamicKeys []string
						envOAuths := os.Getenv("SDS_GITHUB_OAUTH")
						if envOAuths != "" {
							oAuths := strings.Split(envOAuths, ",")
							for _, auth := range oAuths {
								dynamicKeys = append(dynamicKeys, auth)
							}
						} else {
							dynamicKeys = ctx.OAuthKeys
						}
						c = append(c, lib.MultiConfig{Name: "-t", Value: mergeTokens(ctx, inf, vals, dynamicKeys), RedactedValue: []string{lib.Redacted}})
					} else {
						c = append(c, lib.MultiConfig{Name: "-t", Value: mergeTokens(ctx, inf, vals, ctx.OAuthKeys), RedactedValue: []string{lib.Redacted}})
					}
				}
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
			if name == lib.APIToken {
				// we can specify GitHub API token for git (to be able to get 'github_org' or 'github_user' repos
				// but this token is not needed in p2o call
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
				continue
			}
			m[name] = struct{}{}
			c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
		}
		_, ok := m["ssh-id-filepath"]
		if !ok {
			home := os.Getenv("HOME")
			fn := home + "/.ssh-" + idxSlug + "/id_rsa"
			c = append(c, lib.MultiConfig{Name: "ssh-id-filepath", Value: []string{fn}, RedactedValue: []string{fn}})
		}
		_, ok = m["disable-host-key-check"]
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
	} else if ds == lib.GoogleGroups {
		for _, cfg := range *config {
			name := cfg.Name
			value := cfg.Value
			redactedValue := value
			if lib.IsRedacted(name) {
				lib.AddRedacted(value, true)
				redactedValue = lib.Redacted
			}
			m[name] = struct{}{}
			c = append(c, lib.MultiConfig{Name: name, Value: []string{value}, RedactedValue: []string{redactedValue}})
		}
	} else {
		fail = true
	}
	return
}

// p2oEndpoint2dadsEndpoint - map p2o.py endpoint to dads endpoint
func p2oEndpoint2dadsEndpoint(e []string, ds string, dads bool, idxSlug string, project string) (env map[string]string) {
	all, ok := dadsTasks[ds]
	if !ok {
		return
	}
	// fmt.Printf("p2oEndpoint2dadsEndpoint: dads is %v\n", dads)
	if !all && !dads {
		return
	}
	env = make(map[string]string)
	defaults, ok := dadsEnvDefaults[ds]
	if ok {
		for k, v := range defaults {
			env[k] = v
		}
	}
	env["DA_DS"] = ds
	prefix := "DA_" + strings.ToUpper(ds) + "_"
	switch ds {
	case lib.Jira, lib.Git, lib.Gerrit, lib.Confluence:
		env[prefix+"URL"] = e[0]
	case lib.GitHub:
		env[prefix+"ORG"] = e[0]
		env[prefix+"REPO"] = e[1]
	case lib.GroupsIO:
		env[prefix+"GROUP_NAME"] = e[0]
	case lib.RocketChat:
		env[prefix+"URL"] = e[0]
		env[prefix+"CHANNEL"] = e[1]
	case lib.DockerHub:
		type repository struct {
			Owner      string
			Repository string
			Project    string
			ESIndex    string
		}
		repos := make([]repository, 1)
		// fill repos
		repos[0] = repository{e[0], e[1], project, idxSlug}
		data, err := jsoniter.Marshal(&repos)
		if err != nil {
			lib.Fatalf("p2oEndpoint2dadsEndpoint: Error in dockerhub reposiories: DS%s", ds)
		}
		env[prefix+"REPOSITORIES_JSON"] = string(data)
	case lib.Bugzilla, lib.BugzillaRest:

	case lib.Jenkins:
		type buildServer struct {
			URL     string `json:"url"`
			Project string `json:"project"`
			Index   string `json:"index"`
		}
		buildServers := make([]buildServer, 1)
		buildServers[0] = buildServer{
			URL:     e[0],
			Project: project,
			Index:   idxSlug,
		}
		data, err := jsoniter.Marshal(&buildServers)
		if err != nil {
			lib.Fatalf("p2oEndpoint2dadsEndpoint: Error in Jenkins buildservers: DS %s", ds)
		}
		// wrap JSON with single quote for da-ds Unmarshal
		env[prefix+"JENKINS_JSON"] = fmt.Sprintf("%s", string(data))
	case lib.GoogleGroups, lib.Pipermail:

	default:
		// lib.Fatalf("ERROR: p2oEndpoint2dadsEndpoint: DS %s not (yet) supported", ds)
		lib.Printf("ERROR(non fatal): p2oEndpoint2dadsEndpoint: DS %s not (yet) supported", ds)
	}
	// fmt.Printf("%+v --> %+v\n", e, env)
	return
}

// p2oConfig2dadsConfig - map p2o.py configuration to dads configuration
func p2oConfig2dadsConfig(c []lib.MultiConfig, ds string) (oc []lib.MultiConfig, env map[string]string) {
	all, ok := dadsTasks[ds]
	if !ok {
		oc = c
		return
	}
	dads := all
	if !all {
		for _, i := range c {
			if i.Name == lib.DADS && len(i.Value) > 0 && i.Value[0] != "" && i.Value[0] != lib.Nil {
				dads = true
			}
		}
		// fmt.Printf("p2oConfig2dadsConfig: check for DADS %+v --> %v\n", c, dads)
	}
	if !all && !dads {
		oc = c
		return
	}
	env = make(map[string]string)
	env["DA_DS"] = ds
	prefix := "DA_" + strings.ToUpper(ds) + "_"
	for _, i := range c {
		opt := i.Name
		switch opt {
		case "no-archive":
			opt = ""
		case "-u":
			opt = "user"
		case "-e":
			opt = "email"
		case "-p":
			opt = "password"
		case "-t", "api-token":
			if ds == lib.GitHub {
				opt = "tokens"
			} else {
				opt = "token"
			}
		case "from-date":
			opt = "date-from"
		case "to-date":
			opt = "date-to"
		case "ssh-id-filepath":
			opt = "ssh-key-path"
		}
		if opt == "" {
			continue
		}
		envOpt := prefix + strings.ToUpper(strings.Replace(opt, "-", "_", -1))
		var envVal string
		if ds == lib.GitHub {
			envVal = strings.Join(i.Value, ",")
		} else {
			envVal = strings.Join(i.Value, " ")
		}
		if envVal == "" {
			envVal = "1"
		}
		if envVal == lib.Nil {
			envVal = ""
		}
		env[envOpt] = envVal
	}
	oc = []lib.MultiConfig{}
	// fmt.Printf("%+v --> %+v,%+v\n", c, oc, env)
	return
}

//func massageDataSource(ds string) string {
//	return ds
//}

func searchByQueryFirstID(ctx *lib.Ctx, index, esQuery string) (id string) {
	data := lib.EsSearchPayload{Query: lib.EsSearchQuery{QueryString: lib.EsSearchQueryString{Query: esQuery}}}
	payloadBytes, err := jsoniter.Marshal(data)
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
	err = jsoniter.NewDecoder(resp.Body).Decode(&payload)
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
	payloadBytes, err := jsoniter.Marshal(data)
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
	err = jsoniter.NewDecoder(resp.Body).Decode(&payload)
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
	payloadBytes, err := jsoniter.Marshal(data)
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
	payloadBytes, err := jsoniter.Marshal(data)
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
	b, _ := jsoniter.Marshal(str)
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
	err = jsoniter.NewDecoder(resp.Body).Decode(&payload)
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

func sortEnv(env map[string]string) (envStr string) {
	ks := []string{}
	for k := range env {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		envStr += k + "=" + env[k] + " "
	}
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
	rEnvStr := ""
	envStr := ""
	ks := []string{}
	for k := range result.Env {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		v := result.Env[k]
		rEnvStr += k + "=" + lib.FilterRedacted(v) + " "
		envStr += k + "=" + v + " "
	}
	cl := envStr + result.CommandLine
	rcl := rEnvStr + result.RedactedCommandLine
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
		payloadBytes, err = jsoniter.Marshal(data)
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

func getConditionJSON(conds []lib.ColumnCondition, origin string, withOrigin bool) (s string) {
	for _, cond := range conds {
		val := cond.Value
		if val == "{{endpoint}}" {
			val = origin
		}
		s += fmt.Sprintf(`{"regexp":{"%s":{"value":"%s", "flags":"ALL"}}},`, jsonEscape(cond.Column), jsonEscape(val))
	}
	if withOrigin {
		s += fmt.Sprintf(`{"term":{"origin":"%s"}},`, origin)
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
		projectEpoch := time.Now().Unix()
		var err error
		data := ""
		var projectVal string
		if project == lib.Null {
			projectVal = "null"
		} else {
			projectVal = `\"` + jsonEscape(project) + `\"`
		}
		lastEpoch := int64(0)
		payloadBytes := []byte{}
		if origin == lib.ProjectNoOrigin {
			data = fmt.Sprintf(
				`{"script":{"inline":"ctx._source.project=%s;ctx._source.project_ts=%d;"},"query":{"match_all":{}}}`,
				projectVal,
				projectEpoch,
			)
			payloadBytes = []byte(data)
		} else {
			must := getConditionJSON(conf.Must, origin, false)
			mustPartial := ""
			if must != "" {
				mustPartial = "," + must
			}
			mustNot := getConditionJSON(conf.MustNot, origin, false)
			optionalMustNot := ""
			mustNotPartial := ""
			if mustNot != "" {
				optionalMustNot = `,"must_not":[` + mustNot + "]"
				mustNotPartial = "," + mustNot
			}
			if !ctx.SkipProjectTS {
				lastEpoch = lastProjectDate(ctx, index, origin, must, mustNot, true)
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
		err = jsoniter.NewDecoder(resp.Body).Decode(&payload)
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
		`{"query":{"bool":{"must":[{"exists":{"field":"%s"}}%s]%s}},"sort":{"%s":"desc"}}`,
		lib.CopyFromDateField,
		mustPartial,
		optionalMustNot,
		lib.CopyFromDateField,
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
	err = jsoniter.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		lib.Printf("JSON decode error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			lib.Printf("Body:%s\n", body)
		}
		return
	}
	if len(payload.Hits.Hits) == 0 {
		return
	}
	switch lib.CopyFromDateField {
	case "metadata__enriched_on":
		epoch = payload.Hits.Hits[0].Source.MDEnrichedOn
	case "metadata__timestamp":
		epoch = payload.Hits.Hits[0].Source.MDTimestamp
	case "metadata__updated_on":
		epoch = payload.Hits.Hits[0].Source.MDUpdatedOn
	case "grimoire_creation_date":
		epoch = payload.Hits.Hits[0].Source.GrimoireCreationDate
	default:
	}
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
	err = jsoniter.Unmarshal(body, &esResult)
	if err != nil {
		lib.Printf("Bulk result unmarshal error: %+v", err)
		return
	}
	for i, item := range esResult.Items {
		if item.Index.Status != 201 {
			err = fmt.Errorf("failed to create #%d item, status %d, error %+v", i, item.Index.Status, item.Index.Error)
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
	err = jsoniter.NewDecoder(resp.Body).Decode(&result)
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
	jsonBytes, err = jsoniter.Marshal(mapping)
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
			jsonBytes, err = jsoniter.Marshal(mp)
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
	scrollSize := 1000
	scrollTime := "45m"
	bulkSize := 1000
	conf := task.CopyFrom
	origin := mapOrigin(task.Endpoint, task.DsSlug)
	if ctx.Debug > 0 {
		lib.Printf("%s:%s: copy config: %+v\n", index, origin, conf)
	}
	mustNoOrigin := getConditionJSON(conf.Must, origin, false)
	mustWithOrigin := getConditionJSON(conf.Must, origin, true)
	mustPartialNoOrigin := ""
	if mustNoOrigin != "" {
		mustPartialNoOrigin = "," + mustNoOrigin
	}
	mustPartialWithOrigin := "," + mustWithOrigin
	mustNot := getConditionJSON(conf.MustNot, origin, false)
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
	if err != nil {
		lib.Printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode == 200 {
		lib.Printf("copy_from: dropped conflicting alias: %s\n", index)
	}
	// Delete destination index if not incremental mode
	if !conf.Incremental {
		method = lib.Delete
		url = fmt.Sprintf("%s/%s", ctx.ElasticURL, index)
		rurl = fmt.Sprintf("/%s", index)
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
		_ = resp.Body.Close()
		if resp.StatusCode == 200 {
			lib.Printf("copy_from: dropped index: %s (no incremental mode set)\n", index)
		} else {
			lib.Printf("WARNING: copy_from: failed to drop index: %s (will use incremental mode)\n", index)
		}
	}
	pattern := conf.Pattern
	err = copyMapping(ctx, pattern, index)
	if err != nil {
		lib.Printf("copy_from: copyMapping(%s,%s): %v\n", pattern, index, err)
	}
	// Can be used to cleanup origin based copies (reset them)
	// deleted := deleteByQuery(ctx, index, "origin:\""+origin+"\"")
	// lib.Printf("deleted:%v\n", deleted)
	// Now check last date on index (not alias) if present
	lastDateNoOrigin := lastDataDate(ctx, index, mustNoOrigin, mustNot, false)
	lastDateWithOrigin := lastDataDate(ctx, index, mustWithOrigin, mustNot, false)
	if ctx.Debug > 0 {
		lib.Printf("copy_from: %s -> %s (origin: %s): from: %+v (%+v without origin)\n", pattern, index, origin, lastDateWithOrigin, lastDateNoOrigin)
	}
	lastDate := lastDateWithOrigin
	mustPartial := mustPartialWithOrigin
	if task.CopyFrom.NoOrigin {
		lastDate = lastDateNoOrigin
		mustPartial = mustPartialNoOrigin
	}
	lib.Printf("copy_from: %s -> %s (origin: %s): from: %+v\n", pattern, index, origin, lastDate)
	payloadBytes := []byte{}
	data := ""
	if lastDate.IsZero() {
		data = fmt.Sprintf(
			`{"size":%d,"query":{"bool":{"must":[{"exists":{"field":"%s"}}%s]%s}}}`,
			scrollSize,
			lib.CopyFromDateField,
			mustPartial,
			optionalMustNot,
		)
		payloadBytes = []byte(data)
	} else {
		millis := lastDate.UnixNano() / 1000000
		// millis -= 36000000
		data = fmt.Sprintf(
			`{"size":%d,"query":{"bool":{"must_not":[{"range":{"%s":{"lte":%d,"format":"epoch_millis"}}}%s],"must":[{"exists":{"field":"%s"}}%s]}}}`,
			scrollSize,
			lib.CopyFromDateField,
			millis,
			mustNotPartial,
			lib.CopyFromDateField,
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
	var plBody []byte
	plBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		lib.Printf("Read response body error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		return
	}
	_ = resp.Body.Close()
	err = jsoniter.Unmarshal(plBody, &payload)
	if err != nil {
		lib.Printf("JSON decode error: %+v for %s url: %s, data: %s\n", err, method, rurl, data)
		lib.Printf("Body:%s\n", plBody)
		return
	}
	jsons := [][]byte{}
	nJSONs := 0
	bulks := 0
	scrollID := payload.ScrollID
	docs := 0
	header := true
	for {
		var (
			result interface{}
			field  interface{}
		)
		if header {
			err = jsoniter.Unmarshal(plBody, &result)
			header = false
		} else {
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
			err = jsoniter.NewDecoder(resp.Body).Decode(&result)
		}
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
			jsonBytes, err = jsoniter.Marshal(doc)
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
	case lib.GroupsIO:
		return "https://groups.io/g/" + origin
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
		if task.ProjectNoOrigin {
			result.Projects = append(
				result.Projects,
				lib.EndpointProject{
					Name:   task.Project,
					Origin: lib.ProjectNoOrigin,
				},
			)
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
}

func taskFilteredOut(ctx *lib.Ctx, tsk *lib.Task) bool {
	idxSlug := "sds-" + tsk.FxSlug + "-" + tsk.DsFullSlug
	idxSlug = strings.Replace(idxSlug, "/", "-", -1)
	task := idxSlug + ":" + tsk.Endpoint
	if (ctx.TasksRE != nil && !ctx.TasksRE.MatchString(task)) || (ctx.TasksSkipRE != nil && ctx.TasksSkipRE.MatchString(task)) || (ctx.TasksExtraSkipRE != nil && ctx.TasksExtraSkipRE.MatchString(task)) {
		if ctx.Debug > 0 {
			lib.Printf("Task %s filtered out due to RE (match,skip,extraSkip) = (%+v,%+v,%+v)\n", task, ctx.TasksRE, ctx.TasksSkipRE, ctx.TasksExtraSkipRE)
		}
		return true
	}
	return false
}

func isDADS(task *lib.Task) bool {
	all, ok := dadsTasks[strings.Split(task.DsSlug, "/")[0]]
	if !ok {
		return false
	}
	if all {
		return true
	}
	dads := false
	for _, cfg := range task.Config {
		if cfg.Name == lib.DADS && cfg.Value != "" && cfg.Value != lib.Nil {
			dads = true
			break
		}
	}
	// fmt.Printf("isDADS: check for DADS %+v --> %v\n", task.Config, dads)
	return dads
}

func makeTaskGroupsEnv(task *lib.Task) string {
	m := map[string]struct{}{task.AffiliationSource: {}}
	for _, group := range task.Groups {
		m[group] = struct{}{}
	}
	s := []string{}
	for group := range m {
		s = append(s, group)
	}
	return strings.Join(s, ";")
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
	dads := isDADS(&task)
	ds := task.DsSlug
	fds := task.DsFullSlug
	idxSlug := "sds-" + task.FxSlug + "-" + fds
	idxSlug = strings.Replace(idxSlug, "/", "-", -1)
	origIdxSlug := idxSlug
	if dads {
		idxSlug = strings.Replace(idxSlug, "github-pull_request", "github-issue", -1)
	}
	result.Index = idxSlug
	result.Endpoint = task.Endpoint
	result.Ds = strings.Replace(task.DsSlug, "/", "-", -1)
	result.Fx = strings.Replace(task.FxSlug, "/", "-", -1)
	// Filter out by task / task skip RE
	if taskFilteredOut(ctx, &task) {
		result.Code[1] = -1
		return
	}
	// Nothing needs to be done for dummy tasks
	if task.Dummy {
		return
	}
	// Handle copy from another index slug
	if task.CopyFrom.Pattern != "" {
		if affs {
			return
		}
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
	mainEnv := make(map[string]string)
	var (
		commandLine []string
		envPrefix   string
	)
	if dads {
		commandLine = []string{"dads"}
		// add dads arguments
		if task.DsSlug == lib.Bugzilla || task.DsSlug == lib.BugzillaRest || task.DsSlug == lib.GoogleGroups || task.DsSlug == lib.Pipermail {
			for k, v := range task.Flags {
				commandLine = append(commandLine, k)
				commandLine = append(commandLine, v)
			}
		}

		envPrefix = "DA_" + strings.ToUpper(strings.Split(task.DsSlug, "/")[0]) + "_"
		mainEnv[envPrefix+"ENRICH"] = "1"
		mainEnv[envPrefix+"RAW_INDEX"] = idxSlug + "-raw"
		mainEnv[envPrefix+"RICH_INDEX"] = idxSlug
		mainEnv[envPrefix+"ES_URL"] = ctx.ElasticURL
		mainEnv[envPrefix+"AFFILIATION_API_URL"] = ctx.AffiliationAPIURL
		mainEnv["AUTH0_DATA"] = ctx.Auth0Data
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
		}
	}
	redactedCommandLine := make([]string, len(commandLine))
	copy(redactedCommandLine, commandLine)
	if dads {
		if affs {
			mainEnv[envPrefix+"NO_RAW"] = "1"
			mainEnv[envPrefix+"REFRESH_AFFS"] = "1"
			mainEnv[envPrefix+"FORCE_FULL"] = "1"
		}
		if task.PairProgramming {
			mainEnv[envPrefix+"PAIR_PROGRAMMING"] = "1"
		}
		switch ctx.CmdDebug {
		case 0:
		case 1, 2:
			mainEnv[envPrefix+"DEBUG"] = "1"
		default:
			if ctx.CmdDebug > 0 {
				mainEnv[envPrefix+"DEBUG"] = strconv.Itoa(ctx.CmdDebug - 1)
			}
		}
		if ctx.EsBulkSize > 0 {
			mainEnv[envPrefix+"ES_BULK_SIZE"] = strconv.Itoa(ctx.EsBulkSize)
		}
		if ctx.ScrollSize > 0 {
			mainEnv[envPrefix+"ES_SCROLL_SIZE"] = strconv.Itoa(ctx.ScrollSize)
		}
		if ctx.ScrollWait > 0 {
			mainEnv[envPrefix+"ES_SCROLL_WAIT"] = strconv.Itoa(ctx.ScrollWait) + "s"
		}
		if !ctx.SkipSH {
			mainEnv[envPrefix+"DB_HOST"] = ctx.ShHost
			mainEnv[envPrefix+"DB_NAME"] = ctx.ShDB
			mainEnv[envPrefix+"DB_USER"] = ctx.ShUser
			mainEnv[envPrefix+"DB_PASS"] = ctx.ShPass

			mainEnv[envPrefix+"GAP_URL"] = ctx.GapURL
			mainEnv[envPrefix+"RETRIES"] = ctx.Retries
			mainEnv[envPrefix+"DELAY"] = ctx.Delay

			if ctx.ShPort != "" {
				mainEnv[envPrefix+"DB_PORT"] = ctx.ShPort
			}
		}
		if strings.Contains(ds, "/") {
			ary := strings.Split(ds, "/")
			if len(ary) != 2 {
				lib.Printf("%s: %+v: %s\n", ds, task, lib.ErrorStrings[1])
				result.Code[1] = 1
				return
			}
			mainEnv[envPrefix+"CATEGORY"] = ary[1]
			ds = ary[0]
		}
	} else {
		redactedCommandLine[len(redactedCommandLine)-1] = lib.Redacted
		if affs {
			refresh := []string{"--only-enrich", "--refresh-identities", "--no_incremental"}
			commandLine = append(commandLine, refresh...)
			redactedCommandLine = append(redactedCommandLine, refresh...)
		}
		if task.PairProgramming {
			commandLine = append(commandLine, "--pair-programming")
			redactedCommandLine = append(redactedCommandLine, "--pair-programming")
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
	}
	// Handle DS endpoint
	eps, epEnv := massageEndpoint(task.Endpoint, ds, dads, idxSlug, task.Project)
	if len(eps) == 0 {
		lib.Printf("%s: %+v: %s\n", task.Endpoint, task, lib.ErrorStrings[2])
		result.Code[1] = 2
		return
	}
	if dads {
		for k, v := range epEnv {
			mainEnv[k] = v
		}
		if task.PairProgramming {
			mainEnv[envPrefix+"PAIR_PROGRAMMING"] = "1"
		}
		switch ctx.CmdDebug {
		case 0:
		case 1, 2:
			mainEnv[envPrefix+"DEBUG"] = "1"
		default:
			if ctx.CmdDebug > 0 {
				mainEnv[envPrefix+"DEBUG"] = strconv.Itoa(ctx.CmdDebug - 1)
			}
		}
	} else {
		for _, ep := range eps {
			commandLine = append(commandLine, ep)
			redactedCommandLine = append(redactedCommandLine, ep)
		}
	}
	sEp := strings.Join(eps, " ")
	if !ctx.SkipEsData && !ctx.SkipCheckFreq {
		var nilDur time.Duration
		if task.MaxFreq != nilDur {
			freqOK := checkSyncFreq(ctx, tMtx, origIdxSlug, sEp, task.MaxFreq)
			if !freqOK {
				// Mark as not executed due to freq check
				result.Code[1] = -3
				return
			}
		}
	}
	// Handle DS config options
	multiConfig, cfgEnv, fail := massageConfig(ctx, &(task.Config), ds, idxSlug)
	if fail == true {
		lib.Printf("%+v: %s\n", task, lib.ErrorStrings[3])
		result.Code[1] = 3
		return
	}
	if dads {
		for k, v := range cfgEnv {
			mainEnv[k] = v
		}
		// Handle DS project
		if task.Project != "" {
			mainEnv[envPrefix+"PROJECT"] = task.Project
		}
		if task.ProjectP2O {
			mainEnv[envPrefix+"PROJECT_FILTER"] = "1"
		}
	} else {
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
	}
	mainEnv["PROJECT_SLUG"] = task.AffiliationSource
	mainEnv[envPrefix+"PROJECT_SLUG"] = task.AffiliationSource
	mainEnv["GROUPS"] = makeTaskGroupsEnv(&task)
	checkProjectSlug(&task)
	result.CommandLine = strings.Join(commandLine, " ")
	result.RedactedCommandLine = strings.Join(redactedCommandLine, " ")
	result.Env = mainEnv
	redactedEnv := lib.FilterRedacted(fmt.Sprintf("%s", sortEnv(mainEnv)))
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
				_ = setLastRun(ctx, tMtx, origIdxSlug, sEp)
			}
			return
		}
		setSyncInfo(ctx, tMtx, &result, true)
		var (
			err error
			str string
		)
		if !ctx.SkipP2O {
			str, err = lib.ExecCommand(ctx, commandLine, mainEnv, &task.Timeout)
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
		if strings.Contains(str, lib.DadsException) {
			pyE = true
			err = fmt.Errorf("%s %s", redactedEnv, strippedStr)
		}
		if strings.Contains(str, lib.DadsWarning) {
			lib.Printf("Command error for %s %+v: %s\n", redactedEnv, redactedCommandLine, strippedStr)
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
			lib.Printf("Timeout error for %s %+v (took %v, tried %d times): %+v: %s\n", redactedEnv, redactedCommandLine, dtEnd.Sub(dtStart), retries, err, strippedStr)
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
			lib.Printf("Command error for %s %+v (took %v, tried %d times): %+v\n", redactedEnv, redactedCommandLine, dtEnd.Sub(dtStart), retries, err)
		} else {
			lib.Printf("Error for %s %+v (took %v, tried %d times): %+v: %s\n", redactedEnv, redactedCommandLine, dtEnd.Sub(dtStart), retries, err, strippedStr)
			str += fmt.Sprintf(": %+v", err)
		}
		result.Code[1] = 4
		result.Err = fmt.Errorf("last retry took %v: %+v", dtEnd.Sub(dtStart), strippedStr)
		result.Retries = retries
		return
	}
	if !ctx.SkipP2O && !ctx.SkipEsData && !affs {
		updated := setLastRun(ctx, tMtx, origIdxSlug, sEp)
		if !updated {
			lib.Printf("failed to set last sync date for %s/%s/%s\n", origIdxSlug, idxSlug, sEp)
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
	err = jsoniter.NewDecoder(resp.Body).Decode(&rdata)
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

func executeMetricsAPICall(ctx *lib.Ctx, path string) (err error) {
	if ctx.MetricsAPIURL == "" {
		err = fmt.Errorf("Cannot execute DA metrics API calls, no API URL specified")
		return
	}
	method := http.MethodGet
	rurl := path
	url := ctx.MetricsAPIURL + rurl
	if ctx.Debug > 0 {
		lib.Printf("%s\n", url)
	}
	req, e := http.NewRequest(method, url, nil)
	if e != nil {
		err = fmt.Errorf("new request error: %+v for %s url: %s", e, method, rurl)
		return
	}
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		err = fmt.Errorf("do request error: %+v for %s url: %s", e, method, rurl)
		return
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
		Text string `json:"status"`
	}
	err = jsoniter.NewDecoder(resp.Body).Decode(&rdata)
	_ = resp.Body.Close()
	if err != nil {
		return
	}
	lib.Printf("%s\n", rdata.Text)
	return
}

func executeAffiliationsAPICall(ctx *lib.Ctx, path string) (err error) {
	if ctx.AffiliationAPIURL == "" {
		err = fmt.Errorf("Cannot execute DA affiliation API calls, no API URL specified")
		return
	}
	if gToken == "" {
		gToken = os.Getenv("JWT_TOKEN")
	}
	if gToken == "" {
		lib.Printf("Obtaining API token\n")
		// err = getToken(ctx)
		gToken, err = lib.GetAPIToken()
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
			// err = getToken(ctx)
			gToken, err = lib.GetAPIToken()
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
		err = jsoniter.NewDecoder(resp.Body).Decode(&rdata)
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
	err = executeAffiliationsAPICall(ctx, "/v1/affiliation/hide_emails")
	return
}

func mergeAll(ctx *lib.Ctx) (err error) {
	if ctx.SkipMerge || ctx.OnlyP2O || (ctx.DryRun && !ctx.DryRunAllowMerge) {
		return
	}
	err = executeAffiliationsAPICall(ctx, "/v1/affiliation/merge_all")
	return
}

func mapOrgNames(ctx *lib.Ctx) (err error) {
	if ctx.SkipOrgMap || ctx.OnlyP2O || (ctx.DryRun && !ctx.DryRunAllowOrgMap) {
		return
	}
	err = executeAffiliationsAPICall(ctx, "/v1/affiliation/map_org_names")
	return
}

func detAffRange(ctx *lib.Ctx) (err error) {
	if !ctx.RunDetAffRange || ctx.OnlyP2O || (ctx.DryRun && !ctx.DryRunAllowDetAffRange) {
		return
	}
	err = executeAffiliationsAPICall(ctx, "/v1/affiliation/det_aff_range")
	return
}

func cacheTopContributors(ctx *lib.Ctx) (err error) {
	if ctx.SkipCacheTopContributors || ctx.OnlyP2O || (ctx.DryRun && !ctx.DryRunAllowCacheTopContributors) {
		return
	}
	err = executeAffiliationsAPICall(ctx, "/v1/affiliation/cache_top_contributors")
	return
}

func metricsEnrich(ctx *lib.Ctx, slug, ds string) {
	// We want this API to be called even in ONLY p2o mode - because it is tied to a single endpoint enrichment
	// if ctx.SkipEnrichDS || ctx.OnlyP2O || (ctx.DryRun && !ctx.DryRunAllowEnrichDS) {
	if ctx.SkipEnrichDS || (ctx.DryRun && !ctx.DryRunAllowEnrichDS) {
		return
	}
	err := executeMetricsAPICall(ctx, fmt.Sprintf("/v1/enrich/%s/datasource/%s", slug, ds))
	if err != nil {
		lib.Printf("Error metrics API enriching '%s'/'%s': %+v\n", slug, ds, err)
	}
	return
}

func getMetadataQueries(ctx *lib.Ctx, m *lib.Metadata) (result [][3]string) {
	patt := make(map[string][2]string)
	for _, ds := range m.DataSources {
		hasIndices := false
		hasPatterns := false
		items := []string{}
		var key [2]string
		for _, item := range ds.Slugs {
			if strings.HasPrefix(item, "pattern:") {
				items = append(items, item[8:])
				hasPatterns = true
				continue
			}
			items = append(items, "sds-"+strings.Replace(item, "/", "-", -1))
			hasIndices = true
		}
		if hasIndices && hasPatterns {
			lib.Printf("WARNING: incorrect metadata datasources slugs '%s' section, you cannot have both index names and patterns: %+v\n", ds.Name, ds)
			continue
		}
		key[0] = strings.Join(items, ",")
		hasIndices = false
		hasPatterns = false
		items = []string{}
		for _, item := range ds.Externals {
			if strings.HasPrefix(item, "pattern:") {
				items = append(items, item[8:])
				hasPatterns = true
				continue
			}
			items = append(items, strings.Replace(item, "/", "-", -1))
			hasIndices = true
		}
		if hasIndices && hasPatterns {
			lib.Printf("WARNING: incorrect metadata datasources externals '%s' section, you cannot have both index names and patterns: %+v\n", ds.Name, ds)
			continue
		}
		key[1] = strings.Join(items, ",")
		patt[ds.Name] = key
	}
	addQuery := func(dest [2]string, op, query string) {
		if dest[0] != "" {
			result = append(result, [3]string{op, dest[0], query})
		}
		if dest[1] != "" {
			result = append(result, [3]string{op, dest[1], query})
		}
	}
	updateOp := "_update_by_query"
	deleteOp := "_delete_by_query"
	for _, wg := range m.WorkingGroups {
		if len(wg.DataSources) == 0 {
			continue
		}
		noOverwrite := wg.NoOverwrite
		apply := map[string]string{"workinggroup": wg.Name}
		formed := ""
		for k, v := range wg.Meta {
			apply["meta_"+k] = v
			if formed == "" && strings.ToLower(k) == "formed" {
				formed = v
			} else if formed == "" && strings.ToLower(k) == "contributed" {
				formed = v
			}
		}
		var dtFormed time.Time
		if formed != "" {
			var e error
			dtFormed, e = time.Parse("2006-01-02", formed)
			if e != nil {
				lib.Printf("cannot parse formed/contributed date: '%s': %v, from %+v\n", formed, e, wg.Meta)
			}
		}
		hasFormed := !dtFormed.IsZero()
		// {"script":{"inline":"ctx._source.field1=%s;ctx._source.field2=%s;"},
		queryRoot := `{"script":{"inline":"`
		ks := []string{}
		for k := range apply {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		if noOverwrite {
			for _, k := range ks {
				v := apply[k]
				if v == lib.Null {
					queryRoot += `if(!ctx._source.containsKey(\"` + k + `\")){ctx._source.` + k + `=null;}`
					continue
				}
				queryRoot += `if(!ctx._source.containsKey(\"` + k + `\")){ctx._source.` + k + `=\"` + jsonEscape(v) + `\";}`
			}
		} else {
			for _, k := range ks {
				v := apply[k]
				if v == lib.Null {
					queryRoot += `ctx._source.` + k + `=null;`
					continue
				}
				queryRoot += `ctx._source.` + k + `=\"` + jsonEscape(v) + `\";`
			}
		}
		queryRoot += `"},`
		for _, ds := range wg.DataSources {
			dest, ok := patt[ds.Name]
			if !ok {
				lib.Printf("WARNING: working group data source '%s' not found in metadata data sources: '%+v'\n", ds.Name, wg)
				continue
			}
			if len(ds.Filter) == 0 {
				if len(ds.Origins) == 0 {
					lib.Printf("WARNING: working group data source '%s' has no origins and no filter, skipping: '%+v'\n", ds.Name, wg)
					continue
				}
				query := queryRoot + `"query":{"bool":{"should":[`
				for _, origin := range ds.Origins {
					query += `{"term":{"origin":"` + jsonEscape(origin) + `"}},`
				}
				query = query[:len(query)-1] + `]}}}`
				addQuery(dest, updateOp, query)
				if hasFormed {
					query := `{"query":{"bool":{"must":{"range":{"metadata__updated_on":{"lt":"` + formed + `"}}},"minimum_should_match":1,"should":[`
					for _, origin := range ds.Origins {
						query += `{"term":{"origin":"` + jsonEscape(origin) + `"}},`
					}
					query = query[:len(query)-1] + `]}}}`
					addQuery(dest, deleteOp, query)
				}
				continue
			}
			b, err := jsoniter.Marshal(ds.Filter)
			if err != nil {
				lib.Printf("WARNING: working group data source '%s' filter '%v' is broken: '%+v'\n", ds.Name, ds.Filter, wg)
				continue
			}
			if len(ds.Origins) == 0 {
				query := queryRoot + `"query":{"bool":{"filter":` + string(b) + `}}}`
				addQuery(dest, updateOp, query)
				if hasFormed {
					query := `{"query":{"bool":{"must":{"range":{"metadata__updated_on":{"lt":"` + formed + `"}}},"filter":` + string(b) + `}}}`
					addQuery(dest, deleteOp, query)
				}
				continue
			}
			query := queryRoot + `"query":{"bool":{"should":[`
			for _, origin := range ds.Origins {
				query += `{"term":{"origin":"` + jsonEscape(origin) + `"}},`
			}
			query = query[:len(query)-1] + `],"filter":` + string(b) + `}}}`
			addQuery(dest, updateOp, query)
			if hasFormed {
				query := `{"query":{"bool":{"must":{"range":{"metadata__updated_on":{"lt":"` + formed + `"}}},"minimum_should_match":1,"should":[`
				for _, origin := range ds.Origins {
					query += `{"term":{"origin":"` + jsonEscape(origin) + `"}},`
				}
				query = query[:len(query)-1] + `],"filter":` + string(b) + `}}}`
				addQuery(dest, deleteOp, query)
			}
		}
	}
	return
}

func processMetadataItem(ch chan struct{}, ctx *lib.Ctx, cfg [3]string) {
	if ch != nil {
		defer func() {
			ch <- struct{}{}
		}()
	}
	if ctx.Debug > 0 {
		lib.Printf("Processing %v\n", cfg)
	}
	op := cfg[0]
	pattern := cfg[1]
	data := cfg[2]
	payloadBytes := []byte(data)
	payloadBody := bytes.NewReader(payloadBytes)
	method := lib.Post
	url := fmt.Sprintf("%s/%s/%s?conflicts=proceed&refresh=true&timeout=20m", ctx.ElasticURL, pattern, op)
	rurl := fmt.Sprintf("/%s/%s?conflicts=proceed&refresh=true&timeout=20m", pattern, op)
	req, err := http.NewRequest(method, url, payloadBody)
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
	payload := lib.EsByQueryPayload{}
	err = jsoniter.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		body, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			lib.Printf("ReadAll request error when parsing response: %+v/%+v for %s url: %s, data: %+v", err, err2, method, rurl, data)
			return
		}
		lib.Printf("Method:%s url:%s status:%d data:%+v err:%+v\n%s", method, rurl, resp.StatusCode, data, err, body)
		return
	}
	if ctx.Debug > 0 {
		lib.Printf("%s/%s updated: %d, deleted: %d\n", pattern, data, payload.Updated, payload.Deleted)
	} else {
		if payload.Updated > 0 || payload.Deleted > 0 {
			lib.Printf("metadata updated: %d, deleted: %d\n", payload.Updated, payload.Deleted)
		}
	}
}

func processFixturesMetadata(ctx *lib.Ctx, pfixtures *[]lib.Fixture) {
	if ctx.SkipMetadata || ctx.OnlyP2O || (ctx.DryRun && !ctx.DryRunAllowMetadata) {
		return
	}
	fixtures := *pfixtures
	lib.Printf("Processing %d fixtures metadata\n", len(fixtures))
	md := make(map[[3]string]struct{})
	for _, fixture := range fixtures {
		m := fixture.Metadata
		if len(m.DataSources) == 0 || len(m.WorkingGroups) == 0 {
			continue
		}
		data := getMetadataQueries(ctx, &m)
		for _, item := range data {
			md[item] = struct{}{}
		}
	}
	if len(md) == 0 {
		return
	}
	thrN := lib.GetThreadsNum(ctx)
	if thrN > 8 {
		thrN = int(math.Round(math.Sqrt(float64(thrN))))
	}
	lib.Printf("%d metadata objects to process using %d threads\n", len(md), thrN)
	if thrN > 1 {
		ch := make(chan struct{})
		nThreads := 0
		for item := range md {
			go processMetadataItem(ch, ctx, item)
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
		for item := range md {
			processMetadataItem(nil, ctx, item)
		}
	}
	lib.Printf("Processing fixtures metadata finished\n")
}

func main() {
	var ctx lib.Ctx
	dtStart := time.Now()
	ctx.Init()
	// IMPL
	/*
		processFixtureFiles(&ctx, lib.GetFixtures(&ctx, ""))
		if 1 == 1 {
			os.Exit(1)
		}
	*/
	if ctx.DryRun {
		lib.Printf("Running in dry-run mode\n")
	}
	if ctx.OnlyValidate {
		validateFixtureFiles(&ctx, lib.GetFixtures(&ctx, ""))
	} else {
		lib.Printf("da-ds configuration: %+v\n", dadsTasks)
		err := ensureGrimoireStackAvail(&ctx)
		if err != nil {
			lib.Fatalf("Grimoire stack not available: %+v\n", err)
		}
		go finishAfterTimeout(ctx)
		processFixtureFiles(&ctx, lib.GetFixtures(&ctx, ""))
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
		err = cacheTopContributors(&ctx)
		if err != nil {
			lib.Printf("Cache top contributors result: %+v\n", err)
		}
		dtEnd := time.Now()
		lib.Printf("Sync time: %v\n", dtEnd.Sub(dtStart))
	}
}
