package syncdatasources

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// GetGerritRepos - return list of repos for given gerrit server (uses HTML crawler)
func GetGerritRepos(ctx *Ctx, gerritURL string) (projects, repos []string, err error) {
	if ctx.Debug >= 0 {
		Printf("GetGerritRepos(%s)\n", gerritURL)
	}
	partials := []string{"r", "gerrit"}
	for _, partial := range partials {
		method := Get
		if !strings.HasSuffix(gerritURL, "/") {
			gerritURL += "/"
		}
		url := gerritURL + partial + "/projects/"
		var req *http.Request
		req, err = http.NewRequest(method, os.ExpandEnv(url), nil)
		if err != nil {
			Printf("new request error: %+v for %s url: %s", err, method, url)
			return
		}
		var resp *http.Response
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			Printf("do request error: %+v for %s url: %s\n", err, method, url)
			return
		}
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			Printf("ReadAll request error: %+v for %s url: %s\n", err, method, url)
			return
		}
		_ = resp.Body.Close()
		if resp.StatusCode == 404 {
			continue
		}
		if resp.StatusCode != 200 {
			err = fmt.Errorf("Method:%s url:%s status:%d\n%s", method, url, resp.StatusCode, body)
			return
		}
		var (
			i int
			b byte
		)
		jsonStart := []byte("{")[0]
		for i, b = range body {
			if b == jsonStart {
				break
			}
		}
		body = body[i:]
		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		if err != nil {
			Printf("Bulk result unmarshal error: %+v", err)
			return
		}
		for project := range result {
			if project == "All-Projects" || project == "All-Users" {
				continue
			}
			ary := strings.Split(project, "/")
			org := ary[0]
			endpoint := gerritURL + partial + "/" + project
			projects = append(projects, org)
			repos = append(repos, endpoint)
		}
		break
	}
	return
}
