package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/BRUHItsABunny/Premiumize-File-Sync/app"
	"github.com/BRUHItsABunny/Premiumize-File-Sync/utils"
	"github.com/BRUHItsABunny/bunterm"
	"github.com/BRUHItsABunny/gOkHttp/download"
	"github.com/dustin/go-humanize"
	"golang.org/x/sync/errgroup"
	"os"
	"sort"
	"time"
)

func downloadLoop(appData *app.App, dir *utils.PDirectory, workChan chan *download.ThreadedDownloadTask) {
	var (
		err error
	)
	if dir == nil {
		dir = appData.Directory
	}
	appData.BLog.Infof("DLLoop: Starting to download directory: %s", appData.Directory.Path.Load())

	files := []string{}
	for _, fObj := range dir.Files {
		files = append(files, fObj.Name.Load())
	}
	sort.Sort(sort.StringSlice(files))

	for i := 0; i < len(files); i++ {
		if appData.Stats.GraceFulStop.Load() {
			appData.BLog.Debug("DLLoop stopping - graceful stop")
			break
		}

		if appData.Stats.Tasks.Len() >= appData.Cfg.DownloadThreads {
			appData.BLog.Debug("DLLoop: Sending nil task")
			// Don't spam tracker with tasks we don't actively run
			workChan <- nil
			appData.BLog.Debug("DLLoop: Sent nil task")
			i--
		} else {
			file := dir.Files[files[i]]
			appData.BLog.Infof("DLLoop: Preparing task: %s", file.Name.Load())
			task, err := download.NewThreadedDownloadTask(context.Background(), appData.DownloadClient, appData.Stats, file.GetFullPath(), file.Link.Load(), 1, uint64(file.Size.Load())) //requests.NewHeaderOption(http.Header{"Accept-Encoding": []string{"identity"}})
			if err != nil {
				err = fmt.Errorf("download.NewThreadedDownloadTask: %w", err)
				appData.BLog.Error("DLLoop: Failed to prepare task: %s", err.Error())
				break
			}
			appData.Stats.TotalFiles.Dec()
			appData.Stats.TotalBytes.Sub(task.TaskStats.FileSize.Load())
			appData.BLog.Infof("DLLoop: Sending task: %s", file.Name.Load())
			workChan <- task
			appData.BLog.Infof("DLLoop: Sent task: %s", file.Name.Load())
		}

	}

	if err == nil {
		subDirs := []string{}
		for subDirLocation, _ := range dir.Directories {
			subDirs = append(subDirs, subDirLocation)
		}
		sort.Strings(subDirs)

		for _, key := range subDirs {
			if appData.Stats.GraceFulStop.Load() {
				appData.BLog.Debug("DLLoop stopping (recursion) - graceful stop")
				break
			}
			downloadLoop(appData, dir.Directories[key], workChan)
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
	appData.Stats.TotalFiles.Store(uint64(appData.Directory.FileCount.Load()))
	appData.Stats.TotalBytes.Store(uint64(appData.Directory.TotalSize.Load()))
	appData.BLog.Infof("Crawled dir: %s with a total of %d files found (%s)", appData.Directory.Name.Load(), appData.Directory.FileCount.Load(), humanize.Bytes(uint64(appData.Directory.TotalSize.Load())))

	// UI
	go func() {
		appData.BLog.Debug("Starting the UI thread")
		fmt.Println(appData.Stats.Tick(true))
		term := bunterm.DefaultTerminal
		for {
			if appData.Stats.GraceFulStop.Load() || appData.Stats.IdleTimeoutExceeded() {
				if appData.Stats.GraceFulStop.Load() {
					appData.BLog.Info(fmt.Sprintf("[UI] - Graceful stop"))
				} else {
					appData.BLog.Info(fmt.Sprintf("[UI] - Time out stop"))
				}
				appData.Stats.Stop()
				break
			}
			if !appData.Cfg.Daemon {
				// Human-readable means we clear the spam
				term.ClearTerminal()
				term.MoveCursor(0, 0)
			}
			fmt.Println(appData.Stats.Tick(true))
			time.Sleep(time.Second)
		}
		appData.BLog.Debug("Stopping the UI thread")
	}()

	errGr, ctx := errgroup.WithContext(context.Background())
	workChan := make(chan *download.ThreadedDownloadTask, appData.Cfg.DownloadThreads-1)
	for i := 1; i <= appData.Cfg.DownloadThreads; i++ {
		threadId := i
		errGr.Go(func() error {
			return Worker(ctx, threadId, workChan, appData)
		})
	}

	appData.BLog.Debugf("Going to start download loop")
	go downloadLoop(appData, nil, workChan)
	err = errGr.Wait()
	if err != nil {
		appData.BLog.Error(err)
		<-workChan // unlock looper
	}
	appData.BLog.Info("Waiting for all threads to end")
	appData.Stats.Stop()
	appData.BLog.Info("Stopping program")
}

func Worker(ctx context.Context, threadId int, workChan chan *download.ThreadedDownloadTask, appData *app.App) error {
	appData.BLog.Debug(fmt.Sprintf("[thread:%d] Worker starting", threadId))
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			break
		case task := <-workChan:
			if task == nil {
				continue
			} else {
				appData.BLog.Debug(fmt.Sprintf("[thread:%d] Worker downloading: %s", threadId, task.FileLocation.Load()))
				// Blocking
				err := task.Download(ctx)
				if err != nil {
					appData.BLog.Error(fmt.Sprintf("[thread:%d] Worker stopping: %s", threadId, err.Error()))
					appData.BLog.Debug(fmt.Sprintf("[thread:%d] Task: %s", threadId, TaskJSON(task)))
					return fmt.Errorf("[thread:%d] task.Download: %w", threadId, err)
				}
			}
			break
		}
		if appData.Stats.GraceFulStop.Load() {
			appData.BLog.Info(fmt.Sprintf("[thread:%d] Worker stopping - Graceful stop", threadId))
			break
		}
	}
	appData.BLog.Debug(fmt.Sprintf("[thread:%d] Worker stopping", threadId))
	return nil
}

func TaskJSON(task *download.ThreadedDownloadTask) string {
	jsonBytes, err := json.Marshal(task)
	if err != nil {
		return ""
	}

	return string(jsonBytes)
}
