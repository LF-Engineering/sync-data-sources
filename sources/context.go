package syncdatasources

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
)

// Ctx - environment context packed in structure
type Ctx struct {
	Debug                   int            // From SDS_DEBUG Debug level: 0-no, 1-info, 2-verbose
	CmdDebug                int            // From SDS_CMDDEBUG Commands execution Debug level: 0-no, 1-only output commands, 2-output commands and their output, 3-output full environment as well, default 0
	MaxRetry                int            // From SDS_MAXRETRY Try to run grimoire stack (perceval, p2o.py etc) that many times before reporting failure, default 0 (1 original - always runs and 0 more attempts).
	ST                      bool           // From SDS_ST true: use single threaded version, false: use multi threaded version, default false
	NCPUs                   int            // From SDS_NCPUS, set to override number of CPUs to run, this overwrites SDS_ST, default 0 (which means do not use it, use all CPU reported by go library)
	NCPUsScale              float64        // From SDS_NCPUS_SCALE, scale number of CPUs, for example 2.0 will report number of cpus 2.0 the number of actually available CPUs
	FixturesRE              *regexp.Regexp // From SDS_FIXTURES_RE - you can set regular expression specifying which fixtures should be processed, default empty which means all.
	DatasourcesRE           *regexp.Regexp // From SDS_DATASOURCES_RE - you can set regular expression specifying which datasources should be processed, default empty which means all.
	ProjectsRE              *regexp.Regexp // From SDS_PROJECTS_RE - you can set regular expression specifying which projects/subprojects should be processed, default empty which means all.
	EndpointsRE             *regexp.Regexp // From SDS_ENDPOINTS_RE - you can set regular expression specifying which endpoints/origins should be processed, default empty which means all.
	TasksRE                 *regexp.Regexp // From SDS_TASKS_RE - you can set regular expression specifying which tasks should be processed, default empty which means all, exampel task is "sds-lfn-onap-slack:SLACK_CHAN_ID"
	FixturesSkipRE          *regexp.Regexp // From SDS_FIXTURES_SKIP_RE - you can set regular expression specifying which fixtures should be skipped, default empty which means none.
	DatasourcesSkipRE       *regexp.Regexp // From SDS_DATASOURCES_SKIP_RE - you can set regular expression specifying which datasources should be skipped, default empty which means none.
	ProjectsSkipRE          *regexp.Regexp // From SDS_PROJECTS_SKIP_RE - you can set regular expression specifying which projects/subprojects should be slkipped, default empty which means none.
	EndpointsSkipRE         *regexp.Regexp // From SDS_ENDPOINTS_SKIP_RE - you can set regular expression specifying which endpoints/origins should be skipped, default empty which means none.
	TasksSkipRE             *regexp.Regexp // From SDS_TASKS_SKIP_RE - you can set regular expression specifying which tasks should be skipped, default empty which means none.
	CtxOut                  bool           // From SDS_CTXOUT output all context data (this struct), default false
	LogTime                 bool           // From SDS_SKIPTIME, output time with all lib.Printf(...) calls, default true, use SDS_SKIPTIME to disable
	ExecFatal               bool           // default true, set this manually to false to avoid lib.ExecCommand calling os.Exit() on failure and return error instead
	ExecQuiet               bool           // default false, set this manually to true to have quiet exec failures
	ExecOutput              bool           // default false, set to true to capture commands STDOUT
	ExecOutputStderr        bool           // default false, set to true to capture commands STDOUT
	ElasticURL              string         // From SDS_ES_URL, ElasticSearch URL, default http://127.0.0.1:9200
	EsBulkSize              int            // From SDS_ES_BULKSIZE, ElasticSearch bulk size when enriching data, defaults to 0 which means "not specified" (10000)
	NodeHash                bool           // From SDS_NODE_HASH, if set it will generate hashes for each task and only execute them when node number matches hash result
	NodeNum                 int            // From SDS_NODE_NUM, set number of nodes, so hashing function will return [0, ... n)
	NodeIdx                 int            // From SDS_NODE_IDX, set number of current node, so only hashes matching this node will run
	NodeSettleTime          int            // From SDS_NODE_SETTLE_TIME, number of seconds that master gives nodes to start-up and wait for ES mutex9es) to sync with master node, default 10 (in seconds)
	DryRun                  bool           // From SDS_DRY_RUN, if set it will do everything excluding actual grimoire stack execution (will report success for all commands instead)
	DryRunCode              int            // From SDS_DRY_RUN_CODE, dry run exit code, default 0 which means success, possible values 1, 2, 3, 4
	DryRunCodeRandom        bool           // From SDS_DRY_RUN_CODE_RANDOM, dry run exit code, will return random value from 0 to 5
	DryRunSeconds           int            // From SDS_DRY_RUN_SECONDS, simulate each dry run command taking some time to execute
	DryRunSecondsRandom     bool           // From SDS_DRY_RUN_SECONDS_RANDOM, make running time from 0 to SDS_DRY_RUN_SECONDS (in ms resolution)
	DryRunAllowSSH          bool           // From SDS_DRY_RUN_ALLOW_SSH, if set it will allow setting SSH keys in dry run mode
	DryRunAllowFreq         bool           // From SDS_DRY_RUN_ALLOW_FREQ, if set it will allow processing sync frequency data in dry run mode
	DryRunAllowMtx          bool           // From SDS_DRY_RUN_ALLOW_MTX, if set it will allow handling ES mutexes (for nodes concurrency support) in dry run mode
	DryRunAllowRename       bool           // From SDS_DRY_RUN_ALLOW_RENAME, if set it will allow handling ES index renaming in dry run mode
	DryRunAllowOrigins      bool           // From SDS_DRY_RUN_ALLOW_ORIGINS, if set it will allow fetching external indices origins list in dry run mode
	DryRunAllowDedup        bool           // From SDS_DRY_RUN_ALLOW_DEDUP, if set it will allow dedup bitergia data by deleting origins shared with existing SDS indices
	DryRunAllowProject      bool           // From SDS_DRY_RUN_ALLOW_PROJECT, if set it will allow running set project by SDS (on endpoints with project set and p2o mode set to false)
	DryRunAllowSyncInfo     bool           // From SDS_DRY_RUN_ALLOW_SYNC_INFO, if set it will allow setting sync info in sds-sync-info index
	DryRunAllowSortDuration bool           // From SDS_DRY_RUN_ALLOW_SORT_DURATION, if set it will allow setting sync info in sds-sync-info index
	DryRunAllowSSAW         bool           // From SDS_DRY_RUN_ALLOW_SSAW, if set - self serve affiliation workflow notification will be enabled in dry run mode
	DryRunAllowMerge        bool           // From SDS_DRY_RUN_ALLOW_MERGE, if set it will allow calling DA-affiliation merge_all API after all tasks finished in dry run mode
	DryRunAllowHideEmails   bool           // From SDS_DRY_RUN_ALLOW_HIDE_EMAILS, if set it will allow calling DA-affiliation hide_emails API in dry run mode
	TimeoutSeconds          int            // From SDS_TIMEOUT_SECONDS, set entire program execution timeout, program will finish with return code 2 if anything still runs after this time, default 47 h 45 min = 171900
	TaskTimeoutSeconds      int            // From SDS_TASK_TIMEOUT_SECONDS, set single p2o.py task execution timeout, default is 36000s (10 hours)
	NLongest                int            // From SDS_N_LONGEST, number of longest running tasks to display in stats, default 10
	SkipSH                  bool           // From SDS_SKIP_SH, if set sorting hata database processing will be skipped
	SkipData                bool           // From SDS_SKIP_DATA, if set - it will not run incremental data sync
	SkipAffs                bool           // From SDS_SKIP_AFFS, if set - it will not run p2o.py historical affiliations enrichment (--only-enrich --refresh-identities --no_incremental)
	SkipAliases             bool           // From SDS_SKIP_ALIASES, if set - sds will not attempt to create index aliases and will not attempt to drop unused aliases
	SkipDropUnused          bool           // From SDS_SKIP_DROP_UNUSED, if set - it will not attempt to drop unused indexes and aliases
	SkipCheckFreq           bool           // From SDS_SKIP_CHECK_FREQ, will skip maximum task sync frequency if set
	SkipEsData              bool           // From SDS_SKIP_ES_DATA, will totally skip anything related to "sdsdata" index processing (storing SDS state)
	SkipEsLog               bool           // From SDS_SKIP_ES_LOG, will skip writing logs to "sdslog" index
	SkipDedup               bool           // From SDS_SKIP_DEDUP, will skip attemting to dedup data shared on existing SDS index and external bitergia index (by deleting shared origin data from the external Bitergia index)
	SkipExternal            bool           // From SDS_SKIP_EXTERNAL, will skip any external indices processing: enrichments, deduplication, affiliations etc.
	SkipProject             bool           // From SDS_SKIP_PROJECT, will skip adding column "project": "project name" on all documents where origin = endpoint name, will also add timestamp column "project_ts", so next run can start on documents newer than that
	SkipProjectTS           bool           // From SDS_SKIP_PROJECT_TS, will add project column as described above, without using "project_ts" column to determine from which document to start
	SkipSyncInfo            bool           // From SDS_SKIP_SYNC_INFO, will skip adding sync info to sds-sync-info index
	SkipValGitHubAPI        bool           // From SDS_SKIP_VALIDATE_GITHUB_API, will not process GitHub orgs/users in validate step (will not attempt to get org's/user's repo lists)
	SkipSSAW                bool           // From SDS_SKIP_SSAW, if set - it will completelly skip SSAW processing
	SkipSortDuration        bool           // From SDS_SKIP_SORT_DURATION, if set - it will skip tasks run order by last running time duration desc
	SkipMerge               bool           // From SDS_SKIP_MERGE, if set - it will skip calling DA-affiliation merge_all API after all tasks finished
	SkipHideEmails          bool           // From SDS_SKIP_HIDE_EMAILS, if set - it will skip calling DA-affiliation hide_emails API
	SkipP2O                 bool           // From SDS_SKIP_P2O, if set - it will skip all p2o tasks and execute everything else
	StripErrorSize          int            // From SDS_STRIP_ERROR_SIZE, default 16384, error messages longer that this value will be stripped by this value from beginning and from end, so for 16384 error 64000 bytes long will be 16384 bytes from the beginning \n(...)\n 16384 from the end
	GitHubOAuth             string         // From SDS_GITHUB_OAUTH, if not set it attempts to use public access, if contains "/" it will assume that it contains file name, if "," found then it will assume that this is a list of OAuth tokens instead of just one
	LatestItems             bool           // From SDS_LATEST_ITEMS, if set pass "latest items" or similar flag to the p2o.py backend (that should be handled by p2o.py using ES, so this is probably not a good ide, git backend, for example, can return no data then)
	CSVPrefix               string         // From SDS_CSV_PREFIX, CSV logs filename prefix, default "jobs", so files would be "/root/.perceval/jobs_I_N.csv"
	Silent                  bool           // From SDS_SILENT, skip p2o.py debug mode if set, else it will pass "-g" flag to 'p2o.py' call
	NoMultiAliases          bool           // From SDS_NO_MULTI_ALIASES, if set alias can only be defined for single index, so only one index maps to any alias, if not defined multiple input indexies can be accessed through a single alias (so it can have data from more than 1 p2o.py call)
	CleanupAliases          bool           // From SDS_CLEANUP_ALIASES, will delete all aliases before creating them (so it can delete old indexes that were pointed by given alias before adding new indexes to it (single or multiple))
	ScrollWait              int            // From SDS_SCROLL_WAIT, will pass 'p2o.py' '--scroll-wait=N' if set - this is to specify time to wait for available scrolls (in seconds)
	ScrollSize              int            // From SDS_SCROLL_SIZE, ElasticSearch scroll size when enriching data, default 100
	MaxDeleteTrials         int            // From SDS_MAX_DELETE_TRIALS, default 10
	MaxMtxWait              int            // From SDS_MAX_MTX_WAIT, in seconds, default 900s
	MaxMtxWaitFatal         bool           // From SDS_MAX_MTX_WAIT_FATAL, exit with error when waiting for mutex is more than configured amount of time
	EnrichExternalFreq      time.Duration  // From SDS_ENRICH_EXTERNAL_FREQ, how often enrich external indexes, default is 48h which means no more often than 48h.
	SSAWURL                 string         // From SDS_SSAW_URL, URL of the SSAW service to send notification to, must be set or SkipSSAW flag must be set
	SSAWFreq                int            // From SDS_SSAW_FREQ, default 0 - means call SSAW only when all tasks are finished, setting to 30 will spawn a separate thread that will call SSAW every 30 seconds, minimal frequency (when set) is 20
	OnlyValidate            bool           // From SDS_ONLY_VALIDATE, if defined, SDS will only validate fixtures and exit 0 if all of them are valide, non-zero + error message otherwise
	OnlyP2O                 bool           // From SDS_ONLY_P2O, if defined, SDS will only run p2o tasks, will not do anything else.
	AffiliationAPIURL       string         // From AFFILIATION_API_URL - DA affiliations API url
	Auth0URL                string         // From AUTH0_URL: Auth0 parameters for obtaining DA-affiliation API token
	Auth0Audience           string         // From AUTH0_AUDIENCE
	Auth0ClientID           string         // From AUTH0_CLIENT_ID
	Auth0ClientSecret       string         // From AUTH0_CLIENT_SECRET
	ShUser                  string         // From SH_USER: Sorting Hat database parameters
	ShHost                  string         // From SH_HOST
	ShPort                  string         // From SH_PORT
	ShPass                  string         // From SH_PASS
	ShDB                    string         // From SH_DB
	TestMode                bool           // True when running tests
	OAuthKeys               []string       // GitHub oauth keys recevide from SDS_GITHUB_OAUTH configuration (initialized only when lib.GHClient() is called)
}

