package syncdatasources

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

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

// EnsureIndex - ensure that given index exists in ES
// init: when this flag is set, do not use syncdatasources.Printf which would cause infinite recurence
func EnsureIndex(ctx *Ctx, index string, init bool) {
	printf := Printf
	if init {
		printf = fmt.Printf
	}
	method := Head
	url := fmt.Sprintf("%s/%s", ctx.ElasticURL, index)
	req, err := http.NewRequest(method, os.ExpandEnv(url), nil)
	if err != nil {
		printf("New request error: %+v for %s url: %s\n", err, method, url)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		printf("Do request error: %+v for %s url: %s\n", err, method, url)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		if resp.StatusCode != 404 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				printf("ReadAll request error: %+v for %s url: %s\n", err, method, url)
				return
			}
			printf("Method:%s url:%s status:%d\n%s\n", method, url, resp.StatusCode, body)
			return
		}
		printf("Missing %s index, creating\n", index)
		method = Put
		req, err := http.NewRequest(method, os.ExpandEnv(url), nil)
		if err != nil {
			printf("New request error: %+v for %s url: %s\n", err, method, url)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			printf("Do request error: %+v for %s url: %s\n", err, method, url)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode != 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				printf("ReadAll request error: %+v for %s url: %s\n", err, method, url)
				return
			}
			printf("Method:%s url:%s status:%d\n%s\n", method, url, resp.StatusCode, body)
			return
		}
		printf("%s index created\n", index)
	}
}
