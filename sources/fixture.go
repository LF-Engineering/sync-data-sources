package syncdatasources

// Fixture contains full YAML structure of dev-analytics-api fixture files
type Fixture struct {
	Native map[string]string `yaml:"native"`
	Fn     string
}
