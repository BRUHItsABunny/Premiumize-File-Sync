package app

const none string = ""

// `-ldflags "-X app.appVersion=0.0.1"`
var (
	appVersion = none
	buildTime  = none
	gitCommit  = none
	gitRef     = none
)
