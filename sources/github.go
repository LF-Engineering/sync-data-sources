package syncdatasources

import (
	"context"
	"io/ioutil"
	"strings"
	"time"

	// "github.com/google/go-github/v33/github" // with go mod enabled
	"github.com/google/go-github/github" // with go mod disabled
	"golang.org/x/oauth2"
)

// GHClient - get GitHub client
func GHClient(ctx *Ctx) (ghCtx context.Context, clients []*github.Client) {
	// Get GitHub OAuth from env or from file
	oAuth := ctx.GitHubOAuth
	if strings.Contains(ctx.GitHubOAuth, "/") {
		bytes, err := ioutil.ReadFile(ctx.GitHubOAuth)
		FatalOnError(err)
		oAuth = strings.TrimSpace(string(bytes))
	}

	// GitHub authentication or use public access
	ghCtx = context.Background()
	if oAuth == "" {
		client := github.NewClient(nil)
		clients = append(clients, client)
	} else {
		oAuths := strings.Split(oAuth, ",")
		for _, auth := range oAuths {
			ctx.OAuthKeys = append(ctx.OAuthKeys, auth)
			AddRedacted(auth, false)
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: auth},
			)
			tc := oauth2.NewClient(ghCtx, ts)
			client := github.NewClient(tc)
			clients = append(clients, client)
		}
	}
	return
}

// GHClientForKeys - get GitHub client for given keys
func GHClientForKeys(oAuths map[string]struct{}) (ghCtx context.Context, clients []*github.Client) {
	// GitHub authentication or use public access
	ghCtx = context.Background()
	for auth := range oAuths {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: auth},
		)
		tc := oauth2.NewClient(ghCtx, ts)
		client := github.NewClient(tc)
		clients = append(clients, client)
	}
	return
}

// GetRateLimits - returns all and remaining API points and duration to wait for reset
// when core=true - returns Core limits, when core=false returns Search limits
func GetRateLimits(gctx context.Context, ctx *Ctx, gcs []*github.Client, core bool) (int, []int, []int, []time.Duration) {
	var (
		limits     []int
		remainings []int
		durations  []time.Duration
	)
	for idx, gc := range gcs {
		rl, _, err := gc.RateLimits(gctx)
		if err != nil {
			rem, ok := PeriodParse(err.Error())
			if ok {
				Printf("Parsed wait time from error message: %v\n", rem)
				limits = append(limits, -1)
				remainings = append(remainings, -1)
				durations = append(durations, rem)
				continue
			}
			Printf("GetRateLimit(%d): %v\n", idx, err)
		}
		if rl == nil {
			limits = append(limits, -1)
			remainings = append(remainings, -1)
			durations = append(durations, time.Duration(5)*time.Second)
			continue
		}
		if core {
			limits = append(limits, rl.Core.Limit)
			remainings = append(remainings, rl.Core.Remaining)
			durations = append(durations, rl.Core.Reset.Time.Sub(time.Now())+time.Duration(1)*time.Second)
			continue
		}
		limits = append(limits, rl.Search.Limit)
		remainings = append(remainings, rl.Search.Remaining)
		durations = append(durations, rl.Search.Reset.Time.Sub(time.Now())+time.Duration(1)*time.Second)
	}
	hint := 0
	for idx := range limits {
		if remainings[idx] > remainings[hint] {
			hint = idx
		} else if idx != hint && remainings[idx] == remainings[hint] && durations[idx] < durations[hint] {
			hint = idx
		}
	}
	if ctx.Debug > 0 {
		Printf("GetRateLimits: hint: %d, limits: %+v, remaining: %+v, reset: %+v\n", hint, limits, remainings, durations)
	}
	return hint, limits, remainings, durations
}
