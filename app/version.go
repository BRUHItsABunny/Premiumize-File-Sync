package app

const none string = ""

// `-ldflags "-X app.appVersion=0.0.1"`
var (
	appVersion = "v0.0.1"
	buildTime  = none
	gitCommit  = none
	gitRef     = none
	gitRepo    = "https://github.com/BRUHItsABunny/Premiumize-File-Sync/"
)
