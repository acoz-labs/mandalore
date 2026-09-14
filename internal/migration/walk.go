package migration

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// Read directory entries in bounded batches. filepath.WalkDir would allocate
// an entire oversized directory before the census could enforce its file cap.
// Snapshot files are sorted separately for deterministic validation and hashing.
func walkPortable(root string, visit fs.WalkDirFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		return err
	}
	var walk func(string, fs.DirEntry) error
	walk = func(path string, entry fs.DirEntry) error {
		if err := visit(path, entry, nil); err != nil {
			if err == filepath.SkipDir && entry.IsDir() {
				return nil
			}
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return err
		}
		f := os.NewFile(uintptr(fd), path)
		defer f.Close()
		for {
			entries, readErr := f.ReadDir(128)
			for _, child := range entries {
				if err := walk(filepath.Join(path, child.Name()), child); err != nil {
					return err
				}
			}
			if readErr == io.EOF {
				return nil
			}
			if readErr != nil {
				return readErr
			}
		}
	}
	return walk(root, fs.FileInfoToDirEntry(info))
}
