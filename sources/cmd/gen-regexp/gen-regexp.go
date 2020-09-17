package main

import (
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	lib "github.com/LF-Engineering/sync-data-sources/sources"
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

func processEndpoint(ctx *lib.Ctx, ep *lib.RawEndpoint, git bool, orgsMap, reposMap map[string]struct{}, cache map[string][]string) {
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
		reposMap[tokens[l-2]+"/"+tokens[l-1]] = struct{}{}
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
		orgsMap[tokens[l-1]] = struct{}{}
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
	repos, ok := cache[path]
	if !ok {
		gctx, gc := lib.GHClient(ctx)
		hint, _, rem, wait := lib.GetRateLimits(gctx, ctx, gc, true)
		for {
			if rem[hint] <= 100 {
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
		cache[path] = repos
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
		reposMap[tokens[l-2]+"/"+tokens[l-1]] = struct{}{}
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
	orgs := make(map[string]struct{})
	repos := make(map[string]struct{})
	cache := make(map[string][]string)
	for _, fixture := range fixtures {
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
				processEndpoint(ctx, &ep, git, orgs, repos, cache)
			}
			for _, project := range ds.Projects {
				for _, ep := range project.RawEndpoints {
					processEndpoint(ctx, &ep, git, orgs, repos, cache)
				}
			}
		}
	}
	orgsAry := []string{}
	reposAry := []string{}
	for org := range orgs {
		orgsAry = append(orgsAry, org)
	}
	for repo := range repos {
		reposAry = append(reposAry, repo)
	}
	sort.Strings(orgsAry)
	sort.Strings(reposAry)
	re := `^(`
	for _, org := range orgsAry {
		re += org + `\/.*|`
	}
	for _, repo := range reposAry {
		re += strings.Replace(repo, `/`, `\/`, -1) + `|`
	}
	re = re[0:len(re)-1] + `)$`
	cre, err := regexp.Compile(re)
	lib.FatalOnError(err)
	lib.Printf("Final regexp:\n================\n%v\n================\n", cre)
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
