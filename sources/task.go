package syncdatasources

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Task holds single endpoint task and its context (required config, fixture filename etc.)
type Task struct {
	Endpoint            string
	Config              []Config
	DsSlug              string
	FxSlug              string
	FxFn                string
	MaxFreq             time.Duration
	CommandLine         string
	RedactedCommandLine string
	Retries             int
	Err                 error
	Duration            time.Duration
	DsFullSlug          string
	ExternalIndex       string
	Project             string
	ProjectP2O          bool
	Projects            []EndpointProject
	Millis              int64
	Timeout             time.Duration
	CopyFrom            CopyConfig
	PairProgramming     bool
	AffiliationSource   string
	Dummy               bool
}

// String - default string output for a task (generic)
func (t Task) String() string {
	configStr := "["
	for _, cfg := range t.Config {
		configStr += cfg.Name + " "
	}
	configStr += "]"
	return fmt.Sprintf(
		"{Endpoint:%s Project:%s/%v DS:%s FDS:%s Slug:%s File:%s Configs:%s Cmd:%s Retries:%d Error:%v Duration: %v MaxFreq: %v ExternalIndex: %s AffSrc:%s}",
		t.Endpoint, t.Project, t.ProjectP2O, t.DsSlug, t.DsFullSlug, t.FxSlug, t.FxFn, configStr,
		t.RedactedCommandLine, t.Retries, t.Err != nil, t.Duration, t.MaxFreq, t.ExternalIndex, t.AffiliationSource,
	)
}

// ShortString - output quick endpoint info (usually used for non finished tasks)
func (t Task) ShortString() string {
	return fmt.Sprintf("%s:%s / %s:%s", t.FxSlug, t.DsFullSlug, t.Project, t.Endpoint)
}

// ShortStringCmd - output quick endpoint info (with command line)
func (t Task) ShortStringCmd(ctx *Ctx) string {
	s := fmt.Sprintf("%s:%s / %s:%s [%s]: ", t.FxSlug, t.DsFullSlug, t.Project, t.Endpoint, t.RedactedCommandLine)
	if t.Err == nil {
		s += "succeeded"
		if t.Retries > 0 {
			s += fmt.Sprintf(" after %d retries", t.Retries)
		}
	} else {
		s += "errored"
		if t.Retries > 0 {
			s += fmt.Sprintf(" retried %d times", t.Retries)
		}
		if ctx.Debug > 0 {
			s += fmt.Sprintf(": %+v", t.Err)
		}
	}
	return s
}

// CSVHeader - CSV header fields
func CSVHeader() []string {
	return []string{"timestamp", "project", "filename", "datasource", "full_datasource", "sub_project", "p2o", "endpoint", "config", "commandline", "duration", "seconds", "retries", "error"}
}

// ToCSVNotRedacted - outputs array of string for CSV output of this task (without redacting sensitive data)
func (t Task) ToCSVNotRedacted() []string {
	confAry := []string{}
	for _, config := range t.Config {
		confAry = append(confAry, config.String())
	}
	err := ""
	if t.Err != nil {
		err = fmt.Sprintf("%+v", t.Err)
	}
	return []string{
		fmt.Sprintf("%+v", time.Now()),
		t.FxSlug,
		t.FxFn,
		t.DsSlug,
		t.DsFullSlug,
		t.Project,
		fmt.Sprintf("%v", t.ProjectP2O),
		t.Endpoint,
		"{" + strings.Join(confAry, ", ") + "}",
		t.CommandLine,
		t.Duration.String(),
		fmt.Sprintf("%.3f", t.Duration.Seconds()),
		fmt.Sprintf("%d", t.Retries),
		err,
	}
}

// ToCSV - outputs array of string for CSV output of this task
func (t Task) ToCSV() []string {
	confAry := []string{}
	for _, config := range t.Config {
		confAry = append(confAry, config.RedactedString())
	}
	err := ""
	if t.Err != nil {
		err = fmt.Sprintf("%+v", t.Err)
	}
	return []string{
		fmt.Sprintf("%+v", time.Now()),
		t.FxSlug,
		t.FxFn,
		t.DsSlug,
		t.DsFullSlug,
		t.Project,
		fmt.Sprintf("%v", t.ProjectP2O),
		t.Endpoint,
		"{" + strings.Join(confAry, ", ") + "}",
		t.RedactedCommandLine,
		t.Duration.String(),
		fmt.Sprintf("%.3f", t.Duration.Seconds()),
		fmt.Sprintf("%d", t.Retries),
		err,
	}
}

// TaskResult is a return type from task execution
// It contains task index Code[0], error code Code[1] and task final commandline
type TaskResult struct {
	Code                [2]int
	CommandLine         string
	RedactedCommandLine string
	Retries             int
	Affs                bool
	Err                 error
	Index               string
	Endpoint            string
	Ds                  string
	Fx                  string
	Projects            []EndpointProject
}

// TaskMtx - holds are mutexes used in task processing
type TaskMtx struct {
	SSHKeyMtx    *sync.Mutex
	TaskOrderMtx *sync.Mutex
	SyncInfoMtx  *sync.Mutex
	SyncFreqMtx  *sync.RWMutex
	OrderMtx     map[int]*sync.Mutex
}
