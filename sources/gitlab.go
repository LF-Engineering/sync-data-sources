package syncdatasources

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

type GitlabGroupData struct {
	ID       int             `json:"id"`
	Name     string          `json:"name"`
	URL      string          `json:"web_url"`
	Path     string          `json:"full_path"`
	Projects []GitlabProject `json:"projects"`
}

type GitlabSubGroupData struct {
	Groups []GitlabGroupData
}

type GitlabProject struct {
	ID   int    `json:"id"`
	Name string `json:"string"`
	URL  string `json:"web_url"`
}

var repos []string

func getProjects(groupID int, token string) error {
	method := Get
	url := fmt.Sprintf("%s/%d", "https://gitlab.com/api/v4/groups", groupID)

	var req *http.Request
	req, err := http.NewRequest(method, os.ExpandEnv(url), nil)
	if err != nil {
		Printf("new request error: %+v for %s url: %s", err, method, url)
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		Printf("do request error: %+v for %s url: %s\n", err, method, url)
		return err
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		Printf("ReadAll request error: %+v for %s url: %s\n", err, method, url)
		return err
	}

	var data GitlabGroupData
	err = jsoniter.Unmarshal(body, &data)
	if err != nil {
		Printf("Bulk result unmarshal error: %+v", err)
		return err
	}

	for _, project := range data.Projects {
		repos = append(repos, project.URL)
	}

	// Check for sub groups
	subgroupUrl := fmt.Sprintf("%s/%d/subgroups", "https://gitlab.com/api/v4/groups", groupID)
	req, err = http.NewRequest(method, os.ExpandEnv(subgroupUrl), nil)
	if err != nil {
		Printf("new request error: %+v for %s url: %s", err, method, url)
		return err
	}

	req.Header.Set("PRIVATE-TOKEN", token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		Printf("do request error: %+v for %s url: %s\n", err, method, url)
		return err
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		Printf("ReadAll request error: %+v for %s url: %s\n", err, method, url)
		return err
	}

	var subGroupdata []GitlabGroupData
	err = jsoniter.Unmarshal(body, &subGroupdata)
	if err != nil {
		Printf("Bulk result unmarshal error: %+v", err)
		return err
	}

	if len(subGroupdata) == 0 {
		return nil
	} else {
		for _, subgroup := range subGroupdata {
			err = getProjects(subgroup.ID, token)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getGroupID(groupname, token string) (int, error) {
	var err error
	method := Get
	url := fmt.Sprintf("%s/%s", "https://gitlab.com/api/v4/groups", groupname)

	var req *http.Request
	req, err = http.NewRequest(method, os.ExpandEnv(url), nil)
	if err != nil {
		Printf("new request error: %+v for %s url: %s", err, method, url)
		return 0, err
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		Printf("do request error: %+v for %s url: %s\n", err, method, url)
		return 0, err
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		Printf("ReadAll request error: %+v for %s url: %s\n", err, method, url)
		return 0, err
	}

	var data GitlabGroupData
	err = jsoniter.Unmarshal(body, &data)
	if err != nil {
		Printf("Bulk result unmarshal error: %+v", err)
		return 0, err
	}

	return data.ID, nil
}

func GetGitlabGroupRepos(ctx *Ctx, groupURL, token string) ([]string, error) {
	if ctx.Debug >= 0 {
		Printf("GetGithubRepos(%s)\n", groupURL)
	}

	u, err := url.Parse(groupURL)
	if err != nil {
		return nil, err
	}

	parentGroupName := strings.TrimLeft(u.Path, "/")
	parentGroupID, err := getGroupID(parentGroupName, token)
	if err != nil {
		return nil, err
	}

	err = getProjects(parentGroupID, token)
	if err != nil {
		return nil, err
	}

	return repos, nil

}
