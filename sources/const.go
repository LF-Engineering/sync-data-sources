package syncdatasources

// OK - common constant string
const OK string = "ok"

// APIToken - constant string
const APIToken string = "api-token"

// GitlabToken - constant string
const GitlabToken string = "gitlab-token"

// SSHKey - constant string
const SSHKey string = "ssh-key"

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

// Pipermail - pipermail
const Pipermail string = "pipermail"

// Discourse - discourse
const Discourse string = "discourse"

// Jenkins - jenkins
const Jenkins string = "jenkins"

// DockerHub - dockerhub
const DockerHub string = "dockerhub"

// Bugzilla - bugzilla
const Bugzilla string = "bugzilla"

// BugzillaRest - bugzillarest (requires Bugzilla 5.X)
const BugzillaRest string = "bugzillarest"

// MeetUp - meetup
const MeetUp string = "meetup"

// RocketChat - rocketchat
const RocketChat string = "rocketchat"

// PyException - string that identified python exception
const PyException string = "Traceback (most recent call last)"

// DadsException - string that identified dads exception
const DadsException string = "DA_DS_ERROR(time="

// DadsWarning - string that identified dads exception
const DadsWarning string = "da-ds WARNING"

// BackendUser - backend-user
const BackendUser string = "backend-user"

// BackendPassword - backend-password
const BackendPassword string = "backend-password"

// User - user
const User string = "user"

// UserID - user-id
const UserID string = "user-id"

// Email - email
const Email string = "email"

// FromDate - from-date
const FromDate string = "from-date"

// Password - password
const Password string = "password"

// Redacted - [redacted]
const Redacted string = "[redacted]"

// SDSMtx - sdsmtx
const SDSMtx string = "sdsmtx"

// Locked - locked
const Locked string = "locked"

// Unlocked - unlocked
const Unlocked string = "unlocked"

// Bitergia - bitergia
const Bitergia string = "bitergia"

// External - external
const External string = "external"

// Delete - DELETE
const Delete string = "DELETE"

// Put - PUT
const Put string = "PUT"

// Get - GET
const Get string = "GET"

// Head - HEAD
const Head string = "HEAD"

// Post - POST
const Post string = "POST"

// SearchScroll - /_search/scroll
const SearchScroll string = "/_search/scroll"

// GitHubOrg - github_org
const GitHubOrg string = "github_org"

// GitHubUser - github_user
const GitHubUser string = "github_user"

// ProjectNoOrigin - special marker to set project on all index data
const ProjectNoOrigin string = "--no-origin--"

// Nil - used to specify an empty environment variable in the fixture (fo dads)
const Nil string = "<nil>"

// Null - used to specify a null value
const Null string = "(null)"

// DADS - config flag in the fixture that allows selecting when to run dads instead of p2o
const DADS string = "dads"

// ErrorStrings - array of possible errors returned from enrich tasks
var ErrorStrings = map[int]string{
	-3: "task was not executed due to frequency check",
	-2: "task is configured as a copy from another index pattern",
	-1: "task was skipped",
	1:  "datasource slug contains > 1 '/' separators",
	2:  "incorrect endpoint value for given data source",
	3:  "incorrect config option(s) for given data source",
	4:  "p2o.py error", // or da-ds error
	5:  "setting SSH private key error",
	6:  "command timeout error",
	7:  "index copy error",
}

// CopyFromDateField - field used to find most recent document and start copying from datetime from that field
const CopyFromDateField = "metadata__enriched_on" // Date when the item was enriched and stored in the index with enriched documents. (currently best IMHO - LG)
// const CopyFromDateField = "grimoire_creation_date" // Date when the item was created upstream. Used by default to represent data in time series on the dashboards.
// const CopyFromDateField = "metadata__timestamp"    // Date when the item was retrieved from the original data source and stored in the index with raw documents.
// const CopyFromDateField = "metadata__updated_on"   // Date when the item was updated in its original data source.

// GoogleGroups data source
const GoogleGroups string = "googlegroups"

// Gitlab data source
const Gitlab string = "gitlab"
