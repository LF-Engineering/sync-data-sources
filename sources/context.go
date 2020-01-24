package syncdatasources

import (
	"fmt"
	"os"
	"strconv"
)

// Ctx - environment context packed in structure
type Ctx struct {
	Debug            int    // From SDS_DEBUG Debug level: 0-no, 1-info, 2-verbose
	CmdDebug         int    // From SDS_CMDDEBUG Commands execution Debug level: 0-no, 1-only output commands, 2-output commands and their output, 3-output full environment as well, default 0
	MaxRetry         int    // From SDS_MAXRETRY Try to run grimoire stack (perceval, p2o.py etc) that many times before reporting failure, default 3 (1 original and 2 more attempts).
	ST               bool   // From SDS_ST true: use single threaded version, false: use multi threaded version, default false
	NCPUs            int    // From SDS_NCPUS, set to override number of CPUs to run, this overwrites SDS_ST, default 0 (which means do not use it)
	CtxOut           bool   // From SDS_CTXOUT output all context data (this struct), default false
	LogTime          bool   // From SDS_SKIPTIME, output time with all lib.Printf(...) calls, default true, use SDS_SKIPTIME to disable
	ExecFatal        bool   // default true, set this manually to false to avoid lib.ExecCommand calling os.Exit() on failure and return error instead
	ExecQuiet        bool   // default false, set this manually to true to have quiet exec failures
	ExecOutput       bool   // default false, set to true to capture commands STDOUT
	ExecOutputStderr bool   // default false, set to true to capture commands STDOUT
	ElasticURL       string // From SDS_ES_URL, ElasticSearch URL, default http://127.0.0.1:9200
	EsBulkSize       int    // From SDS_ES_BULKSIZE, ElasticSearch bulk size when enriching data, defaults to 0 which means "not specified" (10000)
	NodeHash         bool   // From SDS_NODE_HASH, if set it will generate hashes for each task and only execute them when node number matches hash result
	NodeNum          int    // From SDS_NODE_NUM, set number of nodes, so hashing function will return [0, ... n)
	NodeIdx          int    // From SDS_NODE_NUM, set number of current node, so only hashes matching this node will run
	DryRun           bool   // From SDS_DRY_RUN, if set it will do everything excluding actual grimoire stack execution (will report success for all commands instead)
	DryRunCode       int    // From SDS_DRY_RUN_CODE, dry run exit code, default 0 which means success, possible values 1, 2, 3, 4
	DryRunSeconds    int    // From SDS_DRY_RUN_SECONDS, simulate each dry run command taking some time to execute
	DryRunAllowSSH   bool   // From SDS_DRY_RUN_ALLOW_SSH, if set it will allow setting SSH keys in dry run mode
	TimeoutSeconds   int    // From SDS_TIMEOUT_SECONDS, set entire program execution timeout, program will finish with return code 2 if anything still runs after this time, default 47 h 45 min = 171900
	NLongest         int    // From SDS_N_LONGEST, number of longest running tasks to display in stats, default 10
	SkipSH           bool   // Fro SDS_SKIP_SH, if set sorting hata database processing will be skipped
	StripErrorSize   int    // From SDS_STRIP_ERROR_SIZE, default 1024, error messages longer that this value will be stripped by half of this value from beginning and from end, so for 1024 error 4000 bytes long will be 512 bytes from the beginning ... 512 from the end
	GitHubOAuth      string // From SDS_GITHUB_OAUTH, if not set it attempts to use public access, if contains "/" it will assume that it contains file name, if "," found then it will assume that this is a list of OAuth tokens instead of just one
	LatestItems      bool   // From SDS_LATEST_ITEMS, if set pass "latest items" or similar flag to the p2o.py backend (that should be handled by p2o.py using ES, so this is probably not a good ide, git backend, for example, can return no data then)
	CSVPrefix        string // From SDS_CSV_PREFIX, CSV logs filename prefix, default "jobs", so files would be "/root/.perceval/jobs_I_N.csv"
	Silent           bool   // From SDS_SILENT, skip p2o.py debug mode if set, else it will pass "-g" flag to 'p2o.py' call
	SkipData         bool   // From SDS_SKIP_DATA, if set - it will not run incremental data sync
	SkipAffs         bool   // From SDS_SKIP_AFFS, if set - it will not run p2o.py historical affiliations enrichment (--only-enrich --refresh-identities --no_incremental)
	SkipAliases      bool   // From SDS_SKIP_ALIASES, if set - sds will not attempt to create index aliases and will not attempt to drop unused aliases
	SkipDropUnused   bool   // From SDS_SKIP_DROP_UNUSED, if set - it will not attempt to drop unused indexes and aliases
	NoMultiAliases   bool   // From SDS_NO_MULTI_ALIASES, if set alias can only be defined for single index, so only one index maps to any alias, if not defined multiple input indexs can be accessed through a single alias
	CleanupAliases   bool   // From SDS_CLEANUP_ALIASES, will delete all aliases before creating them (so it can delete old indexes that were pointed by given alias before adding new indexes to it (single or multiple))
	ScrollWait       int    // From SDS_SCROLL_WAIT, will pass 'p2o.py' '--scroll-wait=N' if set - this is to specify time to wait for available scrolls (in seconds)
	ScrollSize       int    // From SDS_SCROLL_SIZE, ElasticSearch scroll size when enriching data, default 1000
	TestMode         bool   // True when running tests
	ShUser           string // Sorting Hat database parameters
	ShHost           string
	ShPass           string
	ShDB             string
}

