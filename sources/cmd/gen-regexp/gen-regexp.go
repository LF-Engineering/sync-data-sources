package main

import (
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	lib "github.com/LF-Engineering/sync-data-sources/sources"
	// "github.com/google/go-github/v33/github"
	"github.com/google/go-github/github"
	yaml "gopkg.in/yaml.v2"
)

func processFixtureFile(ch chan lib.Fixture, ctx *lib.Ctx, fixtureFile string) (fixture lib.Fixture) {
	defer func() {
		if ch != nil {
			ch <- fixture
		}
	}()
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
	slug := fixture.Native.Slug
	if slug == "" {
		lib.Fatalf("Fixture file %s 'native' property has no 'slug' property (or is empty)\n", fixtureFile)
	}
	if fixture.Disabled == true {
		return
	}
	return
}

func processEndpoint(ctx *lib.Ctx, ep *lib.RawEndpoint, git bool, key [2]string, orgsMap, reposMap, resMap map[[2]string]map[string]struct{}, cache map[string][]string) {
	keyAll := [2]string{"", ""}
	if strings.HasPrefix(ep.Name, `regexp:`) {
		re := ep.Name[7:]
		if strings.HasPrefix(re, "(?i)") {
			re = re[4:]
		}
		if strings.HasPrefix(re, "^") {
			re = re[1:]
		}
		if strings.HasSuffix(re, "$") {
			re = re[:len(re)-1]
		}
		resMap[keyAll][re] = struct{}{}
		_, ok := resMap[key]
		if !ok {
			resMap[key] = make(map[string]struct{})
		}
		resMap[key][re] = struct{}{}
		return
	}
	if git && !strings.Contains(ep.Name, `://github.com/`) {
		return
	}
	if len(ep.Flags) == 0 {
		ary := strings.Split(ep.Name, "/")
		tokens := []string{}
		for _, token := range ary {
			if token != "" {
				tokens = append(tokens, token)
			}
		}
		if len(tokens) < 3 {
			return
		}
		l := len(tokens)
		r := tokens[l-2] + "/" + tokens[l-1]
		reposMap[keyAll][r] = struct{}{}
		_, ok := reposMap[key]
		if !ok {
			reposMap[key] = make(map[string]struct{})
		}
		reposMap[key][r] = struct{}{}
		return
	}
	tp, ok := ep.Flags["type"]
	if !ok {
		return
	}
	if tp != lib.GitHubOrg && tp != lib.GitHubUser {
		return
	}
	if len(ep.Skip) == 0 && len(ep.Only) == 0 {
		ary := strings.Split(ep.Name, "/")
		tokens := []string{}
		for _, token := range ary {
			if token != "" {
				tokens = append(tokens, token)
			}
		}
		if len(tokens) < 2 {
			return
		}
		l := len(tokens)
		o := tokens[l-1]
		orgsMap[keyAll][o] = struct{}{}
		_, ok := orgsMap[key]
		if !ok {
			orgsMap[key] = make(map[string]struct{})
		}
		orgsMap[key][o] = struct{}{}
		return
	}
	arr := strings.Split(ep.Name, "/")
	ary := []string{}
	l := len(arr) - 1
	for i, s := range arr {
		if i == l && s == "" {
			break
		}
		ary = append(ary, s)
	}
	lAry := len(ary)
	path := ary[lAry-1]
	root := strings.Join(ary[0:lAry-1], "/")
	cacheKey := path + ":" + strings.Join(ep.Skip, ",") + ":" + strings.Join(ep.Only, ",")
	repos, ok := cache[cacheKey]
	if !ok {
		if len(ep.SkipREs) == 0 {
			for _, skip := range ep.Skip {
				skipRE, err := regexp.Compile(skip)
				lib.FatalOnError(err)
				ep.SkipREs = append(ep.SkipREs, skipRE)
			}
		}
		if len(ep.OnlyREs) == 0 {
			for _, only := range ep.Only {
				onlyRE, err := regexp.Compile(only)
				lib.FatalOnError(err)
				ep.OnlyREs = append(ep.OnlyREs, onlyRE)
			}
		}
		gctx, gc := lib.GHClient(ctx)
		hint, _, rem, wait := lib.GetRateLimits(gctx, ctx, gc, true)
		for {
			if rem[hint] <= 5 {
				lib.Printf("All GH API tokens are overloaded, maximum points %d, waiting %+v\n", rem[hint], wait[hint])
				time.Sleep(time.Duration(1) * time.Second)
				time.Sleep(wait[hint])
				hint, _, rem, wait = lib.GetRateLimits(gctx, ctx, gc, true)
				continue
			}
			break
		}
		var (
			optOrg  *github.RepositoryListByOrgOptions
			optUser *github.RepositoryListOptions
		)
		if tp == lib.GitHubOrg {
			optOrg = &github.RepositoryListByOrgOptions{Type: "public"}
			optOrg.PerPage = 100
		} else {
			optUser = &github.RepositoryListOptions{Type: "public"}
			optUser.PerPage = 100
		}
		for {
			var (
				repositories []*github.Repository
				response     *github.Response
				err          error
			)
			if tp == lib.GitHubOrg {
				repositories, response, err = gc[hint].Repositories.ListByOrg(gctx, path, optOrg)
			} else {
				repositories, response, err = gc[hint].Repositories.List(gctx, path, optUser)
			}
			lib.FatalOnError(err)
			for _, repo := range repositories {
				if repo.Name != nil {
					name := root + "/" + path + "/" + *(repo.Name)
					repos = append(repos, name)
				}
			}
			if response.NextPage == 0 {
				break
			}
			if tp == lib.GitHubOrg {
				optOrg.Page = response.NextPage
			} else {
				optUser.Page = response.NextPage
			}
		}
		cache[cacheKey] = repos
		// lib.Printf("org/user: %s skip=%+v only=%+v -> %+v\n", ep.Name, ep.Skip, ep.Only, repos)
	}
	for _, repo := range repos {
		included, _ := lib.EndpointIncluded(ctx, ep, repo)
		if !included {
			continue
		}
		ary := strings.Split(repo, "/")
		tokens := []string{}
		for _, token := range ary {
			if token != "" {
				tokens = append(tokens, token)
			}
		}
		if len(tokens) < 3 {
			return
		}
		l := len(tokens)
		r := tokens[l-2] + "/" + tokens[l-1]
		reposMap[keyAll][r] = struct{}{}
		_, ok := reposMap[key]
		if !ok {
			reposMap[key] = make(map[string]struct{})
		}
		reposMap[key][r] = struct{}{}
	}
}

