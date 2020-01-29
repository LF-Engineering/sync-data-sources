package syncdatasources

import "time"

// EsIndex - keeps index data as returned by ElasticSearch
type EsIndex struct {
	Index string `json:"index"`
}

// EsAlias - keeps alias data as returned by ElasticSearch
type EsAlias struct {
	Alias string `json:"alias"`
	Index string `json:"index"`
}

// ES search

// EsSearchQueryString - ES search query string
type EsSearchQueryString struct {
	Query string `json:"query"`
}

// EsSearchQuery - ES search query
type EsSearchQuery struct {
	QueryString EsSearchQueryString `json:"query_string"`
}

// EsSearchPayload - ES search payload
type EsSearchPayload struct {
	Query EsSearchQuery `json:"query"`
}

// ES search result

// EsSearchResultSource - search result single hit's  source document
type EsSearchResultSource struct {
	Index    string    `json:"index"`
	Endpoint string    `json:"endpoint"`
	Type     string    `json:"type"`
	Dt       time.Time `json:"dt"`
}

// EsSearchResultHit - search result single hit
type EsSearchResultHit struct {
	Source EsSearchResultSource `json:"_source"`
}

// EsSearchResultHits - search result hits
type EsSearchResultHits struct {
	Hits []EsSearchResultHit `json:"hits"`
}

// EsSearchResultPayload - search result payload
type EsSearchResultPayload struct {
	Hits EsSearchResultHits `json:"hits"`
}

// ES last_run support in sdsdata index

// EsLastRunPayload - last run support
type EsLastRunPayload struct {
	Index    string    `json:"index"`
	Endpoint string    `json:"endpoint"`
	Type     string    `json:"type"`
	Dt       time.Time `json:"dt"`
}
