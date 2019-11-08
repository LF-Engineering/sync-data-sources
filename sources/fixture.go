package syncdatasources

// DataSource contains data source spec from dev-analytics-api
type DataSource struct {
	Slug string `yaml:"slug"`
}

// Fixture contains full YAML structure of dev-analytics-api fixture files
type Fixture struct {
	Native      map[string]string `yaml:"native"`
	DataSources []DataSource      `yaml:"data_sources"`
	Fn          string
}
