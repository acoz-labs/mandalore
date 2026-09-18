package exportreport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/acoz-labs/mandalore/internal/memory"
	"golang.org/x/sys/unix"
)

type Receipt struct {
	Phase            string `json:"phase"`
	StagingDirectory string `json:"staging_directory,omitempty"`
	Destination      string `json:"destination"`
	ReportPath       string `json:"report_path,omitempty"`
	BytesWritten     int    `json:"bytes_written"`
	Published        bool   `json:"published"`
	Durable          bool   `json:"durable"`
	Notice           string `json:"notice"`
}

func Apply(ctx context.Context, reviewed Plan) (Receipt, error) {
	return apply(ctx, reviewed, nil)
}

// phaseHook is a package-local fault seam, never supplied through an interface.
func apply(ctx context.Context, reviewed Plan, phaseHook func(string) error) (receipt Receipt, err error) {
	receipt = Receipt{Phase: "validation", Notice: "Only this selected report is written. Other exports and source history are not inspected or removed. On failure retain and inspect named paths; no automatic cleanup or retry."}
	defer func() {
		if err != nil {
			if ctx.Err() != nil {
				err = ctx.Err()
			} else {
				err = errors.New("export did not complete; inspect its receipt and preserve any partial output before preparing a new plan")
			}
		}
	}()
	if err = ctx.Err(); err != nil {
		return
	}
	encoded, e := json.Marshal(reviewed)
	if e != nil || len(encoded) > 24<<10 {
		err = errPreview
		return
	}
	current, data, e := prepare(ctx, reviewed.Request)
	if e != nil {
		err = e
		return
	}
	if !reflect.DeepEqual(reviewed, current) {
		err = errPreview
		return
	}
	receipt.Destination = current.Request.Destination
	parent, e := os.Open(filepath.Dir(receipt.Destination))
	if e != nil {
		err = e
		return
	}
	defer parent.Close()
	parentFD := int(parent.Fd())
	var parentStat unix.Stat_t
	if err = unix.Fstat(parentFD, &parentStat); err != nil {
		return
	}
	if fmt.Sprintf("%d:%d", parentStat.Dev, parentStat.Ino) != current.ParentIdentity {
		err = errPreview
		return
	}
	check := func(phase string) error {
		receipt.Phase = phase
		if phaseHook != nil {
			if e := phaseHook(phase); e != nil {
				return e
			}
		}
		return ctx.Err()
	}
	if err = check("before_stage"); err != nil {
		return
	}
	if id, e := directoryIdentity(filepath.Dir(receipt.Destination)); e != nil || id != current.ParentIdentity {
		err = errPreview
		return
	}
	stage := ".mandalore-export-" + memory.NewID("stage")
	if err = unix.Mkdirat(parentFD, stage, 0700); err != nil {
		return
	}
	receipt.StagingDirectory = filepath.Join(filepath.Dir(receipt.Destination), stage)
	receipt.Phase = "staging"
	var initial unix.Stat_t
	if err = unix.Fstatat(parentFD, stage, &initial, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return
	}
	if initial.Mode&unix.S_IFMT != unix.S_IFDIR {
		err = errPreview
		return
	}
	stageFD, e := unix.Openat(parentFD, stage, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if e != nil {
		err = e
		return
	}
	stageFile := os.NewFile(uintptr(stageFD), stage)
	defer stageFile.Close()
	stageUnchanged := func() error {
		var held, named unix.Stat_t
		if e := unix.Fstat(stageFD, &held); e != nil {
			return e
		}
		if e := unix.Fstatat(parentFD, stage, &named, unix.AT_SYMLINK_NOFOLLOW); e != nil {
			return e
		}
		if held.Dev != initial.Dev || held.Ino != initial.Ino || named.Dev != held.Dev || named.Ino != held.Ino || named.Mode&unix.S_IFMT != unix.S_IFDIR {
			return errPreview
		}
		return nil
	}
	if err = stageUnchanged(); err != nil {
		return
	}
	if err = check("before_write"); err != nil {
		return
	}
	if id, e := directoryIdentity(filepath.Dir(receipt.Destination)); e != nil || id != current.ParentIdentity {
		err = errPreview
		return
	}
	if err = stageUnchanged(); err != nil {
		return
	}
	fileFD, e := unix.Openat(stageFD, "report.json", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if e != nil {
		err = e
		return
	}
	f := os.NewFile(uintptr(fileFD), "report.json")
	defer f.Close()
	receipt.ReportPath = filepath.Join(receipt.StagingDirectory, "report.json")
	receipt.Phase = "writing"
	for receipt.BytesWritten < len(data) {
		if err = ctx.Err(); err != nil {
			return
		}
		end := min(receipt.BytesWritten+32*1024, len(data))
		n, e := f.Write(data[receipt.BytesWritten:end])
		receipt.BytesWritten += n
		if e != nil {
			err = e
			return
		}
		if n == 0 {
			err = io.ErrShortWrite
			return
		}
	}
	if err = f.Sync(); err != nil {
		return
	}
	if err = f.Close(); err != nil {
		return
	}
	if err = stageFile.Sync(); err != nil {
		return
	}
	if err = check("before_publish"); err != nil {
		return
	}
	latest, _, e := prepare(ctx, current.Request)
	if e != nil {
		err = e
		return
	}
	if !reflect.DeepEqual(current, latest) {
		err = errPreview
		return
	}
	if err = check("publishing"); err != nil {
		return
	}
	if id, e := directoryIdentity(filepath.Dir(receipt.Destination)); e != nil || id != current.ParentIdentity {
		err = errPreview
		return
	}
	if err = stageUnchanged(); err != nil {
		return
	}
	if err = ctx.Err(); err != nil {
		return
	}
	receipt.Phase = "publishing"
	if err = publishDirectory(parentFD, stage, filepath.Base(receipt.Destination)); err != nil {
		return
	}
	receipt.Published = true
	receipt.ReportPath = filepath.Join(receipt.Destination, "report.json")
	receipt.Phase = "published"
	if err = parent.Sync(); err != nil {
		return
	}
	if err = ctx.Err(); err != nil {
		return
	}
	receipt.Durable = true
	receipt.Phase = "complete"
	return
}
