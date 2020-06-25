package syncdatasources

// OK - common constant string
const OK string = "ok"

// APIToken - constant string
const APIToken string = "api-token"

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

// ErrorStrings - array of possible errors returned from enrich tasks
var ErrorStrings = map[int]string{
	-2: "task is configured as a copy from another index pattern",
	-1: "task was skipped",
	1:  "datasource slug contains > 1 '/' separators",
	2:  "incorrect endpoint value for given data source",
	3:  "incorrect config option(s) for given data source",
	4:  "p2o.py error",
	5:  "setting SSH private key error",
	6:  "command timeout error",
	7:  "index copy error",
}
