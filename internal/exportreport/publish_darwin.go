package exportreport

import "golang.org/x/sys/unix"

func publishDirectory(parent int, from, to string) error {
	return unix.RenameatxNp(parent, from, parent, to, unix.RENAME_EXCL)
}
