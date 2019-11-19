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
}

func (t Task) String() string {
	configStr := "["
	for _, cfg := range t.Config {
		configStr += cfg.Name + " "
	}
	configStr += "]"
	return fmt.Sprintf("{Endpoint:%s DS:%s Slug:%s File:%s Configs:%s Cmd:%s}", t.Endpoint, t.DsSlug, t.FxSlug, t.FxFn, configStr, t.CommandLine)
}

// ShortString - output quick endpoint info
func (t Task) ShortString() string {
	return fmt.Sprintf("%s: %s / %s", t.FxSlug, t.DsSlug, t.Endpoint)
}

// ShortStringCmd - output quick endpoint info (with command line)
func (t Task) ShortStringCmd() string {
	return fmt.Sprintf("%s: %s / %s [%s]", t.FxSlug, t.DsSlug, t.Endpoint, t.CommandLine)
}

// TaskResult is a return type from task execution
// It contains task index Code[0], error code Code[1] and task final commandline
type TaskResult struct {
	Code        [2]int
	CommandLine string
	Retries     int
	Err         error
}
