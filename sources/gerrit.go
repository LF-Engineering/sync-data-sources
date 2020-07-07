package syncdatasources

// GetGerritRepos - return list of repos for given gerrit server (uses HTML crawler)
func GetGerritRepos(ctx *Ctx, url string) (repos []string, err error) {
	if ctx.Debug >= 0 {
		Printf("GetGerritRepos(%s)\n", url)
	}
	return
}
