package syncdatasources

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	lib "github.com/LF-Engineering/sync-data-sources/sources"
)

// Copies Ctx structure
func copyContext(in *lib.Ctx) *lib.Ctx {
	out := lib.Ctx{
		Debug:                           in.Debug,
		CmdDebug:                        in.CmdDebug,
		MaxRetry:                        in.MaxRetry,
		ST:                              in.ST,
		NCPUs:                           in.NCPUs,
		NCPUsScale:                      in.NCPUsScale,
		FixturesRE:                      in.FixturesRE,
		DatasourcesRE:                   in.DatasourcesRE,
		ProjectsRE:                      in.ProjectsRE,
		EndpointsRE:                     in.EndpointsRE,
		TasksRE:                         in.TasksRE,
		FixturesSkipRE:                  in.FixturesSkipRE,
		DatasourcesSkipRE:               in.DatasourcesSkipRE,
		ProjectsSkipRE:                  in.ProjectsSkipRE,
		EndpointsSkipRE:                 in.EndpointsSkipRE,
		TasksSkipRE:                     in.TasksSkipRE,
		TasksExtraSkipRE:                in.TasksExtraSkipRE,
		CtxOut:                          in.CtxOut,
		DryRun:                          in.DryRun,
		DryRunCode:                      in.DryRunCode,
		DryRunCodeRandom:                in.DryRunCodeRandom,
		DryRunSeconds:                   in.DryRunSeconds,
		DryRunSecondsRandom:             in.DryRunSecondsRandom,
		DryRunAllowSSH:                  in.DryRunAllowSSH,
		DryRunAllowFreq:                 in.DryRunAllowFreq,
		DryRunAllowMtx:                  in.DryRunAllowMtx,
		DryRunAllowRename:               in.DryRunAllowRename,
		DryRunAllowOrigins:              in.DryRunAllowOrigins,
		DryRunAllowDedup:                in.DryRunAllowDedup,
		DryRunAllowFAliases:             in.DryRunAllowFAliases,
		DryRunAllowProject:              in.DryRunAllowProject,
		DryRunAllowSyncInfo:             in.DryRunAllowSyncInfo,
		DryRunAllowSortDuration:         in.DryRunAllowSortDuration,
		DryRunAllowMerge:                in.DryRunAllowMerge,
		DryRunAllowHideEmails:           in.DryRunAllowHideEmails,
		DryRunAllowCacheTopContributors: in.DryRunAllowCacheTopContributors,
		DryRunAllowOrgMap:               in.DryRunAllowOrgMap,
		DryRunAllowEnrichDS:             in.DryRunAllowEnrichDS,
		DryRunAllowDetAffRange:          in.DryRunAllowDetAffRange,
		DryRunAllowCopyFrom:             in.DryRunAllowCopyFrom,
		DryRunAllowMetadata:             in.DryRunAllowMetadata,
		OnlyValidate:                    in.OnlyValidate,
		OnlyP2O:                         in.OnlyP2O,
		TimeoutSeconds:                  in.TimeoutSeconds,
		TaskTimeoutSeconds:              in.TaskTimeoutSeconds,
		NodeIdx:                         in.NodeIdx,
		NodeNum:                         in.NodeNum,
		NodeHash:                        in.NodeHash,
		NodeSettleTime:                  in.NodeSettleTime,
		NLongest:                        in.NLongest,
		StripErrorSize:                  in.StripErrorSize,
		LogTime:                         in.LogTime,
		ExecFatal:                       in.ExecFatal,
		ExecQuiet:                       in.ExecQuiet,
		ExecOutput:                      in.ExecOutput,
		ElasticURL:                      in.ElasticURL,
		EsBulkSize:                      in.EsBulkSize,
		SkipSH:                          in.SkipSH,
		GitHubOAuth:                     in.GitHubOAuth,
		DynamicOAuth:                    in.DynamicOAuth,
		SkipReenrich:                    in.SkipReenrich,
		LatestItems:                     in.LatestItems,
		ScrollWait:                      in.ScrollWait,
		ScrollSize:                      in.ScrollSize,
		Silent:                          in.Silent,
		SkipData:                        in.SkipData,
		SkipAffs:                        in.SkipAffs,
		SkipAliases:                     in.SkipAliases,
		SkipDropUnused:                  in.SkipDropUnused,
		NoIndexDrop:                     in.NoIndexDrop,
		NoMultiAliases:                  in.NoMultiAliases,
		CleanupAliases:                  in.CleanupAliases,
		CSVPrefix:                       in.CSVPrefix,
		SkipCheckFreq:                   in.SkipCheckFreq,
		SkipEsData:                      in.SkipEsData,
		SkipEsLog:                       in.SkipEsLog,
		SkipDedup:                       in.SkipDedup,
		SkipFAliases:                    in.SkipFAliases,
		SkipExternal:                    in.SkipExternal,
		SkipProject:                     in.SkipProject,
		SkipProjectTS:                   in.SkipProjectTS,
		SkipSyncInfo:                    in.SkipSyncInfo,
		SkipValGitHubAPI:                in.SkipValGitHubAPI,
		SkipSortDuration:                in.SkipSortDuration,
		SkipMerge:                       in.SkipMerge,
		SkipHideEmails:                  in.SkipHideEmails,
		SkipCacheTopContributors:        in.SkipCacheTopContributors,
		SkipOrgMap:                      in.SkipOrgMap,
		SkipEnrichDS:                    in.SkipEnrichDS,
		SkipCopyFrom:                    in.SkipCopyFrom,
		SkipMetadata:                    in.SkipMetadata,
		RunDetAffRange:                  in.RunDetAffRange,
		SkipP2O:                         in.SkipP2O,
		MaxDeleteTrials:                 in.MaxDeleteTrials,
		MaxMtxWait:                      in.MaxMtxWait,
		MaxMtxWaitFatal:                 in.MaxMtxWaitFatal,
		EnrichExternalFreq:              in.EnrichExternalFreq,
		TestMode:                        in.TestMode,
		ShUser:                          in.ShUser,
		ShHost:                          in.ShHost,
		ShPort:                          in.ShPort,
		ShPass:                          in.ShPass,
		ShDB:                            in.ShDB,
		Auth0URL:                        in.Auth0URL,
		Auth0Audience:                   in.Auth0Audience,
		Auth0ClientID:                   in.Auth0ClientID,
		Auth0ClientSecret:               in.Auth0ClientSecret,
		AffiliationAPIURL:               in.AffiliationAPIURL,
		Auth0Data:                       in.Auth0Data,
		MetricsAPIURL:                   in.MetricsAPIURL,
	}
	return &out
}

