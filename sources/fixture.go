package syncdatasources

import "fmt"

// Config holds data source config options
type Config struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Endpoint holds data source endpoint options
type Endpoint struct {
	Name string `yaml:"name"`
}

// DataSource contains data source spec from dev-analytics-api
type DataSource struct {
	Slug      string     `yaml:"slug"`
	Config    []Config   `yaml:"config"`
	Endpoints []Endpoint `yaml:"endpoints"`
}

// Fixture contains full YAML structure of dev-analytics-api fixture files
type Fixture struct {
	Disabled    bool              `yaml:"disabled"`
	Native      map[string]string `yaml:"native"`
	DataSources []DataSource      `yaml:"data_sources"`
	Fn          string
	Slug        string
}

// Task holds single endpoint task and its context (required config, fixture filename etc.)
type Task struct {
	Endpoint string
	Config   []Config
	DsSlug   string
	FxSlug   string
	FxFn     string
}

func (t Task) String() string {
	configStr := "["
	for _, cfg := range t.Config {
		configStr += cfg.Name + " "
	}
	configStr += "]"
	return fmt.Sprintf("{Endpoint:%s DS:%s Slug:%s File:%s Configs:%s}", t.Endpoint, t.DsSlug, t.FxSlug, t.FxFn, configStr)
}

// MultiConfig holds massaged config options, it can have >1 value for single option, for example
// GitHub API tokens: -t token1 token2 token3 ... tokenN
type MultiConfig struct {
	Name  string
	Value []string
}
