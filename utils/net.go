package utils

import (
	"encoding/json"
	"fmt"
	"github.com/BRUHItsABunny/bunnlog"
	"go.uber.org/atomic"
	"io"
	"net/http"
	"os"
)

type NetUtil struct {
	Client *http.Client
	BLog   *bunnlog.BunnyLog
}

// GetCurrentIPAddress Get the current IP that the Go program uses via HTTPBin.org - for VPN confirmation
func (u *NetUtil) GetCurrentIPAddress() (ip string) {
	resp, err := u.Client.Get("https://httpbin.org/get")
	if err != nil {
		u.BLog.Warnf("Error getting IP: %s", fmt.Errorf("u.Client.Get: %w", err).Error())
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		u.BLog.Warnf("Error getting IP: %s", fmt.Errorf("io.ReadAll: %w", err).Error())
		return
	}

	_ = resp.Body.Close()
	data := map[string]any{}
	err = json.Unmarshal(bodyBytes, &data)

	if err != nil {
		u.BLog.Warnf("Error getting IP: %s", fmt.Errorf("json.Unmarshal: %w", err).Error())
		return
	}
	ipPreCast, ok := data["origin"]
	if ok {
		ip = ipPreCast.(string)
	} else {
		u.BLog.Warnf("Error getting IP: %s", string(bodyBytes))
	}
	return
}

type DownloadTask struct {
	FileName     *atomic.String `json:"fileName"`
	FileLocation *atomic.String `json:"fileLocation"`
	FileURL      *atomic.String `json:"fileURL"`
	FileSize     *atomic.Uint64 `json:"fileSize"`
	Downloaded   *atomic.Uint64 `json:"downloaded"`
	Delta        *atomic.Uint64 `json:"delta"`
}

func (t *DownloadTask) JSON() string {
	jsonBytes, err := json.Marshal(t)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

type DownloadGlobal struct {
	LastTick        *atomic.Time   `json:"lastTick"`
	CurrentIP       *atomic.String `json:"currentIP"`
	Directory       *atomic.String `json:"directory"`
	TotalFiles      *atomic.Int64  `json:"totalFiles"`
	TotalSize       *atomic.Uint64 `json:"totalSize"`
	DownloadedFiles *atomic.Int64  `json:"downloadedFiles"`
	Downloaded      *atomic.Uint64 `json:"downloaded"`
	Delta           *atomic.Uint64 `json:"delta"`
}

func (u *NetUtil) supportsRange(resp *http.Response) bool {
	supportsRanges := false
	if resp != nil {
		// u.BLog.Debugf("NetUtil.supportsRange: Response headers: %s", spew.Sdump(resp.Header))
		if resp.Request.Method == "HEAD" {
			if resp.Header.Get("Accept-Ranges") == "bytes" {
				supportsRanges = true
			}
			if resp.Header.Get("Ranges-Supported") == "bytes" {
				supportsRanges = true
			}
		} else {
			// GET?
			contentRange := resp.Header.Get("Content-Range")
			if contentRange != "" {
				supportsRanges = true
			}
		}
	}
	return supportsRanges
}

func (u *NetUtil) isResumable(fileURL string) (bool, error) {
	resp, err := u.Client.Head(fileURL)
	if err != nil {
		// Server doesn't support HEAD properly? Try a range GET with 1 byte
		req, _ := http.NewRequest(http.MethodGet, fileURL, nil)
		req.Header.Set("Range", "bytes=0-0")
		resp, err = u.Client.Do(req)
	}

	if err != nil {
		return false, err
	}
	return u.supportsRange(resp), nil
}

func (u *NetUtil) DownloadFile(global *DownloadGlobal, task *DownloadTask, f *os.File, notification chan struct{}) error {
	// Check if download is resumable, if so make request with byte range
	canResume, err := u.isResumable(task.FileURL.Load())
	if err != nil {
		err = fmt.Errorf("isResumable: %w", err)
		u.BLog.Warn(fmt.Sprintf("Not downloading because of error: %s", err.Error()))
		return err
	}

	stats, err := f.Stat()
	if err != nil {
		err = fmt.Errorf("f.Stat: %w", err)
		u.BLog.Warn(fmt.Sprintf("Not downloading because of error: %s", err.Error()))
		return err
	}
	task.Downloaded.Store(uint64(stats.Size()))
	global.Downloaded.Add(uint64(stats.Size()))

	req, _ := http.NewRequest(http.MethodGet, task.FileURL.Load(), nil)
	if canResume {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", task.Downloaded.Load(), task.FileSize.Load()))
	}

	resp, err := u.Client.Do(req)
	if err != nil {
		err = fmt.Errorf("hClient.Do: %w", err)
		u.BLog.Warn(fmt.Sprintf("Not downloading because of error: %s", err.Error()))
		return err
	}

	// Thread safe downloading and tracking
	var written int64
	for {
		written, err = io.CopyN(f, resp.Body, 1024)
		task.Downloaded.Add(uint64(written))
		task.Delta.Add(uint64(written))
		global.Downloaded.Add(uint64(written))
		global.Delta.Add(uint64(written))

		if task.Downloaded.Load() >= task.FileSize.Load() {
			u.BLog.Debug("Breaking because task is done downloading")
			err = nil
			_ = f.Close()
			break
		}
		if err != nil {
			err = fmt.Errorf("io.CopyN: %w", err)
			u.BLog.Warn(fmt.Sprintf("Breaking because of error: %s", err.Error()))
			break
		}
	}
	return err
}
