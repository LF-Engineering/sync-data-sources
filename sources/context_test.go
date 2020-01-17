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
		Debug:          in.Debug,
		CmdDebug:       in.CmdDebug,
		MaxRetry:       in.MaxRetry,
		ST:             in.ST,
		NCPUs:          in.NCPUs,
		CtxOut:         in.CtxOut,
		NodeHash:       in.NodeHash,
		DryRun:         in.DryRun,
		DryRunCode:     in.DryRunCode,
		DryRunSeconds:  in.DryRunSeconds,
		DryRunAllowSSH: in.DryRunAllowSSH,
		TimeoutSeconds: in.TimeoutSeconds,
		NodeIdx:        in.NodeIdx,
		NodeNum:        in.NodeNum,
		NLongest:       in.NLongest,
		StripErrorSize: in.StripErrorSize,
		LogTime:        in.LogTime,
		ExecFatal:      in.ExecFatal,
		ExecQuiet:      in.ExecQuiet,
		ExecOutput:     in.ExecOutput,
		ElasticURL:     in.ElasticURL,
		EsBulkSize:     in.EsBulkSize,
		SkipSH:         in.SkipSH,
		GitHubOAuth:    in.GitHubOAuth,
		LatestItems:    in.LatestItems,
		ScrollWait:     in.ScrollWait,
		ScrollSize:     in.ScrollSize,
		Silent:         in.Silent,
		SkipData:       in.SkipData,
		SkipAffs:       in.SkipAffs,
		CSVPrefix:      in.CSVPrefix,
		TestMode:       in.TestMode,
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
		Debug:          0,
		CmdDebug:       0,
		MaxRetry:       2,
		ST:             false,
		NCPUs:          0,
		CtxOut:         false,
		NodeHash:       false,
		LatestItems:    false,
		DryRun:         false,
		DryRunCode:     0,
		DryRunSeconds:  0,
		DryRunAllowSSH: false,
		TimeoutSeconds: 171900,
		NodeIdx:        0,
		NodeNum:        1,
		NLongest:       10,
		StripErrorSize: 0x400,
		LogTime:        true,
		ExecFatal:      true,
		ExecQuiet:      false,
		ExecOutput:     false,
		ElasticURL:     "http://127.0.0.1:9200",
		GitHubOAuth:    "",
		EsBulkSize:     0,
		ScrollWait:     0,
		ScrollSize:     1000,
		Silent:         false,
		CSVPrefix:      "jobs",
		SkipSH:         false,
		SkipData:       false,
		SkipAffs:       false,
		TestMode:       true,
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
			map[string]string{"SDS_GITHUB_OAUTH": "key1,key2"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"GitHubOAuth": "key1,key2"},
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
				map[string]interface{}{"StripErrorSize": 1024},
			),
		},
		{
			"Set strip error size 0",
			map[string]string{"SDS_STRIP_ERROR_SIZE": "0"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"StripErrorSize": 1024},
			),
		},
		{
			"Set strip error size 1",
			map[string]string{"SDS_STRIP_ERROR_SIZE": "1"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"StripErrorSize": 1024},
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
				"SDS_DRY_RUN":           "1",
				"SDS_DRY_RUN_CODE":      "4",
				"SDS_DRY_RUN_SECONDS":   "2",
				"SDS_DRY_RUN_ALLOW_SSH": "1",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"DryRun":         true,
					"DryRunCode":     4,
					"DryRunSeconds":  2,
					"DryRunAllowSSH": true,
				},
			),
		},
		{
			"Setting node hash params",
			map[string]string{
				"SDS_NODE_HASH": "1",
				"SDS_NODE_IDX":  "2",
				"SDS_NODE_NUM":  "4",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"NodeHash": true,
					"NodeIdx":  2,
					"NodeNum":  4,
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
			"Set skip SortingHat/Data/Affs mode",
			map[string]string{
				"SDS_SKIP_SH":   "1",
				"SDS_SKIP_AFFS": "t",
			},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{
					"SkipSH":   true,
					"SkipAffs": true,
				},
			),
		},
		{
			"Set skip SortingHat/Data/Affs mode",
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
			map[string]string{"SDS_CSV_PREFIX": "debug_jobs"},
			dynamicSetFields(
				t,
				copyContext(&defaultContext),
				map[string]interface{}{"CSVPrefix": "debug_jobs"},
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
