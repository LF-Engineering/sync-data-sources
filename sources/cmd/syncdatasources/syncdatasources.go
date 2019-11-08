package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	lib "github.com/LF-Engineering/sync-data-sources/sources"
	yaml "gopkg.in/yaml.v2"
)

func ensureGrimoireStackAvail(ctx *lib.Ctx) error {
	lib.Printf("Checking grimoire stack availability\n")
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
	lib.Printf("Grimoire stack available\n%s\n", info)
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
		lib.Printf("Now processing %d fixture files using MT%d version\n", len(fixtureFiles), thrN)
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
		lib.Printf("Final threads join\n")
		for nThreads > 0 {
			fixture := <-ch
			nThreads--
			fixtures = append(fixtures, fixture)
		}
	} else {
		lib.Printf("Now processing %d fixture files using ST version\n", len(fixtureFiles))
		for _, fixtureFile := range fixtureFiles {
			if fixtureFile == "" {
				continue
			}
			fixtures = append(fixtures, processFixtureFile(nil, ctx, fixtureFile))
		}
	}
	if ctx.Debug > 0 {
		lib.Printf("Fixtures: %+v\n", fixtures)
	}
	return nil
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
	fmt.Printf("Time: %v\n", dtEnd.Sub(dtStart))
}
