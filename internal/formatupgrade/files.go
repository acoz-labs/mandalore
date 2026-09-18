package formatupgrade

import (
	"errors"
	"io"
	"os"
	"path"

	"github.com/acoz-labs/mandalore/internal/memory"
)

const stage = ".mandalore/format-upgrade"
const preparedLimit = 8 << 20

func readFile(root *os.Root, name string, limit int64) ([]byte, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() > limit {
		return nil, ErrApply
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return nil, ErrApply
	}
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, ErrApply
	}
	return b, nil
}

func syncPath(root *os.Root, name string) error {
	f, err := root.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func privateFile(root *os.Root, name string, data []byte) error {
	if len(data) > preparedLimit {
		return ErrApply
	}
	f, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return syncPath(root, path.Dir(name))
}

// Retain all owned preparation files on failure. Linking publishes complete,
// fsynced bytes exclusively; the caller separately syncs the destination parent
// and reports publication versus confirmed durability without blind retries.
func publish(root *os.Root, name string, data []byte) error {
	tmp := stage + "/.prepare-" + memory.NewID("file")
	if err := privateFile(root, tmp, data); err != nil {
		return err
	}
	return root.Link(tmp, name)
}

func ensureDirectory(root *os.Root, name string) error {
	if err := root.Mkdir(name, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	info, err := root.Lstat(name)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ErrApply
	}
	return syncPath(root, path.Dir(name))
}
