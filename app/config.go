package app

type Config struct {
	APIKey           string
	DownloadThreads  int
	Folder           string
	Recursive        bool
	ProgressInterval int
	Proxy            string
	Debug            bool
	Daemon           bool
	Version          bool
}
