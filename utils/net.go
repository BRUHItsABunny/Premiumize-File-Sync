package utils

import (
	"encoding/json"
	"fmt"
	"go.uber.org/atomic"
	"io"
	"net/http"
	"os"
)

// GetCurrentIPAddress Get the current IP that the Go program uses via HTTPBin.org - for VPN confirmation
func GetCurrentIPAddress() (ip string) {
	resp, err := http.Get("https://httpbin.org/get")
	defer resp.Body.Close()

	if err != nil {
		ip = "not connected"
		return
	}
	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		ip = "not connected"
		return
	}

	data := map[string]any{}
	err = json.Unmarshal(bodyBytes, &data)

	if err != nil {
		ip = "not connected"
		return
	}

	ip = data["origin"].(string)
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

// TODO: Write the downloader func

func supportsRange(resp *http.Response) bool {
	supportsRanges := false
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
	return supportsRanges
}

func isResumable(hClient *http.Client, fileURL string) (bool, error) {
	resp, err := hClient.Head(fileURL)
	if err != nil {
		// Server doesn't support HEAD properly? Try a range GET with 1 byte
		req, _ := http.NewRequest(http.MethodGet, fileURL, nil)
		req.Header.Set("Range", "bytes=0-0")
		resp, err = hClient.Do(req)
	}

	if err != nil {
		return false, err
	}
	return supportsRange(resp), nil
}

func DownloadFile(hClient *http.Client, global *DownloadGlobal, task *DownloadTask, f *os.File, notification chan struct{}) error {
	// Check if download is resumable, if so make request with byte range
	canResume, err := isResumable(hClient, task.FileURL.Load())
	if err != nil {
		return fmt.Errorf("isResumable: %w", err)
	}

	stats, err := f.Stat()
	if err != nil {
		return fmt.Errorf("f.Stat: %w", err)
	}
	task.Downloaded.Store(uint64(stats.Size()))
	global.Downloaded.Add(uint64(stats.Size()))

	req, _ := http.NewRequest(http.MethodGet, task.FileURL.Load(), nil)
	if canResume {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", task.Downloaded.Load(), task.FileSize.Load()))
	}

	resp, err := hClient.Do(req)
	if err != nil {
		return fmt.Errorf("hClient.Do: %w", err)
	}

	// Thread safe downloading and tracking
	var written int64
	for {
		written, err = io.CopyN(f, resp.Body, 1024)
		task.Downloaded.Add(uint64(written))
		task.Delta.Add(uint64(written))
		global.Downloaded.Add(uint64(written))
		global.Delta.Add(uint64(written))

		if task.Downloaded.Load() >= task.FileSize.Load()-1 {
			err = nil
			_ = f.Close()
			break
		}
		if err != nil {
			err = fmt.Errorf("io.CopyN: %w", err)
			break
		}
	}
	return err
}