// Dynamically sets Ctx fields (uses map of field names into their new values)
func dynamicSetFields(t *testing.T, ctx *lib.Ctx, fields map[string]interface{}) *lib.Ctx {
	// Prepare mapping field name -> index
	valueOf := reflect.Indirect(reflect.ValueOf(*ctx))
	nFields := valueOf.Type().NumField()
	namesToIndex := make(map[string]int)
	for i := 0; i < nFields; i++ {
		namesToIndex[valueOf.Type().Field(i).Name] = i
	}

	// Iterate map of interface{} and set values
	elem := reflect.ValueOf(ctx).Elem()
	for fieldName, fieldValue := range fields {
		// Check if structure actually  contains this field
		fieldIndex, ok := namesToIndex[fieldName]
		if !ok {
			t.Errorf("context has no field: \"%s\"", fieldName)
			return ctx
		}
		field := elem.Field(fieldIndex)
		fieldKind := field.Kind()
		// Switch type that comes from interface
		switch interfaceValue := fieldValue.(type) {
		case int:
			// Check if types match
			if fieldKind != reflect.Int {
				t.Errorf("trying to set value %v, type %T for field \"%s\", type %v", interfaceValue, interfaceValue, fieldName, fieldKind)
				return ctx
			}
			field.SetInt(int64(interfaceValue))
		case float64:
			// Check if types match
			if fieldKind != reflect.Float64 {
				t.Errorf("trying to set value %v, type %T for field \"%s\", type %v", interfaceValue, interfaceValue, fieldName, fieldKind)
				return ctx
			}
			field.SetFloat(float64(interfaceValue))
		case bool:
			// Check if types match
			if fieldKind != reflect.Bool {
				t.Errorf("trying to set value %v, type %T for field \"%s\", type %v", interfaceValue, interfaceValue, fieldName, fieldKind)
				return ctx
			}
			field.SetBool(interfaceValue)
		case string:
			// Check if types match
			if fieldKind != reflect.String {
				t.Errorf("trying to set value %v, type %T for field \"%s\", type %v", interfaceValue, interfaceValue, fieldName, fieldKind)
				return ctx
			}
			field.SetString(interfaceValue)
		case time.Time:
			// Check if types match
			fieldType := field.Type()
			if fieldType != reflect.TypeOf(time.Now()) {
				t.Errorf("trying to set value %v, type %T for field \"%s\", type %v", interfaceValue, interfaceValue, fieldName, fieldKind)
				return ctx
			}
			field.Set(reflect.ValueOf(fieldValue))
		case time.Duration:
			// Check if types match
			fieldType := field.Type()
			if fieldType != reflect.TypeOf(time.Now().Sub(time.Now())) {
				t.Errorf("trying to set value %v, type %T for field \"%s\", type %v", interfaceValue, interfaceValue, fieldName, fieldKind)
				return ctx
			}
			field.Set(reflect.ValueOf(fieldValue))
		case []int:
			// Check if types match
			fieldType := field.Type()
			if fieldType != reflect.TypeOf([]int{}) {
				t.Errorf("trying to set value %v, type %T for field \"%s\", type %v", interfaceValue, interfaceValue, fieldName, fieldKind)
				return ctx
			}
			field.Set(reflect.ValueOf(fieldValue))
		case []int64:
			// Check if types match
			fieldType := field.Type()
			if fieldType != reflect.TypeOf([]int64{}) {
				t.Errorf("trying to set value %v, type %T for field \"%s\", type %v", interfaceValue, interfaceValue, fieldName, fieldKind)
				return ctx
			}
			field.Set(reflect.ValueOf(fieldValue))
		case []string:
			// Check if types match
			fieldType := field.Type()
			if fieldType != reflect.TypeOf([]string{}) {
				t.Errorf("trying to set value %v, type %T for field \"%s\", type %v", interfaceValue, interfaceValue, fieldName, fieldKind)
				return ctx
			}
			field.Set(reflect.ValueOf(fieldValue))
		case map[string]bool:
			// Check if types match
			fieldType := field.Type()
			if fieldType != reflect.TypeOf(map[string]bool{}) {
				t.Errorf("trying to set value %v, type %T for field \"%s\", type %v", interfaceValue, interfaceValue, fieldName, fieldKind)
				return ctx
			}
			field.Set(reflect.ValueOf(fieldValue))
		case map[string]map[bool]struct{}:
			// Check if types match
			fieldType := field.Type()
			if fieldType != reflect.TypeOf(map[string]map[bool]struct{}{}) {
				t.Errorf("trying to set value %v, type %T for field \"%s\", type %v", interfaceValue, interfaceValue, fieldName, fieldKind)
				return ctx
			}
			field.Set(reflect.ValueOf(fieldValue))
		case *regexp.Regexp:
			// Check if types match
			fieldType := field.Type()
			if fieldType != reflect.TypeOf(regexp.MustCompile("a")) {
				t.Errorf("trying to set value %v, type %T for field \"%s\", type %v", interfaceValue, interfaceValue, fieldName, fieldKind)
				return ctx
			}
			field.Set(reflect.ValueOf(fieldValue))
		default:
			// Unknown type provided
			t.Errorf("unknown type %T for field \"%s\"", interfaceValue, fieldName)
		}
	}

	// Return dynamically updated structure
	return ctx
}

