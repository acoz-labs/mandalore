package exportreport

import "golang.org/x/sys/unix"

func publishDirectory(parent int, from, to string) error {
	return unix.Renameat2(parent, from, parent, to, unix.RENAME_NOREPLACE)
}