// Init - get context from environment variables
func (ctx *Ctx) Init() {
	ctx.ExecFatal = true
	ctx.ExecQuiet = false
	ctx.ExecOutput = false
	ctx.ExecOutputStderr = false

	// ElasticSearch
	ctx.ElasticURL = os.Getenv("SDS_ES_URL")
	if ctx.ElasticURL == "" {
		ctx.ElasticURL = "http://127.0.0.1:9200"
	}
	AddRedacted(ctx.ElasticURL, false)

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
		ctx.MaxRetry = 0
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
	if os.Getenv("SDS_NCPUS_SCALE") == "" {
		ctx.NCPUsScale = 1.0
	} else {
		nCPUsScale, err := strconv.ParseFloat(os.Getenv("SDS_NCPUS_SCALE"), 64)
		FatalNoLog(err)
		if nCPUsScale > 0 {
			ctx.NCPUsScale = nCPUsScale
		}
	}

	// Only/Skip REs
	fixturesREStr := os.Getenv("SDS_FIXTURES_RE")
	datasourcesREStr := os.Getenv("SDS_DATASOURCES_RE")
	projectsREStr := os.Getenv("SDS_PROJECTS_RE")
	endpointsREStr := os.Getenv("SDS_ENDPOINTS_RE")
	tasksREStr := os.Getenv("SDS_TASKS_RE")
	if fixturesREStr != "" {
		ctx.FixturesRE = regexp.MustCompile(fixturesREStr)
	}
	if datasourcesREStr != "" {
		ctx.DatasourcesRE = regexp.MustCompile(datasourcesREStr)
	}
	if projectsREStr != "" {
		ctx.ProjectsRE = regexp.MustCompile(projectsREStr)
	}
	if endpointsREStr != "" {
		ctx.EndpointsRE = regexp.MustCompile(endpointsREStr)
	}
	if tasksREStr != "" {
		ctx.TasksRE = regexp.MustCompile(tasksREStr)
	}
	fixturesSkipREStr := os.Getenv("SDS_FIXTURES_SKIP_RE")
	datasourcesSkipREStr := os.Getenv("SDS_DATASOURCES_SKIP_RE")
	projectsSkipREStr := os.Getenv("SDS_PROJECTS_SKIP_RE")
	endpointsSkipREStr := os.Getenv("SDS_ENDPOINTS_SKIP_RE")
	tasksSkipREStr := os.Getenv("SDS_TASKS_SKIP_RE")
	if fixturesSkipREStr != "" {
		ctx.FixturesSkipRE = regexp.MustCompile(fixturesSkipREStr)
	}
	if datasourcesSkipREStr != "" {
		ctx.DatasourcesSkipRE = regexp.MustCompile(datasourcesSkipREStr)
	}
	if projectsSkipREStr != "" {
		ctx.ProjectsSkipRE = regexp.MustCompile(projectsSkipREStr)
	}
	if endpointsSkipREStr != "" {
		ctx.EndpointsSkipRE = regexp.MustCompile(endpointsSkipREStr)
	}
	if tasksSkipREStr != "" {
		ctx.TasksSkipRE = regexp.MustCompile(tasksSkipREStr)
	}

	// Dry Run mode
	ctx.DryRun = os.Getenv("SDS_DRY_RUN") != ""
	ctx.DryRunAllowSSH = os.Getenv("SDS_DRY_RUN_ALLOW_SSH") != ""
	ctx.DryRunAllowFreq = os.Getenv("SDS_DRY_RUN_ALLOW_FREQ") != ""
	ctx.DryRunAllowMtx = os.Getenv("SDS_DRY_RUN_ALLOW_MTX") != ""
	ctx.DryRunAllowRename = os.Getenv("SDS_DRY_RUN_ALLOW_RENAME") != ""
	ctx.DryRunAllowOrigins = os.Getenv("SDS_DRY_RUN_ALLOW_ORIGINS") != ""
	ctx.DryRunAllowDedup = os.Getenv("SDS_DRY_RUN_ALLOW_DEDUP") != ""
	ctx.DryRunAllowProject = os.Getenv("SDS_DRY_RUN_ALLOW_PROJECT") != ""
	ctx.DryRunCodeRandom = os.Getenv("SDS_DRY_RUN_CODE_RANDOM") != ""
	ctx.DryRunSecondsRandom = os.Getenv("SDS_DRY_RUN_SECONDS_RANDOM") != ""
	ctx.DryRunAllowSyncInfo = os.Getenv("SDS_DRY_RUN_ALLOW_SYNC_INFO") != ""
	ctx.DryRunAllowSortDuration = os.Getenv("SDS_DRY_RUN_ALLOW_SORT_DURATION") != ""
	ctx.DryRunAllowMerge = os.Getenv("SDS_DRY_RUN_ALLOW_MERGE") != ""
	ctx.DryRunAllowHideEmails = os.Getenv("SDS_DRY_RUN_ALLOW_HIDE_EMAILS") != ""
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
	ctx.ShPort = os.Getenv("SH_PORT")
	ctx.ShPass = os.Getenv("SH_PASS")
	ctx.ShDB = os.Getenv("SH_DB")
	AddRedacted(ctx.ShUser, false)
	AddRedacted(ctx.ShHost, false)
	AddRedacted(ctx.ShPort, false)
	AddRedacted(ctx.ShPass, false)
	AddRedacted(ctx.ShDB, false)

	// Auth0 parameters for obtaining DA-affiliation API token
	ctx.Auth0URL = os.Getenv("AUTH0_URL")
	ctx.Auth0Audience = os.Getenv("AUTH0_AUDIENCE")
	ctx.Auth0ClientID = os.Getenv("AUTH0_CLIENT_ID")
	ctx.Auth0ClientSecret = os.Getenv("AUTH0_CLIENT_SECRET")
	AddRedacted(ctx.Auth0URL, false)
	AddRedacted(ctx.Auth0Audience, false)
	AddRedacted(ctx.Auth0ClientID, false)
	AddRedacted(ctx.Auth0ClientSecret, false)

	// DA affiliation API URL
	ctx.AffiliationAPIURL = os.Getenv("AFFILIATION_API_URL")
	AddRedacted(ctx.AffiliationAPIURL, false)

	// Only validate support
	ctx.OnlyValidate = os.Getenv("SDS_ONLY_VALIDATE") != ""

	if !ctx.OnlyValidate && !ctx.TestMode && !ctx.DryRun && !ctx.SkipSH && (ctx.ShUser == "" || ctx.ShHost == "" || ctx.ShPort == "" || ctx.ShPass == "" || ctx.ShDB == "") {
		fmt.Printf("%v %v %s %s %s %s\n", ctx.TestMode, ctx.SkipSH, ctx.ShUser, ctx.ShHost, ctx.ShPass, ctx.ShDB)
		FatalNoLog(fmt.Errorf("SortingHat parameters (user, host, port, password, db) must all be defined unless skiping SortingHat"))
	}

	// Only P2O support
	ctx.OnlyP2O = os.Getenv("SDS_ONLY_P2O") != ""

	// Log Time
	ctx.LogTime = os.Getenv("SDS_SKIPTIME") == ""

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
	if os.Getenv("SDS_NODE_SETTLE_TIME") == "" {
		ctx.NodeSettleTime = 10
	} else {
		nst, err := strconv.Atoi(os.Getenv("SDS_NODE_SETTLE_TIME"))
		FatalNoLog(err)
		if nst > 0 {
			ctx.NodeSettleTime = nst
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

	// Single task timeout
	if os.Getenv("SDS_TASK_TIMEOUT_SECONDS") == "" {
		ctx.TaskTimeoutSeconds = 36000
	} else {
		secs, err := strconv.Atoi(os.Getenv("SDS_TASK_TIMEOUT_SECONDS"))
		FatalNoLog(err)
		if secs > 0 {
			ctx.TaskTimeoutSeconds = secs
		} else {
			ctx.TaskTimeoutSeconds = 36000
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
		ctx.StripErrorSize = 16384
	} else {
		n, err := strconv.Atoi(os.Getenv("SDS_STRIP_ERROR_SIZE"))
		FatalNoLog(err)
		if n > 1 {
			ctx.StripErrorSize = n
		} else {
			ctx.StripErrorSize = 16384
		}
	}

	// GitHub OAuth
	ctx.GitHubOAuth = os.Getenv("SDS_GITHUB_OAUTH")
	AddRedacted(ctx.GitHubOAuth, false)

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
	// ES scroll size p2o.py --scroll-size 100
	if os.Getenv("SDS_SCROLL_SIZE") == "" {
		ctx.ScrollSize = 100
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
	if !ctx.DryRun && ctx.SkipSH && !ctx.SkipAffs {
		FatalNoLog(fmt.Errorf("you cannot skip SortingHat and not skip affiliations at the same time"))
	}
	if !ctx.DryRun && ctx.SkipData && ctx.SkipAffs && ctx.SkipAliases {
		FatalNoLog(fmt.Errorf("you cannot skip incremental data sync, historical affiliations sync and aliases at the same time"))
	}

	// Skip sdsdata index processing
	ctx.SkipEsData = os.Getenv("SDS_SKIP_ES_DATA") != ""

	// Skip ES logs
	ctx.SkipEsLog = os.Getenv("SDS_SKIP_ES_LOG") != ""

	// Skip check sync frequency
	ctx.SkipCheckFreq = os.Getenv("SDS_SKIP_CHECK_FREQ") != ""

	// Skip dedup
	ctx.SkipDedup = os.Getenv("SDS_SKIP_DEDUP") != ""

	// Skip external
	ctx.SkipExternal = os.Getenv("SDS_SKIP_EXTERNAL") != ""

	// Skip project/TS settings
	ctx.SkipProject = os.Getenv("SDS_SKIP_PROJECT") != ""
	ctx.SkipProjectTS = os.Getenv("SDS_SKIP_PROJECT_TS") != ""

	// Skip sync info
	ctx.SkipSyncInfo = os.Getenv("SDS_SKIP_SYNC_INFO") != ""

	// Skip processing GitHub org's/user's repos in validate mode
	ctx.SkipValGitHubAPI = os.Getenv("SDS_SKIP_VALIDATE_GITHUB_API") != ""

	// Skip sort by running duration
	ctx.SkipSortDuration = os.Getenv("SDS_SKIP_SORT_DURATION") != ""

	// Skip calling DA-affiliation merge_all API at the end
	ctx.SkipMerge = os.Getenv("SDS_SKIP_MERGE") != ""

	// Skip calling DA-affiliation hide_emails API
	ctx.SkipHideEmails = os.Getenv("SDS_SKIP_HIDE_EMAILS") != ""

	// Skip all p2o commands
	ctx.SkipP2O = os.Getenv("SDS_SKIP_P2O") != ""

	// Max delete by query attempts - this can fail due to version conflicts
	if os.Getenv("SDS_MAX_DELETE_TRIALS") == "" {
		ctx.MaxDeleteTrials = 10
	} else {
		maxDeleteTrials, err := strconv.Atoi(os.Getenv("SDS_MAX_DELETE_TRIALS"))
		FatalNoLog(err)
		if maxDeleteTrials > 0 {
			ctx.MaxDeleteTrials = maxDeleteTrials
		} else {
			ctx.MaxDeleteTrials = 10
		}
	}

	// Max wait for ES giant mutex in seconds, if you set to 0, it means infinity
	if os.Getenv("SDS_MAX_MTX_WAIT") == "" {
		ctx.MaxMtxWait = 900
	} else {
		maxMtxWait, err := strconv.Atoi(os.Getenv("SDS_MAX_MTX_WAIT"))
		FatalNoLog(err)
		if maxMtxWait >= 0 {
			ctx.MaxMtxWait = maxMtxWait
		} else {
			ctx.MaxMtxWait = 900
		}
	}
	ctx.MaxMtxWaitFatal = os.Getenv("SDS_MAX_MTX_WAIT_FATAL") != ""

	if os.Getenv("SDS_ENRICH_EXTERNAL_FREQ") == "" {
		ctx.EnrichExternalFreq = time.Duration(48) * time.Hour
	} else {
		dur, err := time.ParseDuration(os.Getenv("SDS_ENRICH_EXTERNAL_FREQ"))
		FatalNoLog(err)
		ctx.EnrichExternalFreq = dur
	}

	// SSAW stuff
	ctx.SkipSSAW = os.Getenv("SDS_SKIP_SSAW") != ""
	ctx.SSAWURL = os.Getenv("SDS_SSAW_URL")
	if !ctx.TestMode && !ctx.OnlyValidate && !ctx.SkipSSAW && ctx.SSAWURL == "" {
		FatalNoLog(fmt.Errorf("you must provide SDS_SSAW_URL=... or use SDS_SKIP_SSAW=1"))
	}
	ctx.DryRunAllowSSAW = os.Getenv("SDS_DRY_RUN_ALLOW_SSAW") != ""
	if os.Getenv("SDS_SSAW_FREQ") == "" {
		ctx.SSAWFreq = 0
	} else {
		secs, err := strconv.Atoi(os.Getenv("SDS_SSAW_FREQ"))
		FatalNoLog(err)
		if secs > 0 {
			ctx.SSAWFreq = secs
			if ctx.SSAWFreq < 20 {
				ctx.SSAWFreq = 20
			}
		}
	}

	// Only validate support - overrides
	if ctx.OnlyValidate {
		ctx.SkipEsLog = true
		ctx.SkipEsData = true
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
