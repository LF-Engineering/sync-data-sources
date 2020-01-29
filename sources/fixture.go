package syncdatasources

import (
	"fmt"
	"time"
)

// Config holds data source config options
type Config struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// String - default string output for a config
func (c Config) String() string {
	return fmt.Sprintf(
		"'%s':'%s'",
		c.Name,
		c.Value,
	)
}

// Endpoint holds data source endpoint (final endpoint generated from RawEndpoint)
type Endpoint struct {
	Name string
}

// RawEndpoint holds data source endpoint with possible flags how to generate the final endpoints
// flags can be "type: github_org/github_user" which means that we need to get actual repository list from github org/user
type RawEndpoint struct {
	Name  string            `yaml:"name"`
	Flags map[string]string `yaml:"flags"`
}

// DataSource contains data source spec from dev-analytics-api
type DataSource struct {
	Slug         string        `yaml:"slug"`
	Config       []Config      `yaml:"config"`
	MaxFrequency string        `yaml:"max_frequency"`
	RawEndpoints []RawEndpoint `yaml:"endpoints"`
	Endpoints    []Endpoint    `yaml:"-"`
	MaxFreq      time.Duration `yaml:"-"`
}

// Fixture contains full YAML structure of dev-analytics-api fixture files
type Fixture struct {
	Disabled    bool              `yaml:"disabled"`
	Native      map[string]string `yaml:"native"`
	DataSources []DataSource      `yaml:"data_sources"`
	Aliases     []Alias           `yaml:"aliases"`
	Fn          string
	Slug        string
}

// Alias conatin indexing aliases data, single index from (source) and list of aliases that should point to that index
type Alias struct {
	From string   `yaml:"from"`
	To   []string `yaml:"to"`
}

// MultiConfig holds massaged config options, it can have >1 value for single option, for example
// GitHub API tokens: -t token1 token2 token3 ... tokenN
type MultiConfig struct {
	Name  string
	Value []string
}
