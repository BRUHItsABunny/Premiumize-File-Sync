package app

import (
	"encoding/json"
	"fmt"
	"github.com/BRUHItsABunny/Premiumize-File-Sync/utils"
	"github.com/cornelk/hashmap"
	"github.com/dustin/go-humanize"
	"go.uber.org/atomic"
	"math"
	"strings"
	"time"
)

// channel vs atomic
// channel: aggregation, many structs, per download notifications
// atomic: less structs, only global notifs?

type Statistics struct {
	Global *utils.DownloadGlobal                     `json:"global"`
	Tasks  *hashmap.Map[string, *utils.DownloadTask] `json:"tasks"`
	// DoneTasks *hashmap.Map[string, *utils.DownloadTask] `json:"doneTasks"`
}

func NewStatistics() *Statistics {
	return &Statistics{
		Global: &utils.DownloadGlobal{
			LastTick:        atomic.NewTime(time.Now()),
			Directory:       atomic.NewString(""),
			TotalFiles:      atomic.NewInt64(0),
			DownloadedFiles: atomic.NewInt64(0),
			TotalSize:       atomic.NewUint64(0),
			Downloaded:      atomic.NewUint64(0),
			Delta:           atomic.NewUint64(0),
			CurrentIP:       atomic.NewString(utils.GetCurrentIPAddress()),
		},
		Tasks: hashmap.New[string, *utils.DownloadTask](),
		// DoneTasks: hashmap.New[string, *utils.DownloadTask](),
	}
}

func (s *Statistics) Tick(humanReadable bool, notification chan struct{}) string {
	result := &strings.Builder{}
	if humanReadable {
		s.fmtReadable(result)
	} else {
		s.fmtNonReadable(result)
	}

	s.Global.Delta.Store(0)
	s.Tasks.Range(func(k string, v *utils.DownloadTask) bool {
		v.Delta.Store(0)
		if v.Downloaded.Load() >= v.FileSize.Load()-1 {
			s.Tasks.Del(k)
			// s.DoneTasks.Set(v.FileLocation.Load(), v)
			s.Global.DownloadedFiles.Inc()
			notification <- struct{}{}
		}
		return true
	})
	s.Global.LastTick.Store(time.Now())
	return result.String()
}

func (s *Statistics) fmtNonReadable(result *strings.Builder) {
	/*
		Backend CLI

		{total: {}, tasks: {taskN: {}}}
	*/
	jsonBytes, _ := json.Marshal(s)
	result.Write(jsonBytes)
}

func (s *Statistics) fmtReadable(result *strings.Builder) {
	/*
		Live CLI

		Current IP: $IP and TIME: $DATETIME
		Download statistics: $TOTAL_DONE ($TOTAL_DONE_SIZE) out of $TOTAL ($TOTAL_SIZE)
		[######################################] ($TOTAL_DONE_PERCENT% at $TOTAL_SPEED/s, ETA: $TOTAL_TIME_LEFT)

		$FILENAME
		Download statistic: $TOTAL_DONE ($TOTAL_DONE_SIZE) out of $TOTAL ($TOTAL_SIZE)
		[######################################] ($TOTAL_DONE_PERCENT% at $TOTAL_SPEED/s, ETA: $TOTAL_TIME_LEFT)
	*/
	result.WriteString(fmt.Sprintf("Current IP: %s and last tick %s\nCurrent directory: %s\n", s.Global.CurrentIP.Load(), s.Global.LastTick.Load().Format(time.RFC3339), s.Global.Directory.Load()))
	result.WriteString(fmt.Sprintf("Download statistics: %d (%s) out of %d (%s)\n", s.Global.DownloadedFiles.Load(), humanize.Bytes(s.Global.Downloaded.Load()), s.Global.TotalFiles.Load(), humanize.Bytes(s.Global.TotalSize.Load())))
	progress(result, s.Global.Delta.Load(), s.Global.Downloaded.Load(), s.Global.TotalSize.Load(), 40)
	result.WriteString(fmt.Sprintf("Downloading %d files:\n", s.Tasks.Len()))

	s.Tasks.Range(func(k string, v *utils.DownloadTask) bool {
		result.WriteString(fmt.Sprintf("%s\n", utils.Truncate(v.FileLocation.Load(), 128, 0)))
		result.WriteString(fmt.Sprintf("Download statistic: %s out of %s\n", humanize.Bytes(v.Downloaded.Load()), humanize.Bytes(v.FileSize.Load())))
		progress(result, v.Delta.Load(), v.Downloaded.Load(), v.FileSize.Load(), 40)
		return true
	})
}

func progress(result *strings.Builder, delta, downloaded, totalSize uint64, barSize int) {
	result.WriteString("[")
	percentage := float64(downloaded) / float64(totalSize)
	pieces := int(math.Floor(float64(barSize) * percentage))
	for i := 0; i < pieces; i++ {
		result.WriteString("X")
	}
	for i := 0; i < barSize-pieces; i++ {
		result.WriteString(" ")
	}
	result.WriteString("]")
	etaSecs := math.Ceil(float64(totalSize-downloaded) / float64(delta))
	eta := time.Second * time.Duration(etaSecs)
	result.WriteString(fmt.Sprintf(" (%.2f%% at %s/s, ETA: %s)\n", percentage*100, humanize.Bytes(delta), eta.String()))
}
