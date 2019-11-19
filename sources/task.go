package syncdatasources

import "fmt"

// Task holds single endpoint task and its context (required config, fixture filename etc.)
type Task struct {
	Endpoint    string
	Config      []Config
	DsSlug      string
	FxSlug      string
	FxFn        string
	CommandLine string
	Retries     int
	Err         error
}

// String - default string output for a task (generic)
func (t Task) String() string {
	configStr := "["
	for _, cfg := range t.Config {
		configStr += cfg.Name + " "
	}
	configStr += "]"
	return fmt.Sprintf(
		"{Endpoint:%s DS:%s Slug:%s File:%s Configs:%s Cmd:%s Retries:%d, Error:%v}",
		t.Endpoint, t.DsSlug, t.FxSlug, t.FxFn, configStr, t.CommandLine, t.Retries, t.Err != nil,
	)
}

// ShortString - output quick endpoint info (usually used for non finished tasks)
func (t Task) ShortString() string {
	return fmt.Sprintf("%s: %s / %s", t.FxSlug, t.DsSlug, t.Endpoint)
}

// ShortStringCmd - output quick endpoint info (with command line)
func (t Task) ShortStringCmd(ctx *Ctx) string {
	s := fmt.Sprintf("%s: %s / %s [%s]: ", t.FxSlug, t.DsSlug, t.Endpoint, t.CommandLine)
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

// TaskResult is a return type from task execution
// It contains task index Code[0], error code Code[1] and task final commandline
type TaskResult struct {
	Code        [2]int
	CommandLine string
	Retries     int
	Err         error
}
