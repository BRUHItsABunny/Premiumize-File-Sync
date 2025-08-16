package utils

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BRUHItsABunny/bunnlog"
	"go.uber.org/atomic"
)

func BuildDirectoryTree(rootPath string) (*PDirectory, error) {
	abs, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("rootPath is not a directory")
	}
	return buildDir(abs)
}

func buildDir(dirPath string) (*PDirectory, error) {
	d := &PDirectory{
		ID:          atomic.NewString(dirPath),
		Path:        atomic.NewString(dirPath),
		Prefix:      atomic.NewString(""),
		Name:        atomic.NewString(filepath.Base(dirPath)),
		Directories: make(map[string]*PDirectory, 8),
		Files:       make(map[string]*PFile, 32),
		TotalSize:   atomic.NewInt64(0),
		FileCount:   atomic.NewInt64(0),
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		name := e.Name()
		full := filepath.Join(dirPath, name)

		if e.Type()&fs.ModeSymlink != 0 {
			continue
		}

		if e.IsDir() {
			child, err := buildDir(full)
			if err != nil {
				return nil, err
			}
			d.Directories[name] = child
			d.TotalSize.Add(child.TotalSize.Load())
			d.FileCount.Add(child.FileCount.Load())
			continue
		}

		fi, err := e.Info()
		if err != nil {
			continue
		}
		mode := fi.Mode()
		if mode.IsRegular() {
			size := fi.Size()
			pf := &PFile{
				ID:   atomic.NewString(full),
				Path: atomic.NewString(full),
				Name: atomic.NewString(name),
				Size: atomic.NewInt64(size),
			}
			d.Files[name] = pf
			d.TotalSize.Add(size)
			d.FileCount.Add(1)
		}
	}

	return d, nil
}

type SizeMismatch struct {
	Path       string
	LocalSize  int64
	RemoteSize int64
}

type DiffReport struct {
	// Files present locally but not in the matching remote directory.
	MissingInRemote []string
	// Files present in both but with different sizes.
	SizeMismatches []SizeMismatch

	// Stats
	MatchedCount int // files present in both with equal size
	CheckedCount int // files present in both (matched or mismatched)
}

// OK returns true if there are no missing files on remote and no size mismatches.
func (r DiffReport) OK() bool {
	return len(r.MissingInRemote) == 0 && len(r.SizeMismatches) == 0
}

// CompareLocalToRemote recursively enumerates all files from `local`,
// looking up counterparts (by name) in the parallel subtree under `remote`.
func CompareLocalToRemote(bLog *bunnlog.BunnyLog, local, remote *PDirectory, remove bool) DiffReport {
	var rep DiffReport

	var walk func(l *PDirectory, r *PDirectory, rel string)
	walk = func(l *PDirectory, r *PDirectory, rel string) {
		// Compare files in this directory (by base name key).
		for name, lf := range l.Files {
			relPath := filepath.Join(rel, name)
			if r == nil {
				rep.MissingInRemote = append(rep.MissingInRemote, relPath)
				continue
			}
			rf, ok := r.Files[name]
			if !ok || rf == nil {
				rep.MissingInRemote = append(rep.MissingInRemote, relPath)
				continue
			}

			rep.CheckedCount++
			ls := lf.Size.Load()
			rs := rf.Size.Load()
			if ls == rs {
				rep.MatchedCount++
			} else {
				rep.SizeMismatches = append(rep.SizeMismatches, SizeMismatch{
					Path: relPath, LocalSize: ls, RemoteSize: rs,
				})

				msg := fmt.Sprintf("Size mismatch: %s (local: %d vs remote: %d)", relPath, ls, rs)
				fmt.Println(msg)
				bLog.Warn(msg)
				if remove {
					os.Remove(relPath)
					bLog.Info("Removed")
				}
			}
		}

		// Recurse into subdirectories found locally.
		for name, lchild := range l.Directories {
			var rchild *PDirectory
			if r != nil {
				rchild = r.Directories[name]
			}
			walk(lchild, rchild, filepath.Join(rel, name))
		}
	}

	// Start at root with empty relative path for nice paths like "dir/file".
	walk(local, remote, remote.Name.Load())

	return rep
}