// Init - get context from environment variables
func (ctx *Ctx) Init() {
	ctx.ExecFatal = true
	ctx.ExecQuiet = false
	ctx.ExecOutput = false
	ctx.ExecOutputStderr = false

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
	// MaxRetry
	if os.Getenv("SDS_MAXRETRY") == "" {
		ctx.MaxRetry = 2
	} else {
		maxRetry, err := strconv.Atoi(os.Getenv("SDS_MAXRETRY"))
		FatalNoLog(err)
		ctx.MaxRetry = maxRetry
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

	// Dry Run mode
	ctx.DryRun = os.Getenv("SDS_DRY_RUN") != ""
	ctx.DryRunAllowSSH = os.Getenv("SDS_DRY_RUN_ALLOW_SSH") != ""
	if os.Getenv("SDS_DRY_RUN_CODE") == "" {
		ctx.DryRunCode = 0
	} else {
		code, err := strconv.Atoi(os.Getenv("SDS_DRY_RUN_CODE"))
		FatalNoLog(err)
		if code >= 1 && code <= 4 {
			ctx.DryRunCode = code
		}
	}
	if os.Getenv("SDS_DRY_RUN_SECONDS") == "" {
		ctx.DryRunSeconds = 0
	} else {
		secs, err := strconv.Atoi(os.Getenv("SDS_DRY_RUN_SECONDS"))
		FatalNoLog(err)
		if secs > 0 {
			ctx.DryRunSeconds = secs
		}
	}

	// Skip SortingHat mode
	ctx.SkipSH = os.Getenv("SDS_SKIP_SH") != ""

	// Sorting Hat DB parameters
	ctx.ShUser = os.Getenv("SH_USER")
	ctx.ShHost = os.Getenv("SH_HOST")
	ctx.ShPass = os.Getenv("SH_PASS")
	ctx.ShDB = os.Getenv("SH_DB")

	if !ctx.TestMode && !ctx.DryRun && !ctx.SkipSH && (ctx.ShUser == "" || ctx.ShHost == "" || ctx.ShPass == "" || ctx.ShDB == "") {
		fmt.Printf("%v %v %s %s %s %s\n", ctx.TestMode, ctx.SkipSH, ctx.ShUser, ctx.ShHost, ctx.ShPass, ctx.ShDB)
		FatalNoLog(fmt.Errorf("SortingHat parameters (user, host, password, db) must all be defined unless skiping SortingHat"))
	}

	// Log Time
	ctx.LogTime = os.Getenv("SDS_SKIPTIME") == ""

	// ElasticSearch
	ctx.ElasticURL = os.Getenv("SDS_ES_URL")
	if ctx.ElasticURL == "" {
		ctx.ElasticURL = "http://127.0.0.1:9200"
	}
	// ES bulk size
	if os.Getenv("SDS_ES_BULKSIZE") == "" {
		ctx.EsBulkSize = 0
	} else {
		bulkSize, err := strconv.Atoi(os.Getenv("SDS_ES_BULKSIZE"))
		FatalNoLog(err)
		if bulkSize > 0 {
			ctx.EsBulkSize = bulkSize
		}
	}

	// Node hash support
	ctx.NodeHash = os.Getenv("SDS_NODE_HASH") != ""
	if os.Getenv("SDS_NODE_NUM") == "" {
		ctx.NodeNum = 1
	} else {
		nodeNum, err := strconv.Atoi(os.Getenv("SDS_NODE_NUM"))
		FatalNoLog(err)
		if nodeNum > 0 {
			ctx.NodeNum = nodeNum
		} else {
			ctx.NodeNum = 1
		}
	}
	if os.Getenv("SDS_NODE_IDX") == "" {
		ctx.NodeIdx = 0
	} else {
		nodeIdx, err := strconv.Atoi(os.Getenv("SDS_NODE_IDX"))
		FatalNoLog(err)
		if nodeIdx >= 0 && nodeIdx < ctx.NodeNum {
			ctx.NodeIdx = nodeIdx
		}
	}

	// Timeout
	if os.Getenv("SDS_TIMEOUT_SECONDS") == "" {
		ctx.TimeoutSeconds = 171900
	} else {
		secs, err := strconv.Atoi(os.Getenv("SDS_TIMEOUT_SECONDS"))
		FatalNoLog(err)
		if secs > 0 {
			ctx.TimeoutSeconds = secs
		} else {
			ctx.TimeoutSeconds = 171900
		}
	}

	// Longest running tasks stats
	if os.Getenv("SDS_N_LONGEST") == "" {
		ctx.NLongest = 10
	} else {
		n, err := strconv.Atoi(os.Getenv("SDS_N_LONGEST"))
		FatalNoLog(err)
		if n > 0 {
			ctx.NLongest = n
		} else {
			ctx.NLongest = 10
		}
	}

	// Strip error size (default 512)
	if os.Getenv("SDS_STRIP_ERROR_SIZE") == "" {
		ctx.StripErrorSize = 1024
	} else {
		n, err := strconv.Atoi(os.Getenv("SDS_STRIP_ERROR_SIZE"))
		FatalNoLog(err)
		if n > 1 {
			ctx.StripErrorSize = n
		} else {
			ctx.StripErrorSize = 1024
		}
	}

	// GitHub OAuth
	ctx.GitHubOAuth = os.Getenv("SDS_GITHUB_OAUTH")

	// Latest items p2o.py backend flag support
	ctx.LatestItems = os.Getenv("SDS_LATEST_ITEMS") != ""

	// CSV logs prefix
	ctx.CSVPrefix = os.Getenv("SDS_CSV_PREFIX")
	if ctx.CSVPrefix == "" {
		ctx.CSVPrefix = "jobs"
	}

	// Scroll wait p2o.py --scroll-wait 900
	if os.Getenv("SDS_SCROLL_WAIT") == "" {
		ctx.ScrollWait = 0
	} else {
		scrollWait, err := strconv.Atoi(os.Getenv("SDS_SCROLL_WAIT"))
		FatalNoLog(err)
		if scrollWait > 0 {
			ctx.ScrollWait = scrollWait
		}
	}
	// ES scroll size p2o.py --scroll-size 1000
	if os.Getenv("SDS_SCROLL_SIZE") == "" {
		ctx.ScrollSize = 1000
	} else {
		scrollSize, err := strconv.Atoi(os.Getenv("SDS_SCROLL_SIZE"))
		FatalNoLog(err)
		if scrollSize >= 0 {
			ctx.ScrollSize = scrollSize
		}
	}

	// Skip -d p2o.py flag
	ctx.Silent = os.Getenv("SDS_SILENT") != ""

	// Skip data/affs mode
	ctx.SkipData = os.Getenv("SDS_SKIP_DATA") != ""
	ctx.SkipAffs = os.Getenv("SDS_SKIP_AFFS") != ""
	ctx.SkipAliases = os.Getenv("SDS_SKIP_ALIASES") != ""
	ctx.NoMultiAliases = os.Getenv("SDS_NO_MULTI_ALIASES") != ""
	ctx.CleanupAliases = os.Getenv("SDS_CLEANUP_ALIASES") != ""
	ctx.SkipDropUnused = os.Getenv("SDS_SKIP_DROP_UNUSED") != ""

	// Forbidden configurations
	if ctx.SkipSH && !ctx.SkipAffs {
		FatalNoLog(fmt.Errorf("you cannot skip SortingHat and not skip affiliations at the same time"))
	}
	if ctx.SkipData && ctx.SkipAffs && ctx.SkipAliases {
		FatalNoLog(fmt.Errorf("you cannot skip incremental data sync, historical affiliations sync and aliases at the same time"))
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
