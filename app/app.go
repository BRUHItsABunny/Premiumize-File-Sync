package app

import (
	"context"
	"flag"
	"fmt"
	"github.com/BRUHItsABunny/Premiumize-File-Sync/utils"
	"github.com/BRUHItsABunny/bunnlog"
	"github.com/BRUHItsABunny/go-premiumize/api"
	premiumize_client "github.com/BRUHItsABunny/go-premiumize/client"
	"github.com/google/go-github/v48/github"
	"github.com/hashicorp/go-version"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type App struct {
	Cfg            *Config
	Client         *premiumize_client.PremiumizeClient
	DownloadClient *utils.NetUtil
	BLog           *bunnlog.BunnyLog
	Stats          *Statistics
	Directory      *utils.PDirectory
}

func NewApp() (*App, error) {
	app := &App{}

	// CLI args to struct
	app.ParseCfg()

	// Logger
	err := app.SetupLogger()
	if err != nil {
		return nil, err
	}

	// HTTP client
	err = app.SetupHTTPClient()
	if err != nil {
		return nil, err
	}

	// Premiumize client
	err = app.SetupPremiumizeClient()
	if err != nil {
		return nil, err
	}

	app.Stats = NewStatistics(app.DownloadClient.GetCurrentIPAddress())

	return app, err
}

func (a *App) ParseCfg() {
	if a.Cfg == nil {
		a.Cfg = &Config{}
	}

	flag.StringVar(&a.Cfg.APIKey, "apikey", "", "This is our APIKey - not needed and can also be set as env variable PREMIUMIZE_API_KEY, if missing it will authenticate via device code")
	flag.IntVar(&a.Cfg.DownloadThreads, "threads", 1, "This is how many files we download in parallel (min=1, max=6)")
	flag.StringVar(&a.Cfg.Folder, "folder", "", "This is the folder we will start crawling in")
	flag.BoolVar(&a.Cfg.Recursive, "recursion", false, "This controls if we want all files inside all folders of the folder you selected or just all files in the folder you selected")
	flag.IntVar(&a.Cfg.ProgressInterval, "pinterval", 5, "This is how many seconds we wait in between each progress print")
	flag.StringVar(&a.Cfg.Proxy, "proxy", "", "This argument is for proxying this program (format: proto://ip:port)")
	flag.BoolVar(&a.Cfg.Debug, "debug", false, "This argument is for how verbose the logger will be")
	flag.BoolVar(&a.Cfg.Daemon, "daemon", false, "This argument is for how the UI feedback will be, if set to true it will print JSON")
	flag.BoolVar(&a.Cfg.Version, "version", false, "This argument will print the current version data and exit")
	flag.Parse()

	if a.Cfg.DownloadThreads > 6 {
		a.Cfg.DownloadThreads = 6
	}
	if a.Cfg.DownloadThreads < 1 {
		a.Cfg.DownloadThreads = 1
	}
}

func (a *App) SetupLogger() error {
	logFile, err := os.Create("premiumize-file-sync.log")
	if err != nil {
		return err
	}
	var bLog bunnlog.BunnyLog
	if a.Cfg.Debug {
		bLog = bunnlog.GetBunnLog(true, bunnlog.VerbosityDEBUG, log.Ldate|log.Ltime)
	} else {
		bLog = bunnlog.GetBunnLog(false, bunnlog.VerbosityWARNING, log.Ldate|log.Ltime)
	}
	bLog.SetOutputFile(logFile)
	a.BLog = &bLog
	return nil
}

func (a *App) SetupHTTPClient() error {
	trans := http.Transport{}
	if len(a.Cfg.Proxy) > 0 {
		a.BLog.Debugf("Trying to parse proxy: %s", a.Cfg.Proxy)
		puo, err := url.Parse(a.Cfg.Proxy)
		if err != nil {
			// Fatal indeed, never again
			a.BLog.Fatalf("Failed to parse proxy: %s", a.Cfg.Proxy)
			return err
		}
		trans.Proxy = http.ProxyURL(puo)
	}

	a.DownloadClient = &utils.NetUtil{Client: &http.Client{Transport: &trans}, BLog: a.BLog}
	return nil
}

func (a *App) SetupPremiumizeClient() error {
	var session *api.PremiumizeSession
	if len(a.Cfg.APIKey) == 0 {
		a.Cfg.APIKey = os.Getenv("PREMIUMIZE_API_KEY")
		if len(a.Cfg.APIKey) > 8 {
			a.BLog.Infof("Using API Key from ENV variables: %s", utils.Censor(a.Cfg.APIKey, "*", 6, true))
		}
	}
	if len(a.Cfg.APIKey) > 0 {
		session = &api.PremiumizeSession{SessionType: "apikey", AuthToken: a.Cfg.APIKey}
	}
	a.Client = premiumize_client.NewPremiumizeClient(session, a.DownloadClient.Client)
	return nil
}

func (a *App) VersionRoutine() string {
	latestVersion := appVersion
	client := github.NewClient(nil)

	// list all organizations for user "willnorris"
	urLsplitter := strings.Split(gitRepo, "/")
	tags, _, err := client.Repositories.ListTags(context.Background(), urLsplitter[len(urLsplitter)-3], urLsplitter[len(urLsplitter)-2], nil)
	if len(tags) > 0 {
		latestTag := tags[0]
		if latestTag != nil && latestTag.Name != nil {
			latestTagVersion := *latestTag.Name
			v1, _ := version.NewVersion(appVersion[0:])
			v2, _ := version.NewVersion(latestTagVersion[0:])
			if v1.LessThan(v2) {
				latestVersion = latestTagVersion
			}
		}
	}
	result := fmt.Sprintf("Version:\t%s\nBuild Time:\t%s\nGit Commit:\t%s\nGit Ref:\t%s\n",
		appVersion, buildTime, gitCommit, gitRef)
	if latestVersion != appVersion {
		latestVersionURL := gitRepo + "releases/tag/" + latestVersion
		result += fmt.Sprintf("There is a newer version %s to find out more:\n%s\n", latestVersion, latestVersionURL)
	}
	if err != nil {
		result += fmt.Sprintf("Error checkking for new version: %s", err.Error())
	}
	return result
}
