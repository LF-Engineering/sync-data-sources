package syncdatasources

// OK - common constant string
const OK string = "ok"

// ErrorStrings - array of possible errors returned from enrich tasks
var ErrorStrings = map[int]string{
	1: "datasource slug contains > 1 '/' separators",
	2: "incorrect endpoint value for given data source",
}
