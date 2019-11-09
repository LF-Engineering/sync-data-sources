package syncdatasources

// OK - common constant string
const OK string = "ok"

// Git - git
const Git string = "git"

// GitHub - github
const GitHub string = "github"

// ErrorStrings - array of possible errors returned from enrich tasks
var ErrorStrings = map[int]string{
	1: "datasource slug contains > 1 '/' separators",
	2: "incorrect endpoint value for given data source",
	3: "incorrect config option(s) for given data source",
}
