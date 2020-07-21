package syncdatasources

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
)

type slackChannel struct {
	ID       string `json:"id"`
	Name     string `json:"name_normalized"`
	Archived bool   `json:"is_archived"`
	Channel  bool   `json:"is_channel"`
}

type responseMetadata struct {
	Cursor string `json:"next_cursor"`
}

type slackChannels struct {
	Channels []slackChannel   `json:"channels"`
	Metadata responseMetadata `json:"response_metadata"`
	OK       bool             `json:"ok"`
	Error    string           `json:"error"`
}

// GetSlackBotUsersConversation - return list of channels (Slack users.converstations API) available for a given slack bot user (specified by a bearer token)
func GetSlackBotUsersConversation(ctx *Ctx, token string) (ids, channels []string, err error) {
	// curl -s -X POST -H 'Authorization: Bearer ...' -H 'Content-type: application/x-www-form-urlencoded' https://slack.com/api/users.conversations -d "limit=200&cursor=..." | jq .
	rtoken := token[len(token)-len(token)/4:]
	if ctx.Debug > 0 {
		Printf("GetSlackBotUsersConversation(%s)\n", rtoken)
	}
	method := Post
	cursor := ""
	for {
		url := "https://slack.com/api/users.conversations"
		var req *http.Request
		strData := "limit=200"
		if cursor != "" {
			strData += "&cursor=" + cursor
		}
		data := []byte(strData)
		payloadBody := bytes.NewReader(data)
		req, err = http.NewRequest(method, os.ExpandEnv(url), payloadBody)
		if err != nil {
			Printf("New request error: %+v for %s url: %s\n", err, method, url)
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-type", "application/x-www-form-urlencoded")
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
		chans := slackChannels{}
		err = json.NewDecoder(resp.Body).Decode(&chans)
		_ = resp.Body.Close()
		if err != nil {
			Printf("JSON decode error: %+v for %s url: %s\n", err, method, url)
			return
		}
		if !chans.OK {
			Printf("%s: API returned an error state: %+v\n", rtoken, chans)
			return
		}
		for _, channel := range chans.Channels {
			if channel.Archived || !channel.Channel {
				continue
			}
			ids = append(ids, channel.ID)
			channels = append(channels, channel.Name)
		}
		cursor = chans.Metadata.Cursor
		if cursor == "" {
			break
		}
	}
	return
}
