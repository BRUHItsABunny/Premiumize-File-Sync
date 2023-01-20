package app

import (
	"context"
	"flag"
	"fmt"
	"github.com/BRUHItsABunny/Premiumize-File-Sync/utils"
	"github.com/BRUHItsABunny/bunnlog"
	"github.com/BRUHItsABunny/gOkHttp/client"
	"github.com/BRUHItsABunny/gOkHttp/download"
	"github.com/BRUHItsABunny/go-premiumize/api"
	premiumize_client "github.com/BRUHItsABunny/go-premiumize/client"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type App struct {
	Cfg            *Config
	Client         *premiumize_client.PremiumizeClient
	DownloadClient *http.Client
	BLog           *bunnlog.BunnyLog
	Stats          *download.GlobalDownloadTracker
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

	app.Stats = download.NewGlobalDownloadTracker(time.Duration(3) * time.Second)
	app.Stats.PollIP(app.DownloadClient)

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
	opts := []client.Option{}
	if len(a.Cfg.Proxy) > 0 {
		opts = append(opts, client.NewProxyOption(a.Cfg.Proxy))
	}

	var err error
	a.DownloadClient, err = client.NewHTTPClient(opts...)
	if err != nil {
		return fmt.Errorf("client.NewHTTPClient: %w", err)
	}
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
	a.Client = premiumize_client.NewPremiumizeClient(session, a.DownloadClient)
	return nil
}

func (a *App) VersionRoutine() string {
	result := strings.Builder{}
	currentPrompt := CurrentCodeBase.PromptCurrentVersion(CurrentVersion)
	latestVersion, err := CurrentCodeBase.GetLatestVersion(context.Background(), nil)
	if err != nil {
		panic(fmt.Errorf("CurrentCodeBase.GetLatestVersion: %w", err))
	}
	isOutdated, latestPrompt := CurrentCodeBase.PromptLatestVersion(CurrentVersion, latestVersion)
	result.WriteString(currentPrompt.Output)
	if isOutdated {
		result.WriteString("\n")
		result.WriteString(latestPrompt.Output)
		result.WriteString(fmt.Sprintf("You can find more here:\n%s\n", latestPrompt.UpdateURL))
	}
	return result.String()
}
