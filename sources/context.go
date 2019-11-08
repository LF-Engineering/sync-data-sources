package syncdatasources

import (
	"fmt"
	"os"
	"strconv"
)

// Ctx - environment context packed in structure
type Ctx struct {
	Debug      int    // From SDS_DEBUG Debug level: 0-no, 1-info, 2-verbose
	CmdDebug   int    // From SDS_CMDDEBUG Commands execution Debug level: 0-no, 1-only output commands, 2-output commands and their output, 3-output full environment as well, default 0
	ST         bool   // From SDS_ST true: use single threaded version, false: use multi threaded version, default false
	NCPUs      int    // From SDS_NCPUS, set to override number of CPUs to run, this overwrites SDS_ST, default 0 (which means do not use it)
	CtxOut     bool   // From SDS_CTXOUT output all context data (this struct), default false
	LogTime    bool   // From SDS_SKIPTIME, output time with all lib.Printf(...) calls, default true, use SDS_SKIPTIME to disable
	ExecFatal  bool   // default true, set this manually to false to avoid lib.ExecCommand calling os.Exit() on failure and return error instead
	ExecQuiet  bool   // default false, set this manually to true to have quiet exec failures
	ExecOutput bool   // default false, set to true to capture commands STDOUT
	ElasticURL string // From SDS_ES_URL, ElasticSearch URL, default http://127.0.0.1:9200
	TestMode   bool   // True when running tests
}

// Init - get context from environment variables
func (ctx *Ctx) Init() {
	ctx.ExecFatal = true
	ctx.ExecQuiet = false
	ctx.ExecOutput = false

	// Debug
	if os.Getenv("SDS_DEBUG") == "" {
		ctx.Debug = 0
	} else {
		debugLevel, err := strconv.Atoi(os.Getenv("SDS_DEBUG"))
		FatalNoLog(err)
		if debugLevel != 0 {
			ctx.Debug = debugLevel
		}
	}
	// CmdDebug
	if os.Getenv("SDS_CMDDEBUG") == "" {
		ctx.CmdDebug = 0
	} else {
		debugLevel, err := strconv.Atoi(os.Getenv("SDS_CMDDEBUG"))
		FatalNoLog(err)
		ctx.CmdDebug = debugLevel
	}
	ctx.CtxOut = os.Getenv("SDS_CTXOUT") != ""

	// Threading
	ctx.ST = os.Getenv("SDS_ST") != ""
	// NCPUs
	if os.Getenv("SDS_NCPUS") == "" {
		ctx.NCPUs = 0
	} else {
		nCPUs, err := strconv.Atoi(os.Getenv("SDS_NCPUS"))
		FatalNoLog(err)
		if nCPUs > 0 {
			ctx.NCPUs = nCPUs
			if ctx.NCPUs == 1 {
				ctx.ST = true
			}
		}
	}

	// Log Time
	ctx.LogTime = os.Getenv("SDS_SKIPTIME") == ""

	// ElasticSearch
	ctx.ElasticURL = os.Getenv("SDS_ES_URL")
	if ctx.ElasticURL == "" {
		ctx.ElasticURL = "http://127.0.0.1:9200"
	}

	// Context out if requested
	if ctx.CtxOut {
		ctx.Print()
	}
}

// Print context contents
func (ctx *Ctx) Print() {
	fmt.Printf("Environment Context Dump\n%+v\n", ctx)
}