func TestInit(t *testing.T) {
	// This is the expected default struct state
	defaultContext := lib.Ctx{
		Debug:                           0,
		CmdDebug:                        0,
		MaxRetry:                        0,
		ST:                              false,
		NCPUs:                           0,
		NCPUsScale:                      1.0,
		FixturesRE:                      nil,
		DatasourcesRE:                   nil,
		ProjectsRE:                      nil,
		EndpointsRE:                     nil,
		TasksRE:                         nil,
		FixturesSkipRE:                  nil,
		DatasourcesSkipRE:               nil,
		ProjectsSkipRE:                  nil,
		EndpointsSkipRE:                 nil,
		TasksSkipRE:                     nil,
		TasksExtraSkipRE:                nil,
		CtxOut:                          false,
		LatestItems:                     false,
		DryRun:                          false,
		DryRunCode:                      0,
		DryRunCodeRandom:                false,
		DryRunSeconds:                   0,
		DryRunSecondsRandom:             false,
		DryRunAllowSSH:                  false,
		DryRunAllowFreq:                 false,
		DryRunAllowMtx:                  false,
		DryRunAllowRename:               false,
		DryRunAllowOrigins:              false,
		DryRunAllowDedup:                false,
		DryRunAllowFAliases:             false,
		DryRunAllowProject:              false,
		DryRunAllowSyncInfo:             false,
		DryRunAllowSortDuration:         false,
		DryRunAllowMerge:                false,
		DryRunAllowHideEmails:           false,
		DryRunAllowCacheTopContributors: false,
		DryRunAllowOrgMap:               false,
		DryRunAllowEnrichDS:             false,
		DryRunAllowDetAffRange:          false,
		DryRunAllowCopyFrom:             false,
		DryRunAllowMetadata:             false,
		OnlyValidate:                    false,
		OnlyP2O:                         false,
		TimeoutSeconds:                  258660,
		TaskTimeoutSeconds:              86400,
		NodeIdx:                         0,
		NodeNum:                         1,
		NodeHash:                        false,
		NodeSettleTime:                  10,
		NLongest:                        30,
		StripErrorSize:                  16384,
		LogTime:                         true,
		ExecFatal:                       true,
		ExecQuiet:                       false,
		ExecOutput:                      false,
		ElasticURL:                      "http://127.0.0.1:9200",
		GitHubOAuth:                     "",
		DynamicOAuth:                    false,
		SkipReenrich:                    "",
		EsBulkSize:                      0,
		ScrollWait:                      2700,
		ScrollSize:                      500,
		Silent:                          false,
		CSVPrefix:                       "/root/.perceval/jobs",
		SkipSH:                          false,
		SkipData:                        false,
		SkipAffs:                        false,
		SkipAliases:                     false,
		NoMultiAliases:                  false,
		CleanupAliases:                  false,
		SkipDropUnused:                  false,
		NoIndexDrop:                     false,
		SkipCheckFreq:                   false,
		SkipEsData:                      false,
		SkipEsLog:                       false,
		SkipDedup:                       false,
		SkipFAliases:                    false,
		SkipExternal:                    false,
		SkipProject:                     false,
		SkipProjectTS:                   false,
		SkipSyncInfo:                    false,
		SkipValGitHubAPI:                false,
		SkipSortDuration:                false,
		SkipMerge:                       false,
		SkipHideEmails:                  false,
		SkipCacheTopContributors:        false,
		SkipOrgMap:                      false,
		SkipEnrichDS:                    false,
		SkipCopyFrom:                    false,
		SkipMetadata:                    false,
		RunDetAffRange:                  false,
		SkipP2O:                         false,
		MaxDeleteTrials:                 10,
		MaxMtxWait:                      900,
		MaxMtxWaitFatal:                 false,
		EnrichExternalFreq:              time.Duration(168) * time.Hour,
		TestMode:                        true,
		ShUser:                          "",
		ShHost:                          "",
		ShPort:                          "",
		ShPass:                          "",
		ShDB:                            "",
		Auth0URL:                        "",
		Auth0Audience:                   "",
		Auth0ClientID:                   "",
		Auth0ClientSecret:               "",
		AffiliationAPIURL:               "",
		Auth0Data:                       "",
		MetricsAPIURL:                   "",
	}

	// Test cases
	var testCases = []struct {
		name            string
		environment     map[string]string
		expectedContext *lib.Ctx
	}{
		{
			"Default values",
			map[string]string{},
			&defaultContext,
		},
		{
			"Setting debug level",
			map[string]string{"SDS_DEBUG": "2"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"Debug": 2},
			),
		},
		{
			"Setting negative debug level",
			map[string]string{"SDS_DEBUG": "-1"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"Debug": -1},
			),
		},
		{
			"Setting command debug level",
			map[string]string{"SDS_CMDDEBUG": "3"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"CmdDebug": 3},
			),
		},
		{
			"Setting max delete trials to 5",
			map[string]string{"SDS_MAX_DELETE_TRIALS": "5"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"MaxDeleteTrials": 5},
			),
		},
		{
			"Setting max delete trials to 0 (not allowed)",
			map[string]string{"SDS_MAX_DELETE_TRIALS": "0"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"MaxDeleteTrials": 10},
			),
		},
		{
			"Setting max ES mutex wait (30 seconds) and set it as fatal",
			map[string]string{
				"SDS_MAX_MTX_WAIT":       "30",
				"SDS_MAX_MTX_WAIT_FATAL": "yeah",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"MaxMtxWait":      30,
					"MaxMtxWaitFatal": true,
				},
			),
		},
		{
			"Setting maximum enrich external indices frequency",
			map[string]string{"SDS_ENRICH_EXTERNAL_FREQ": "12h"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"EnrichExternalFreq": time.Duration(12) * time.Hour},
			),
		},
		{
			"Setting max ES mutex wait (0 seconds)",
			map[string]string{"SDS_MAX_MTX_WAIT": "0"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"MaxMtxWait": 0},
			),
		},
		{
			"Setting max ES mutex wait (-1 seconds)",
			map[string]string{"SDS_MAX_MTX_WAIT": "-1"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"MaxMtxWait": 900},
			),
		},
		{
			"Setting elastic search bulk size",
			map[string]string{"SDS_ES_BULKSIZE": "10000"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"EsBulkSize": 10000},
			),
		},
		{
			"Setting scroll size and wait",
			map[string]string{
				"SDS_SCROLL_WAIT": "30",
				"SDS_SCROLL_SIZE": "500",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"ScrollWait": 30,
					"ScrollSize": 500,
				},
			),
		},
		{
			"Setting scroll size and wait to 0s",
			map[string]string{
				"SDS_SCROLL_WAIT": "0",
				"SDS_SCROLL_SIZE": "0",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"ScrollWait": 0,
					"ScrollSize": 0,
				},
			),
		},
		{
			"Setting max retry parameter",
			map[string]string{"SDS_MAXRETRY": "5"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"MaxRetry": 5},
			),
		},
		{
			"Setting filtering parameters (match)",
			map[string]string{
				"SDS_FIXTURES_RE":    `^lfn/.*`,
				"SDS_DATASOURCES_RE": `^git$`,
				"SDS_PROJECTS_RE":    `(?i)network`,
				"SDS_ENDPOINTS_RE":   `\.gerrit\.`,
				"SDS_TASKS_RE":       `-bugzilla-`,
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"FixturesRE":    regexp.MustCompile(`^lfn/.*`),
					"DatasourcesRE": regexp.MustCompile(`^git$`),
					"ProjectsRE":    regexp.MustCompile(`(?i)network`),
					"EndpointsRE":   regexp.MustCompile(`\.gerrit\.`),
					"TasksRE":       regexp.MustCompile(`-bugzilla-`),
				},
			),
		},
		{
			"Setting filtering parameters (not match)",
			map[string]string{
				"SDS_FIXTURES_SKIP_RE":    `^lfn/.*`,
				"SDS_DATASOURCES_SKIP_RE": `^git$`,
				"SDS_PROJECTS_SKIP_RE":    `(?i)network`,
				"SDS_ENDPOINTS_SKIP_RE":   `\.gerrit\.`,
				"SDS_TASKS_SKIP_RE":       `-bugzilla-`,
				"SDS_TASKS_EXTRA_SKIP_RE": `-bugzillarest-`,
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"FixturesSkipRE":    regexp.MustCompile(`^lfn/.*`),
					"DatasourcesSkipRE": regexp.MustCompile(`^git$`),
					"ProjectsSkipRE":    regexp.MustCompile(`(?i)network`),
					"EndpointsSkipRE":   regexp.MustCompile(`\.gerrit\.`),
					"TasksSkipRE":       regexp.MustCompile(`-bugzilla-`),
					"TasksExtraSkipRE":  regexp.MustCompile(`-bugzillarest-`),
				},
			),
		},
		{
			"Setting ST (singlethreading) and NCPUs",
			map[string]string{"SDS_ST": "1", "SDS_NCPUS": "1"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"ST": true, "NCPUs": 1},
			),
		},
		{
			"Setting NCPUs to 2",
			map[string]string{"SDS_NCPUS": "2"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"ST": false, "NCPUs": 2},
			),
		},
		{
			"Setting NCPUs to 1 should also set ST mode",
			map[string]string{"SDS_NCPUS": "1"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"ST": true, "NCPUs": 1},
			),
		},
		{
			"Setting NCPUs Scale to 1.5",
			map[string]string{"SDS_NCPUS_SCALE": "1.5"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"ST": false, "NCPUsScale": 1.5},
			),
		},
		{
			"Setting skip log time",
			map[string]string{
				"SDS_SKIPTIME": "Y",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"LogTime": false,
				},
			),
		},
		{
			"Setting context out",
			map[string]string{"SDS_CTXOUT": "1"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"CtxOut": true},
			),
		},
		{
			"Set ES URL",
			map[string]string{"SDS_ES_URL": "http://other.server:9222"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"ElasticURL": "http://other.server:9222"},
			),
		},
		{
			"Set GitHubOAuth",
			map[string]string{"SDS_GITHUB_OAUTH": "key1,key2", "SDS_DYNAMIC_OAUTH": "1"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"GitHubOAuth": "key1,key2", "DynamicOAuth": true},
			),
		},
		{
			"Set Skip Re-enrich",
			map[string]string{"SDS_SKIP_REENRICH": "jira,gerrit,bugzilla,confluence"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"SkipReenrich": "jira,gerrit,bugzilla,confluence"},
			),
		},
		{
			"Set Timeout",
			map[string]string{"SDS_TIMEOUT_SECONDS": "7200"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"TimeoutSeconds": 7200},
			),
		},
		{
			"Set Task Timeout",
			map[string]string{"SDS_TASK_TIMEOUT_SECONDS": "7200"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"TaskTimeoutSeconds": 7200},
			),
		},
		{
			"Set number of longest running tasks stats",
			map[string]string{"SDS_N_LONGEST": "7"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"NLongest": 7},
			),
		},
		{
			"Set strip error size -1",
			map[string]string{"SDS_STRIP_ERROR_SIZE": "-1"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"StripErrorSize": 16384},
			),
		},
		{
			"Set strip error size 0",
			map[string]string{"SDS_STRIP_ERROR_SIZE": "0"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"StripErrorSize": 16384},
			),
		},
		{
			"Set strip error size 1",
			map[string]string{"SDS_STRIP_ERROR_SIZE": "1"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"StripErrorSize": 16384},
			),
		},
		{
			"Set strip error size",
			map[string]string{"SDS_STRIP_ERROR_SIZE": "2"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"StripErrorSize": 2},
			),
		},
		{
			"Set strip error size",
			map[string]string{"SDS_STRIP_ERROR_SIZE": "2048"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"StripErrorSize": 2048},
			),
		},
		{
			"Set dry run mode",
			map[string]string{
				"SDS_DRY_RUN":                              "1",
				"SDS_DRY_RUN_CODE":                         "4",
				"SDS_DRY_RUN_CODE_RANDOM":                  "yep",
				"SDS_DRY_RUN_SECONDS":                      "2",
				"SDS_DRY_RUN_SECONDS_RANDOM":               "yes",
				"SDS_DRY_RUN_ALLOW_SSH":                    "1",
				"SDS_DRY_RUN_ALLOW_FREQ":                   "y",
				"SDS_DRY_RUN_ALLOW_MTX":                    "t",
				"SDS_DRY_RUN_ALLOW_RENAME":                 "x",
				"SDS_DRY_RUN_ALLOW_ORIGINS":                "1",
				"SDS_DRY_RUN_ALLOW_DEDUP":                  "t",
				"SDS_DRY_RUN_ALLOW_F_ALIASES":              "t",
				"SDS_DRY_RUN_ALLOW_PROJECT":                "x",
				"SDS_DRY_RUN_ALLOW_SYNC_INFO":              "1",
				"SDS_DRY_RUN_ALLOW_SORT_DURATION":          "1",
				"SDS_DRY_RUN_ALLOW_MERGE":                  "1",
				"SDS_DRY_RUN_ALLOW_HIDE_EMAILS":            "1",
				"SDS_DRY_RUN_ALLOW_CACHE_TOP_CONTRIBUTORS": "1",
				"SDS_DRY_RUN_ALLOW_ORG_MAP":                "t",
				"SDS_DRY_RUN_ALLOW_ENRICH_DS":              "t",
				"SDS_DRY_RUN_ALLOW_DET_AFF_RANGE":          "t",
				"SDS_DRY_RUN_ALLOW_COPY_FROM":              "1",
				"SDS_DRY_RUN_ALLOW_METADATA":               "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"DryRun":                          true,
					"DryRunCode":                      4,
					"DryRunCodeRandom":                true,
					"DryRunSeconds":                   2,
					"DryRunSecondsRandom":             true,
					"DryRunAllowSSH":                  true,
					"DryRunAllowFreq":                 true,
					"DryRunAllowMtx":                  true,
					"DryRunAllowRename":               true,
					"DryRunAllowOrigins":              true,
					"DryRunAllowDedup":                true,
					"DryRunAllowFAliases":             true,
					"DryRunAllowProject":              true,
					"DryRunAllowSyncInfo":             true,
					"DryRunAllowSortDuration":         true,
					"DryRunAllowMerge":                true,
					"DryRunAllowHideEmails":           true,
					"DryRunAllowCacheTopContributors": true,
					"DryRunAllowOrgMap":               true,
					"DryRunAllowEnrichDS":             true,
					"DryRunAllowDetAffRange":          true,
					"DryRunAllowCopyFrom":             true,
					"DryRunAllowMetadata":             true,
				},
			),
		},
		{
			"Set validate only mode",
			map[string]string{
				"SDS_ONLY_VALIDATE": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"OnlyValidate": true,
					"SkipEsData":   true,
					"SkipEsLog":    true,
				},
			),
		},
		{
			"Set p2o only mode",
			map[string]string{
				"SDS_ONLY_P2O": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"OnlyP2O": true,
				},
			),
		},
		{
			"Set p2o skip mode",
			map[string]string{
				"SDS_SKIP_P2O": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipP2O": true,
				},
			),
		},
		{
			"Setting node hash params",
			map[string]string{
				"SDS_NODE_HASH":        "1",
				"SDS_NODE_IDX":         "2",
				"SDS_NODE_NUM":         "4",
				"SDS_NODE_SETTLE_TIME": "30",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"NodeHash":       true,
					"NodeIdx":        2,
					"NodeNum":        4,
					"NodeSettleTime": 30,
				},
			),
		},
		{
			"Setting node hash params (incorrect)",
			map[string]string{
				"SDS_NODE_HASH": "",
				"SDS_NODE_IDX":  "-1",
				"SDS_NODE_NUM":  "-1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"NodeHash": false,
					"NodeIdx":  0,
					"NodeNum":  1,
				},
			),
		},
		{
			"Setting node hash params (incorrect)",
			map[string]string{
				"SDS_NODE_HASH": "yes",
				"SDS_NODE_IDX":  "3",
				"SDS_NODE_NUM":  "3",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"NodeHash": true,
					"NodeIdx":  0,
					"NodeNum":  3,
				},
			),
		},
		{
			"Set skip sync frequency check",
			map[string]string{
				"SDS_SKIP_CHECK_FREQ": "y",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipCheckFreq": true,
				},
			),
		},
		{
			"Set skip sdsdata index processing (SDS state storage)",
			map[string]string{
				"SDS_SKIP_ES_DATA": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipEsData": true,
				},
			),
		},
		{
			"Set skip ES log storage",
			map[string]string{
				"SDS_SKIP_ES_LOG": "y",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipEsLog": true,
				},
			),
		},
		{
			"Set skip dedup",
			map[string]string{
				"SDS_SKIP_DEDUP": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipDedup": true,
				},
			),
		},
		{
			"Set skip external",
			map[string]string{
				"SDS_SKIP_EXTERNAL": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipExternal": true,
				},
			),
		},
		{
			"Set skip foundation-f aliases",
			map[string]string{
				"SDS_SKIP_F_ALIASES": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipFAliases": true,
				},
			),
		},
		{
			"Set skip project",
			map[string]string{
				"SDS_SKIP_PROJECT":    "x",
				"SDS_SKIP_PROJECT_TS": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipProject":   true,
					"SkipProjectTS": true,
				},
			),
		},
		{
			"Set skip sync info",
			map[string]string{
				"SDS_SKIP_SYNC_INFO": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipSyncInfo": true,
				},
			),
		},
		{
			"Set skip sort duration",
			map[string]string{
				"SDS_SKIP_SORT_DURATION": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipSortDuration": true,
				},
			),
		},
		{
			"Set skip merge",
			map[string]string{
				"SDS_SKIP_MERGE": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipMerge": true,
				},
			),
		},
		{
			"Set skip hide emails",
			map[string]string{
				"SDS_SKIP_HIDE_EMAILS": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipHideEmails": true,
				},
			),
		},
		{
			"Set skip cache top contributors",
			map[string]string{
				"SDS_SKIP_CACHE_TOP_CONTRIBUTORS": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipCacheTopContributors": true,
				},
			),
		},
		{
			"Set skip copy from",
			map[string]string{
				"SDS_SKIP_COPY_FROM": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipCopyFrom": true,
				},
			),
		},
		{
			"Set skip map org names",
			map[string]string{
				"SDS_SKIP_ORG_MAP": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipOrgMap": true,
				},
			),
		},
		{
			"Set skip fixture metadata",
			map[string]string{
				"SDS_SKIP_METADATA": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipMetadata": true,
				},
			),
		},
		{
			"Set skip enrich",
			map[string]string{
				"SDS_SKIP_ENRICH_DS": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipEnrichDS": true,
				},
			),
		},
		{
			"Set skip detect affiliations date range",
			map[string]string{
				"SDS_RUN_DET_AFF_RANGE": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"RunDetAffRange": true,
				},
			),
		},
		{
			"Set skip validate GitHub user's/org's repos in validate mode",
			map[string]string{
				"SDS_SKIP_VALIDATE_GITHUB_API": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipValGitHubAPI": true,
				},
			),
		},
		{
			"Set skip SortingHat/Data/Affs/Aliases mode",
			map[string]string{
				"SDS_SKIP_SH":          "1",
				"SDS_SKIP_AFFS":        "t",
				"SDS_SKIP_ALIASES":     "y",
				"SDS_NO_MULTI_ALIASES": "y",
				"SDS_CLEANUP_ALIASES":  "y",
				"SDS_SKIP_DROP_UNUSED": "y",
				"SDS_NO_INDEX_DROP":    "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipSH":         true,
					"SkipAffs":       true,
					"SkipAliases":    true,
					"NoMultiAliases": true,
					"CleanupAliases": true,
					"SkipDropUnused": true,
					"NoIndexDrop":    true,
				},
			),
		},
		{
			"Set skip Data mode",
			map[string]string{
				"SDS_SKIP_DATA": "y",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipData": true,
				},
			),
		},
		{
			"Setting backend latest items flag",
			map[string]string{"SDS_LATEST_ITEMS": "1"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"LatestItems": true},
			),
		},
		{
			"Setting silent mode",
			map[string]string{"SDS_SILENT": "y"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"Silent": true},
			),
		},
		{
			"Set CSV prefix",
			map[string]string{"SDS_CSV_PREFIX": "/debug_jobs"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"CSVPrefix": "/debug_jobs"},
			),
		},
		{
			"Set SH DB params",
			map[string]string{
				"SH_HOST": "my.host",
				"SH_PORT": "13306",
				"SH_USER": "user-name",
				"SH_PASS": "123!@#",
				"SH_DB":   "shdb",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"ShHost": "my.host",
					"ShPort": "13306",
					"ShUser": "user-name",
					"ShPass": "123!@#",
					"ShDB":   "shdb",
				},
			),
		},
		{
			"Set Auth0 params",
			map[string]string{
				"AUTH0_URL":           "https://auth0.com/",
				"AUTH0_AUDIENCE":      "my-api",
				"AUTH0_CLIENT_ID":     "123456",
				"AUTH0_CLIENT_SECRET": "abcdefghi",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"Auth0URL":          "https://auth0.com/",
					"Auth0Audience":     "my-api",
					"Auth0ClientID":     "123456",
					"Auth0ClientSecret": "abcdefghi",
				},
			),
		},
		{
			"Set DA affiliation/metrics API url",
			map[string]string{
				"AFFILIATION_API_URL": "https://affs-api.com",
				"METRICS_API_URL":     "https://metrics-api.com",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"AffiliationAPIURL": "https://affs-api.com",
					"MetricsAPIURL":     "https://metrics-api.com",
				},
			),
		},
		{
			"Set da-ds auth0 data JSON",
			map[string]string{
				"AUTH0_DATA": `{"a":"b", "c": "d", "x": 120, "y": null, z: [1, 2, 3], h: {"a":"b"}, "aa": ["a", "b", "c"]}`,
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"Auth0Data": `{"a":"b", "c": "d", "x": 120, "y": null, z: [1, 2, 3], h: {"a":"b"}, "aa": ["a", "b", "c"]}`,
				},
			),
		},
	}

	// Context Init() is verbose when called with CtxDebug
	// For this case we want to discard its STDOUT
	stdout := os.Stdout

	// Execute test cases
	for index, test := range testCases {
		var gotContext lib.Ctx

		// Remember initial environment
		currEnv := make(map[string]string)
		for key := range test.environment {
			currEnv[key] = os.Getenv(key)
		}

		// Set new environment
		for key, value := range test.environment {
			err := os.Setenv(key, value)
			if err != nil {
				t.Errorf(err.Error())
			}
		}

		// When CTXOUT is set, Ctx.Init() writes debug data to STDOUT
		// We don't want to see it while running tests
		if test.environment["SDS_CTXOUT"] != "" {
			fd, err := os.Open(os.DevNull)
			if err != nil {
				t.Errorf(err.Error())
			}
			os.Stdout = fd
		}

		// Initialize context while new environment is set
		gotContext.TestMode = true
		gotContext.Init()
		if test.environment["SDS_CTXOUT"] != "" {
			os.Stdout = stdout
		}

		// Restore original environment
		for key := range test.environment {
			err := os.Setenv(key, currEnv[key])
			if err != nil {
				t.Errorf(err.Error())
			}
		}

		// Check if we got expected context
		got := fmt.Sprintf("%+v", gotContext)
		expected := fmt.Sprintf("%+v", *test.expectedContext)
		if got != expected {
			t.Errorf(
				"Test case number %d \"%s\"\nExpected:\n%+v\nGot:\n%+v\n",
				index+1, test.name, expected, got,
			)
		}
	}
}
