package syncdatasources

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
)

type DockerHubData struct {
	Count   int                `json:"count"`
	Next    string             `json:"next"`
	Results []DockerHubResults `json:"results"`
}

type DockerHubResults struct {
	User string `json:"user"`
	Name string `json:"name"`
}

// GetDockerHubRepos - return list of repos for given dockerhub server
func GetDockerHubRepos(ctx *Ctx, dockerhubOwner string) (repos []string, err error) {
	pageSize := "50"
	if ctx.Debug >= 0 {
		Printf("GetDockerHubRepos(%s)\n", dockerhubOwner)
	}

	method := Get
	url := "https://hub.docker.com/v2/repositories/" + dockerhubOwner + "?page_size=" + pageSize

	for {

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

		var data DockerHubData
		err = json.Unmarshal(body, &data)
		if err != nil {
			Printf("Bulk result unmarshal error: %+v", err)
			return
		}

		for _, repo := range data.Results {
			repos = append(repos, repo.User+" "+repo.Name)
		}

		if data.Next == "" {
			break
		}

		url = data.Next
	}

	return repos, nil

}
