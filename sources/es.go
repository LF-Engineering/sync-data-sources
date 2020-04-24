package syncdatasources

import (
	"bytes"
	"encoding/json"
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
	Index     string    `json:"index"`
	Endpoint  string    `json:"endpoint"`
	Type      string    `json:"type"`
	Mtx       string    `json:"mtx"`
	Dt        time.Time `json:"dt"`
	ProjectTS int64     `json:"project_ts"`
}

// EsSearchResultHit - search result single hit
type EsSearchResultHit struct {
	Source EsSearchResultSource `json:"_source"`
	ID     string               `json:"_id"`
}

// EsSearchResultHits - search result hits
type EsSearchResultHits struct {
	Hits []EsSearchResultHit `json:"hits"`
}

// EsSearchResultPayload - search result payload
type EsSearchResultPayload struct {
	Hits         EsSearchResultHits `json:"hits"`
	Aggregations interface{}        `json:"aggregations"`
}

// EsUpdateByQueryPayload - update by query result payload
type EsUpdateByQueryPayload struct {
	Updated int64 `json:"updated"`
}

// ES last_run support in sdsdata index

// EsLastRunPayload - last run support
type EsLastRunPayload struct {
	Index    string    `json:"index"`
	Endpoint string    `json:"endpoint"`
	Type     string    `json:"type"`
	Dt       time.Time `json:"dt"`
}

// EsSyncInfoPayload - sync info support
type EsSyncInfoPayload struct {
	Index             string     `json:"index"`
	Endpoint          string     `json:"endpoint"`
	Dt                time.Time  `json:"dt"`
	DataSyncAttemptDt *time.Time `json:"data_sync_attempt_dt"`
	DataSyncSuccessDt *time.Time `json:"data_sync_success_dt"`
	DataSyncErrorDt   *time.Time `json:"data_sync_error_dt"`
	DataSyncError     *string    `json:"data_sync_error"`
	DataSyncCL        *string    `json:"data_sync_command_line"`
	DataSyncRCL       *string    `json:"data_sync_redacted_command_line"`
	EnrichAttemptDt   *time.Time `json:"enrich_attempt_dt"`
	EnrichSuccessDt   *time.Time `json:"enrich_success_dt"`
	EnrichErrorDt     *time.Time `json:"enrich_error_dt"`
	EnrichError       *string    `json:"enrich_error"`
	EnrichCL          *string    `json:"enrich_command_line"`
	EnrichRCL         *string    `json:"enrich_redacted_command_line"`
}

// EsMtxPayload - ES mutex support (for locking concurrent nodes)
type EsMtxPayload struct {
	Mtx string    `json:"mtx"`
	Dt  time.Time `json:"dt"`
}

// EsLogPayload - ES log single document
type EsLogPayload struct {
	Msg string    `json:"msg"`
	Dt  time.Time `json:"dt"`
}

// EsIndexSettings - index settings
type EsIndexSettings struct {
	IndexBlocksWrite *bool `json:"index.blocks.write"`
}

// EsIndexSettingsPayload - index settings payload
type EsIndexSettingsPayload struct {
	Settings EsIndexSettings `json:"settings"`
}

// EsScript - internal
type EsScript struct {
	Inline string `json:"inline"`
}

// EsTermOrigin - internal
type EsTermOrigin struct {
	Origin string `json:"origin"`
}

// EsQueryTerm - internal
type EsQueryTerm struct {
	Term EsTermOrigin `json:"term"`
}

// EsUpdateByQuery - to update/add project fields on all documents with a given origin
type EsUpdateByQuery struct {
	Script EsScript    `json:"script"`
	Query  EsQueryTerm `json:"query"`
}

// EnsureIndex - ensure that given index exists in ES
// init: when this flag is set, do not use syncdatasources.Printf which would cause infinite recurence
func EnsureIndex(ctx *Ctx, index string, init bool) {
	printf := Printf
	if init {
		printf = PrintfRedacted
	}
	method := Head
	url := fmt.Sprintf("%s/%s", ctx.ElasticURL, index)
	rurl := fmt.Sprintf("/%s", index)
	req, err := http.NewRequest(method, os.ExpandEnv(url), nil)
	if err != nil {
		printf("New request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		if resp.StatusCode != 404 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				printf("ReadAll request error: %+v for %s url: %s\n", err, method, rurl)
				return
			}
			printf("Method:%s url:%s status:%d\n%s\n", method, rurl, resp.StatusCode, body)
			return
		}
		printf("Missing %s index, creating\n", index)
		method = Put
		req, err := http.NewRequest(method, os.ExpandEnv(url), nil)
		if err != nil {
			printf("New request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			printf("Do request error: %+v for %s url: %s\n", err, method, rurl)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode != 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				printf("ReadAll request error: %+v for %s url: %s\n", err, method, rurl)
				return
			}
			printf("Method:%s url:%s status:%d\n%s\n", method, rurl, resp.StatusCode, body)
			return
		}
		printf("%s index created\n", index)
	}
}

// EsLog - log data into ES "sdslog" index
func EsLog(ctx *Ctx, msg string, dt time.Time) error {
	data := EsLogPayload{Msg: msg, Dt: dt}
	index := "sdslog"
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		PrintfRedacted("JSON marshall error: %+v for index: %s, data: %+v\n", err, index, data)
		return err
	}
	payloadBody := bytes.NewReader(payloadBytes)
	method := Post
	url := fmt.Sprintf("%s/%s/_doc", ctx.ElasticURL, index)
	rurl := fmt.Sprintf("/%s/_doc", index)
	req, err := http.NewRequest(method, os.ExpandEnv(url), payloadBody)
	if err != nil {
		PrintfRedacted("New request error: %+v for %s url: %s, data: %+v\n", err, method, rurl, data)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		PrintfRedacted("Do request error: %+v for %s url: %s, data: %+v\n", err, method, rurl, data)
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 201 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			PrintfRedacted("ReadAll request error: %+v for %s url: %s, data: %+v\n", err, method, rurl, data)
			return err
		}
		PrintfRedacted("Method:%s url:%s status:%d, data:%+v\n%s\n", method, rurl, resp.StatusCode, data, body)
		return err
	}
	return nil
}
