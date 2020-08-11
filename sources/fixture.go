package syncdatasources

import (
	"fmt"
	"regexp"
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

// RedactedString - redacted string output
func (c Config) RedactedString() string {
	name := c.Name
	if IsRedacted(name) {
		return fmt.Sprintf(
			"'%s':'%s'",
			c.Name,
			Redacted,
		)
	}
	return fmt.Sprintf(
		"'%s':'%s'",
		c.Name,
		c.Value,
	)
}

// IsRedacted - returns whatever "name" config option should be redacted or not
func IsRedacted(name string) bool {
	if name == APIToken || name == Email || name == User || name == SSHKey || name == BackendUser || name == BackendPassword || name == Password || name == UserID {
		return true
	}
	return false
}

// ColumnCondition - holds single must or must_not condition for setting project witing a single endpoint
type ColumnCondition struct {
	Column string `yaml:"column"`
	Value  string `yaml:"value"`
}

// EndpointProject - holds data for a single sub-endpoint project configuration
type EndpointProject struct {
	Name    string            `yaml:"name"`
	Origin  string            `yaml:"origin"`
	Must    []ColumnCondition `yaml:"must"`
	MustNot []ColumnCondition `yaml:"must_not"`
}

// CopyConfig - holds data related to copy from other index configuration
type CopyConfig struct {
	Pattern  string `yaml:"pattern"`
	NoOrigin bool   `yaml:"no_origin"` // skip checking origin when calculating start date to copy
	// if no_origin is set, then copying will start from the date of the last document stored in the destination index
	//    (can be used when the source has multiple origins or origin(s) different than endpoint's origin)
	// if no_origin is not set it will query destination index for origin of the destination endpoint
	//    and will start copying source -> dest from that date (this is the default)
	Must    []ColumnCondition `yaml:"must"`
	MustNot []ColumnCondition `yaml:"must_not"`
}

// Endpoint holds data source endpoint (final endpoint generated from RawEndpoint)
type Endpoint struct {
	Name       string // Endpoint name
	Project    string // optional project (allows groupping endpoints), for example "Project value"
	ProjectP2O bool   // if true SDS will pass `--project "Project value"` to p2o.py
	// if false, SDS will post-process index and will add `"project": "Project value"`
	// column where `"origin": "Endpoint name"`
	Timeout           time.Duration // specifies maximum running time for a given endpoint (if specified)
	CopyFrom          CopyConfig    // specifies optional 'copy_from' configuration
	AffiliationSource string
	Projects          []EndpointProject
	PairProgramming   bool
}

// RawEndpoint holds data source endpoint with possible flags how to generate the final endpoints
// flags can be "type: github_org/github_user" which means that we need to get actual repository list from github org/user
type RawEndpoint struct {
	Name              string            `yaml:"name"`
	Flags             map[string]string `yaml:"flags"`
	Skip              []string          `yaml:"skip"`
	Only              []string          `yaml:"only"`
	Project           string            `yaml:"project"`
	ProjectP2O        *bool             `yaml:"p2o"`
	Timeout           *string           `yaml:"timeout"`
	Projects          []EndpointProject `yaml:"endpoint_projects"`
	CopyFrom          CopyConfig        `yaml:"copy_from"`
	AffiliationSource string            `yaml:"affiliation_source"`
	PairProgramming   bool              `yaml:"pair_programming"`
	SkipREs           []*regexp.Regexp  `yaml:"-"`
	OnlyREs           []*regexp.Regexp  `yaml:"-"`
}

// EndpointIncluded - checks if given endpoint's origin should be included or excluded based on endpoint's skip/only regular expressions lists
// First return value specifies if endpoint is included or not
// Second value specifies: 1 - included by 'only' condition, 2 - skipped by 'skip' condition
func EndpointIncluded(ctx *Ctx, ep *RawEndpoint, origin string) (bool, int) {
	for _, skipRE := range ep.SkipREs {
		if skipRE.MatchString(origin) {
			if ctx.Debug > 0 {
				fmt.Printf("%s: skipped %s (%v)\n", ep.Name, origin, skipRE)
			}
			return false, 2
		}
	}
	if len(ep.OnlyREs) == 0 {
		if ctx.Debug > 0 {
			fmt.Printf("%s: included all\n", ep.Name)
		}
		return true, 0
	}
	included := false
	inc := 0
	for _, onlyRE := range ep.OnlyREs {
		if onlyRE.MatchString(origin) {
			if ctx.Debug > 0 {
				fmt.Printf("%s: included %s (%v)\n", ep.Name, origin, onlyRE)
			}
			included = true
			inc = 1
			break
		}
	}
	return included, inc
}

// Project holds project data and list of endpoints
type Project struct {
	Name         string        `yaml:"name"`
	P2O          *bool         `yaml:"p2o"`
	RawEndpoints []RawEndpoint `yaml:"endpoints"`
}

// DataSource contains data source spec from dev-analytics-api
type DataSource struct {
	Slug         string        `yaml:"slug"`
	Config       []Config      `yaml:"config"`
	MaxFrequency string        `yaml:"max_frequency"`
	Projects     []Project     `yaml:"projects"`
	RawEndpoints []RawEndpoint `yaml:"endpoints"`
	IndexSuffix  string        `yaml:"index_suffix"`
	Endpoints    []Endpoint    `yaml:"-"`
	MaxFreq      time.Duration `yaml:"-"`
	FullSlug     string        `yaml:"-"`
}

// Configs - return redacted configs as a string
func (ds DataSource) Configs() string {
	configStr := "["
	for _, cfg := range ds.Config {
		configStr += cfg.RedactedString()
	}
	configStr += "]"
	return configStr
}

func (ds DataSource) String() string {
	configStr := ds.Configs()
	return fmt.Sprintf(
		"{Slug:%s,Config:%s,MaxFrequency:%s,Projects:%+v,RawEndpoints:%+v,Endpoints:%+v,MaxFreq:%+v,IndexSuffix:%s,FullSlug:%s}",
		ds.Slug,
		configStr,
		ds.MaxFrequency,
		ds.Projects,
		ds.RawEndpoints,
		ds.Endpoints,
		ds.MaxFreq,
		ds.IndexSuffix,
		ds.FullSlug,
	)
}

// Native - keeps fixture slug and eventual global affiliation source
type Native struct {
	Slug              string `yaml:"slug"`
	AffiliationSource string `yaml:"affiliation_source"`
}

// Fixture contains full YAML structure of dev-analytics-api fixture files
type Fixture struct {
	Disabled    bool         `yaml:"disabled"`
	Native      Native       `yaml:"native"`
	DataSources []DataSource `yaml:"data_sources"`
	Aliases     []Alias      `yaml:"aliases"`
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
	Name          string
	Value         []string
	RedactedValue []string
}

func (mc MultiConfig) String() string {
	return fmt.Sprintf(
		"{Name:%s,Value:%+v}",
		mc.Name,
		mc.RedactedValue,
	)
}
