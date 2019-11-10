package syncdatasources

// OK - common constant string
const OK string = "ok"

// APIToken - constant string
const APIToken string = "api-token"

// Git - git
const Git string = "git"

// GitHub - github
const GitHub string = "github"

// Gerrit - gerrit
const Gerrit string = "gerrit"

// Jira - jira
const Jira string = "jira"

// Confluence - confluence
const Confluence string = "confluence"

// Slack - slack
const Slack string = "slack"

// GroupsIO - groupsio
const GroupsIO string = "groupsio"

// PyException - string that identified python exception
const PyException string = "Traceback (most recent call last)"

// BackendUser - backend-user
const BackendUser string = "backend-user"

// BackendPassword - backend-password
const BackendPassword string = "backend-password"

// ErrorStrings - array of possible errors returned from enrich tasks
var ErrorStrings = map[int]string{
	1: "datasource slug contains > 1 '/' separators",
	2: "incorrect endpoint value for given data source",
	3: "incorrect config option(s) for given data source",
	4: "p2o.py error",
}
