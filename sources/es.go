package syncdatasources

// EsIndex - keeps index data as returned by ElasticSearch
type EsIndex struct {
	Index string `json:"index"`
}

// EsAlias - keeps alias data as returned by ElasticSearch
type EsAlias struct {
	Alias string `json:"alias"`
	Index string `json:"index"`
}
