package app

type Config struct {
	APIKey          string
	DownloadThreads int
	Folder          string
	Recursive       bool
	ProgressTimeOut int
	Proxy           string
	Debug           bool
	Daemon          bool
	Version         bool
}
