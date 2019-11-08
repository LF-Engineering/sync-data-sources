package syncdatasources

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
	Native      map[string]string `yaml:"native"`
	DataSources []DataSource      `yaml:"data_sources"`
	Fn          string
}
