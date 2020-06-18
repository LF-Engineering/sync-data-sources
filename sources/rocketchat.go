package syncdatasources

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

type rocketChatChannel struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
}

type rocketChatChannels struct {
	Channels []rocketChatChannel `json:"channels"`
	Count    int                 `json:"count"`
	Offset   int                 `json:"offset"`
	Total    int                 `json:"total"`
}

// GetRocketChatChannels - return list of channels defined on a given RocketChat server
func GetRocketChatChannels(ctx *Ctx, srv, token, uid string) (channels []string, err error) {
	// curl -s -H "X-Auth-Token: aaa_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb-cccc" -H "X-User-Id: ddddddddddddddddd" 'server_url/api/v1/channels.list?fields=%7b%22name%22%3a1%7d&count=100&offset=100'
	if ctx.Debug > 0 {
		Printf("GetRocketChatChannels(%s, %s, %s)\n", srv, token, uid)
	}
	method := Get
	offset := 0
	count := 100
	queryRoot := url.QueryEscape(`fields={"name":1}`) + fmt.Sprintf("&count=%d&", count)
	for {
		query := queryRoot + fmt.Sprintf("offset=%d", offset)
		url := fmt.Sprintf("%s/api/v1/channels.list?%s", srv, query)
		var req *http.Request
		req, err = http.NewRequest(method, os.ExpandEnv(url), nil)
		if err != nil {
			Printf("New request error: %+v for %s url: %s\n", err, method, url)
			return
		}
		req.Header.Set("X-Auth-Token", token)
		req.Header.Set("X-User-Id", uid)
		var resp *http.Response
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			Printf("Do request error: %+v for %s url: %s\n", err, method, url)
			return
		}
		if resp.StatusCode != 200 {
			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				Printf("ReadAll request error: %+v for %s url: %s\n", err, method, url)
				return
			}
			Printf("Method:%s url:%s status:%d\n%s\n", method, url, resp.StatusCode, body)
			return
		}
		chans := rocketChatChannels{}
		err = json.NewDecoder(resp.Body).Decode(&chans)
		_ = resp.Body.Close()
		if err != nil {
			Printf("JSON decode error: %+v for %s url: %s\n", err, method, url)
			return
		}
		got := chans.Count + chans.Offset
		all := chans.Total
		miss := all - got
		if ctx.Debug > 0 {
			Printf("RocketChat: all: %d, got: %d, miss: %d, channels: %+v\n", all, got, miss, chans)
		}
		for _, channel := range chans.Channels {
			channels = append(channels, channel.Name)
		}
		if miss <= 0 {
			break
		}
		offset += count
	}
	//fmt.Printf("channels: %+v\n", channels)
	return
}
