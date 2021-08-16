package syncdatasources

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/LF-Engineering/dev-analytics-libraries/auth0"
	"github.com/LF-Engineering/dev-analytics-libraries/elastic"
	"github.com/LF-Engineering/dev-analytics-libraries/http"
	"github.com/LF-Engineering/dev-analytics-libraries/slack"
	jsoniter "github.com/json-iterator/go"
)

var (
	gAuth0Client *auth0.ClientProvider
	gTokenEnv    string
)

// InitializeAuth0 - initializes Auth0 client using data stored in AUTH0_DATA
func InitializeAuth0() error {
	// DEBUG:
	fmt.Printf("DEBUG: InitializeAuth0\n")
	var err error
	auth0DataB64 := os.Getenv("AUTH0_DATA")
	if auth0DataB64 == "" {
		return fmt.Errorf("you must specify AUTH0_DATA (so the program can generate an API token) or specify token with JWT_TOKEN")
	}
	var auth0Data []byte
	auth0Data, err = base64.StdEncoding.DecodeString(auth0DataB64)
	if err != nil {
		Printf("decode base64 error: %+v\n", err)
		return err
	}
	var data map[string]string
	err = jsoniter.Unmarshal([]byte(auth0Data), &data)
	if err != nil {
		Printf("unmarshal error: %+v\n", err)
		return err
	}
	// DEBUG:
	fmt.Printf("DEBUG: InitializeAuth0: unmarshaled '%+v'\n", data)
	fmt.Printf("DEBUG: InitializeAuth0: env: '%s'\n", data["env"])
	fmt.Printf("DEBUG: InitializeAuth0: es_url: '%s'\n", data["es_url"])
	fmt.Printf("DEBUG: InitializeAuth0: es_user: '%s'\n", data["es_user"])
	fmt.Printf("DEBUG: InitializeAuth0: es_pass: '%s'\n", data["es_pass"])
	fmt.Printf("DEBUG: InitializeAuth0: url: '%s'\n", data["url"])
	fmt.Printf("DEBUG: InitializeAuth0: slack: '%s'\n", data["slack_webhook_url"])
	AddRedacted(data["es_url"], false)
	AddRedacted(data["es_user"], false)
	AddRedacted(data["es_pass"], false)
	AddRedacted(data["client_id"], false)
	AddRedacted(data["client_secret"], false)
	AddRedacted(data["audience"], false)
	AddRedacted(data["url"], false)
	AddRedacted(data["slack_webhook_url"], false)
	// Providers
	httpClientProvider := http.NewClientProvider(60 * time.Second)
	esCacheClientProvider, err := elastic.NewClientProvider(
		&elastic.Params{
			URL:      data["es_url"],
			Username: data["es_user"],
			Password: data["es_pass"],
		},
	)
	// DEBUG:
	fmt.Printf("DEBUG: InitializeAuth0: esProvider: %+v\n", esCacheClientProvider)
	if err != nil {
		Printf("ES client provider error: %+v\n", err)
		return err
	}
	slackProvider := slack.New(data["slack_webhook_url"])
	// DEBUG:
	fmt.Printf("DEBUG: InitializeAuth0: slackProvider: %+v\n", slackProvider)
	gAuth0Client, err = auth0.NewAuth0Client(
		data["env"],
		data["grant_type"],
		data["client_id"],
		data["client_secret"],
		data["audience"],
		data["url"],
		httpClientProvider,
		esCacheClientProvider,
		&slackProvider,
		"SDS",
	)
	if err == nil {
		gTokenEnv = data["env"]
	}
	// DEBUG:
	fmt.Printf("DEBUG: InitializeAuth0: gAuth0Client(%s): %+v,%+v\n", data["env"], gAuth0Client, err)
	return err
}

// GetAPIToken - return an API token to use dev-analytics-api API calls
// If JWT_TOKEN env is specified - just use the provided token without any checks
// Else get auth0 data from AUTH0_DATA and generate/reuse a token stored in ES cache
func GetAPIToken() (string, error) {
	envToken := os.Getenv("JWT_TOKEN")
	// DEBUG:
	fmt.Printf("DEBUG: GetAPIToken: envToken: '%s'\n", envToken)
	if envToken != "" {
		return envToken, nil
	}
	if gTokenEnv == "" {
		// DEBUG:
		fmt.Printf("DEBUG: GetAPIToken: will call initialize\n")
		err := InitializeAuth0()
		if err != nil {
			return "", err
		}
	}
	token, err := gAuth0Client.GetToken()
	// DEBUG:
	fmt.Printf("DEBUG: GetAPIToken: GetToken -> ('%s',%+v)\n", token, err)
	if err == nil && token != "" {
		token = "Bearer " + token
	}
	return token, err
}