func processFixtures(ctx *lib.Ctx, fixtureFiles []string) {
	thrN := lib.GetThreadsNum(ctx)
	fixtures := []lib.Fixture{}
	if thrN > 1 {
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
		for nThreads > 0 {
			fixture := <-ch
			nThreads--
			if fixture.Disabled != true {
				fixtures = append(fixtures, fixture)
			}
		}
	} else {
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
		lib.Fatalf("No fixtures read, this is error, please define at least one")
	}
	keyAll := [2]string{"", ""}
	keys := make(map[[2]string]struct{})
	orgs := make(map[[2]string]map[string]struct{})
	repos := make(map[[2]string]map[string]struct{})
	res := make(map[[2]string]map[string]struct{})
	orgs[keyAll] = make(map[string]struct{})
	repos[keyAll] = make(map[string]struct{})
	res[keyAll] = make(map[string]struct{})
	cache := make(map[string][]string)
	for _, fixture := range fixtures {
		fSlug := fixture.Native.Slug
		for _, ds := range fixture.DataSources {
			include := false
			git := false
			dss := strings.ToLower(strings.TrimSpace(ds.Slug))
			if dss == "git" {
				include = true
				git = true
			} else {
				ary := strings.Split(dss, "/")
				if ary[0] == "github" {
					include = true
				}
			}
			if !include {
				continue
			}
			for _, ep := range ds.RawEndpoints {
				key := [2]string{fSlug, ep.Project}
				processEndpoint(ctx, &ep, git, key, orgs, repos, res, cache)
				keys[key] = struct{}{}
			}
			for _, ep := range ds.HistEndpoints {
				key := [2]string{fSlug, ep.Project}
				processEndpoint(ctx, &ep, git, key, orgs, repos, res, cache)
				keys[key] = struct{}{}
			}
			for _, project := range ds.Projects {
				proj := project.Name
				if proj == "" {
					lib.Fatalf("Empty project name entry in %+v, data source %s, fixture %s\n", project, ds.Slug, fSlug)
				}
				for _, ep := range project.RawEndpoints {
					eProj := proj
					if ep.Project != "" {
						eProj = ep.Project
					}
					key := [2]string{fSlug, eProj}
					processEndpoint(ctx, &ep, git, key, orgs, repos, res, cache)
					keys[key] = struct{}{}
				}
				for _, ep := range project.HistEndpoints {
					eProj := proj
					if ep.Project != "" {
						eProj = ep.Project
					}
					key := [2]string{fSlug, eProj}
					processEndpoint(ctx, &ep, git, key, orgs, repos, res, cache)
					keys[key] = struct{}{}
				}
			}
		}
	}
	keysAry := [][2]string{}
	keysAry = append(keysAry, keyAll)
	for key := range keys {
		keysAry = append(keysAry, key)
	}
	sort.Slice(
		keysAry,
		func(i, j int) bool {
			a := keysAry[i][0] + "," + keysAry[i][1]
			b := keysAry[j][0] + "," + keysAry[j][1]
			return a < b
		},
	)
	for _, key := range keysAry {
		orgsAry := []string{}
		reposAry := []string{}
		resAry := []string{}
		for org := range orgs[key] {
			orgsAry = append(orgsAry, org)
		}
		for repo := range repos[key] {
			reposAry = append(reposAry, repo)
		}
		for re := range res[key] {
			resAry = append(resAry, re)
		}
		sort.Strings(orgsAry)
		sort.Strings(reposAry)
		sort.Strings(resAry)
		slug := key[0]
		proj := key[1]
		re := ``
		n := 0
		for _, org := range orgsAry {
			re += org + `\/.*|`
			n++
		}
		for _, repo := range reposAry {
			re += strings.Replace(repo, `/`, `\/`, -1) + `|`
			n++
		}
		for _, r := range resAry {
			re += `(` + r + `)|`
			n++
		}
		if n == 0 {
			// lib.Printf("Slug: '%s', Project: '%s': no data\n", slug, proj)
			continue
		}
		if n == 1 {
			re = `^` + re[0:len(re)-1] + `$`
		} else {
			re = `^(` + re[0:len(re)-1] + `)$`
		}
		cre, err := regexp.Compile(re)
		if err != nil {
			lib.Printf("Failed: Slug: '%s', Project: '%s', RE: %s\n", slug, proj, re)
		}
		lib.FatalOnError(err)
		lib.Printf("Slug: '%s', Project: '%s', RE: %v\n", slug, proj, cre)
	}
}

func main() {
	var ctx lib.Ctx
	dtStart := time.Now()
	ctx.TestMode = true
	ctx.Init()
	_ = os.Setenv("SDS_SIMPLE_PRINTF", "1")
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	processFixtures(&ctx, lib.GetFixtures(&ctx, path))
	dtEnd := time.Now()
	lib.Printf("Took: %v\n", dtEnd.Sub(dtStart))
}
