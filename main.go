package main

import (
	"fmt"
	"github.com/BRUHItsABunny/Premiumize-File-Sync/app"
	"github.com/BRUHItsABunny/Premiumize-File-Sync/utils"
	"github.com/BRUHItsABunny/bunterm"
	"github.com/dustin/go-humanize"
	"go.uber.org/atomic"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func downloadLoop(appData *app.App, dir *utils.PDirectory, notification chan struct{}) {
	var (
		f   *os.File
		err error
	)
	if dir == nil {
		dir = appData.Directory
	}
	appData.Stats.Global.Directory.Store(appData.Directory.Path.Load())
	appData.BLog.Infof("Starting to download directory: %s", appData.Directory.Path.Load())

	files := []string{}
	for _, fObj := range dir.Files {
		files = append(files, fObj.Name.Load())
	}
	sort.Sort(sort.StringSlice(files))

	for _, fileKey := range files {
		// Wait for space in queue
		appData.BLog.Debug("Waiting for notification...")
		<-notification
		file := dir.Files[fileKey]
		appData.BLog.Infof("Starting to download file: %s", file.Name.Load())
		task := &utils.DownloadTask{
			FileName:     atomic.NewString(file.Name.Load()),
			FileLocation: atomic.NewString(file.GetFullPath()),
			FileURL:      atomic.NewString(file.Link.Load()),
			FileSize:     atomic.NewUint64(uint64(file.Size.Load())),
			Downloaded:   atomic.NewUint64(0),
			Delta:        atomic.NewUint64(0),
		}
		appData.BLog.Debugf("task: %s", task.JSON())

		dName := filepath.Dir(task.FileLocation.Load())
		appData.BLog.Debugf("Making directory: %s", dName)
		err = os.MkdirAll(dName, 0600)
		if err != nil {
			err = fmt.Errorf("os.MkdirAll: %w", err)
			appData.BLog.Fatalf("Failed to create dir: %s", err.Error())
			break
		}
		appData.BLog.Infof("Made directory: %s", dName)

		f, err = os.OpenFile(task.FileLocation.Load(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil || f == nil {
			err = fmt.Errorf("os.OpenFile: %w", err)
			appData.BLog.Fatalf("Failed to open or create file: %s", err.Error())
			break
		}

		appData.Stats.Tasks.Set(task.FileLocation.Load(), task)
		go appData.DownloadClient.DownloadFile(appData.Stats.Global, task, f, notification)
		appData.BLog.Infof("Started downloading thread url: %s", task.FileURL.Load())
	}

	if err == nil {
		subDirs := []string{}
		for subDirLocation, _ := range dir.Directories {
			subDirs = append(subDirs, subDirLocation)
		}
		sort.Strings(sort.StringSlice(subDirs))

		for _, key := range subDirs {
			downloadLoop(appData, dir.Directories[key], notification)
		}
	}
}

func main() {
	appData, err := app.NewApp()
	if err != nil {
		panic(err)
	}

	versionOutput := appData.VersionRoutine()
	if appData.Cfg.Version {
		fmt.Println(versionOutput)
		os.Exit(0)
	}
	appData.BLog.Debug(versionOutput)

	appData.Directory = utils.LocateDirectory(appData.Client, appData.Cfg.Folder, appData.Cfg.Recursive)
	if appData.Directory == nil {
		// can't get dir
		panic("dir is nil")
	}
	appData.Stats.Global.TotalFiles.Store(appData.Directory.FileCount.Load())
	appData.Stats.Global.TotalSize.Store(uint64(appData.Directory.TotalSize.Load()))
	appData.BLog.Infof("Crawled dir: %s with a total of %d files found (%s)", appData.Directory.Name.Load(), appData.Directory.FileCount.Load(), humanize.Bytes(uint64(appData.Directory.TotalSize.Load())))

	notification := make(chan struct{}, appData.Cfg.DownloadThreads)
	for j := 0; j < appData.Cfg.DownloadThreads; j++ {
		notification <- struct{}{}
	}

	// UI
	go func() {
		appData.BLog.Debug("Starting the UI thread")
		term := bunterm.DefaultTerminal
		continueLoop := true
		i := 0
		for continueLoop {
			i++
			if i >= 60 {
				i = 1
				ip := appData.DownloadClient.GetCurrentIPAddress()
				if len(ip) > 0 {
					appData.Stats.Global.CurrentIP.Store(ip)
				}
				appData.BLog.Debugf("Refreshed IP address: %s", appData.Stats.Global.CurrentIP.Load())
			}
			if !appData.Cfg.Daemon {
				// Human-readable means we clear the spam
				term.ClearTerminal()
				term.MoveCursor(0, 0)
			}
			fmt.Println(appData.Stats.Tick(!appData.Cfg.Daemon, notification))

			if appData.Stats.Global.DownloadedFiles.Load() >= appData.Stats.Global.TotalFiles.Load() {
				continueLoop = false
			}

			time.Sleep(time.Second)
		}
		appData.BLog.Debug("Stopping the UI thread")
	}()

	appData.BLog.Debugf("Going to start download loop with %d threads", appData.Cfg.DownloadThreads)
	downloadLoop(appData, nil, notification)
}
