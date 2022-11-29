package utils

import (
	"context"
	"github.com/BRUHItsABunny/go-premiumize/api"
	premiumize_client "github.com/BRUHItsABunny/go-premiumize/client"
	"go.uber.org/atomic"
	"strings"
	"time"
)

type PDirectory struct {
	ID          *atomic.String
	Path        *atomic.String
	Prefix      *atomic.String
	Name        *atomic.String
	Directories map[string]*PDirectory // use thread unsafe maps, we fill it once and the keys nor the pointers should ever change after that
	Files       map[string]*PFile      // could switch to github.com/cornelk/hashmap if really needed, low write perf. should not be a deal-breaker for our use case since we write once then read only
	TotalSize   *atomic.Int64
	FileCount   *atomic.Int64
}

type PFile struct {
	ID      *atomic.String
	Path    *atomic.String
	Name    *atomic.String
	Size    *atomic.Int64
	Link    *atomic.String
	Created *atomic.Time
}

func (f *PFile) GetFullPath() string {
	return f.Path.Load() + "/" + f.Name.Load()
}

// RefreshLinks Refreshes links inside a directory recursively
func RefreshLinks(pClient *premiumize_client.PremiumizeClient, directory *PDirectory, recursive bool) error {
	listResp, err := pClient.FoldersList(context.Background(), &api.FolderListRequest{ID: directory.ID.Load()})
	if err != nil {
		return err
	}

	for _, item := range listResp.Content {
		if item.Type == "folder" && recursive {
			err = RefreshLinks(pClient, directory.Directories[item.ID], recursive)
			if err != nil {
				return err
			}
		} else {
			directory.Files[item.ID].Link.Store(*item.Link)
		}
	}

	return nil
}

// CrawlFilesystem crawls the directory we are syncing on the cloud's filesystem, collecting links and statistics while doing so, recursively?
func CrawlFilesystem(pClient *premiumize_client.PremiumizeClient, pathPrefix, directoryId string, recursive bool) *PDirectory {
	listResp, err := pClient.FoldersList(context.Background(), &api.FolderListRequest{ID: directoryId})
	if err != nil {
		return nil
	}
	result := &PDirectory{
		ID:          atomic.NewString(listResp.FolderID),
		Path:        atomic.NewString(""),
		Name:        atomic.NewString(listResp.Name),
		Directories: map[string]*PDirectory{},
		Files:       map[string]*PFile{},
		TotalSize:   atomic.NewInt64(0),
		FileCount:   atomic.NewInt64(0),
	}
	result.Prefix = atomic.NewString(pathPrefix)
	if len(pathPrefix) > 0 {
		pathPrefix += "/"
	}
	pathPrefix += listResp.Name
	result.Path = atomic.NewString(pathPrefix)
	for _, item := range listResp.Content {
		if item.Type == "folder" && recursive {
			result.Directories[item.ID] = CrawlFilesystem(pClient, pathPrefix, item.ID, recursive)
			result.TotalSize.Add(result.Directories[item.ID].TotalSize.Load())
			result.FileCount.Add(result.Directories[item.ID].FileCount.Load())
		} else {
			result.Files[item.ID] = &PFile{
				ID:      atomic.NewString(item.ID),
				Path:    atomic.NewString(result.Path.Load()),
				Name:    atomic.NewString(item.Name),
				Size:    atomic.NewInt64(int64(*item.Size)),
				Link:    atomic.NewString(*item.Link),
				Created: atomic.NewTime(time.Unix(int64(*item.CreatedAt), 0)),
			}
			result.FileCount.Inc()
			result.TotalSize.Add(result.Files[item.ID].Size.Load())
		}
	}

	return result
}

// LocateDirectory locates the directory on the cloud we want to sync to our local filesystem
func LocateDirectory(pClient *premiumize_client.PremiumizeClient, path string, recursive bool) *PDirectory {
	// Find the folder, check against name and id?
	offset := 0
	if strings.HasPrefix(path, "My Files/") || strings.HasPrefix(path, "/") {
		offset = 1
	}
	crumbs := strings.Split(path, "/")[offset:]
	folderID := "" // Start in root
	lastCrumb := len(crumbs) - 1
	for i, crumb := range crumbs {
		listResp, err := pClient.FoldersList(context.Background(), &api.FolderListRequest{ID: folderID})
		if err == nil {
			for _, item := range listResp.Content {
				if item.Type == "folder" {
					if item.Name == crumb {
						folderID = item.ID
						if i == lastCrumb {
							break
						}
						continue
					}
				}
			}
		} else {
			panic(err)
		}
	}

	folder := CrawlFilesystem(pClient, "", folderID, recursive)
	return folder
}
