// Package readiness observes selected memory-integration metadata without
// executing native programs, contacting providers or changing product state.
package readiness

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode"
)

var (
	errMetadataUnsafe  = errors.New("metadata path or file type is unsafe")
	errMetadataLimit   = errors.New("metadata exceeds the read limit")
	errMetadataChanged = errors.New("metadata changed during inspection")
)

// readMetadata returns all bounded bytes or none. It does not decode content or
// expose it in a report. Callers must map I/O errors to fixed public findings.
// Cancellation is checked between reads; it cannot interrupt a blocked kernel
// filesystem operation and does not launch a goroutine that could outlive it.
func readMetadata(ctx context.Context, path string, limit int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 4<<20 {
		return nil, errMetadataLimit
	}
	if !filepath.IsAbs(path) || len(path) > 4096 || strings.IndexFunc(path, unicode.IsControl) >= 0 {
		return nil, errMetadataUnsafe
	}
	path = filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	if resolved != path {
		return nil, errMetadataUnsafe
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, errMetadataUnsafe
	}
	if before.Size() > limit {
		return nil, errMetadataLimit
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Supported targets are Darwin/Linux. Nonblocking + no-follow protects the
	// final component from being substituted with a blocking FIFO or symlink.
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, errMetadataUnsafe
		}
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() {
		return nil, errMetadataUnsafe
	}
	if !sameMetadata(before, opened) {
		return nil, errMetadataChanged
	}
	reader := io.LimitReader(f, limit+1)
	buffer := make([]byte, 32<<10)
	data := make([]byte, 0, min(opened.Size(), limit))
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		n, err := reader.Read(buffer)
		data = append(data, buffer[:n]...)
		if int64(len(data)) > limit {
			return nil, errMetadataLimit
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	current, err := os.Lstat(path)
	if err != nil || !sameMetadata(opened, after) || !sameMetadata(after, current) || int64(len(data)) != after.Size() {
		return nil, errMetadataChanged
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

func sameMetadata(a, b os.FileInfo) bool {
	return a != nil && b != nil && b.Mode().IsRegular() && os.SameFile(a, b) &&
		a.Size() == b.Size() && a.Mode() == b.Mode() && a.ModTime().Equal(b.ModTime())
}
