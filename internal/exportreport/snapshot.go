package exportreport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

const maxReportSourceBytes = 128 << 20

func ReadReportSnapshot(ctx context.Context, directory string) (memory.ReportSnapshot, error) {
	return readReportSnapshot(ctx, directory, maxReportSourceBytes)
}

func readReportSnapshot(ctx context.Context, directory string, limit int64) (result memory.ReportSnapshot, err error) {
	// Never expose raw decoder errors that can quote sensitive input.
	defer func() {
		if err != nil && ctx.Err() == nil {
			err = errors.New("cannot validate a bounded report snapshot; inspect source structure or narrow the source")
		}
	}()
	s, err := memory.Open(directory)
	if err != nil {
		return result, err
	}
	if err = s.ValidateReadLayout(); err != nil {
		return result, err
	}
	r, err := os.OpenRoot(s.Root)
	if err != nil {
		return result, err
	}
	defer r.Close()
	identity, err := r.Stat(".")
	if err != nil {
		return result, err
	}
	h := sha256.New()
	var consumed int64
	read := func(name string, out any) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		before, err := r.Lstat(name)
		if err != nil || !before.Mode().IsRegular() || before.Size() > 4<<20 {
			return errors.New("invalid portable file")
		}
		f, err := r.Open(name)
		if err != nil {
			return err
		}
		opened, e := f.Stat()
		if e != nil || !os.SameFile(before, opened) {
			f.Close()
			return errors.New("source identity changed")
		}
		b, e := io.ReadAll(io.LimitReader(f, (4<<20)+1))
		closeErr := f.Close()
		if e != nil {
			return e
		}
		if closeErr != nil {
			return closeErr
		}
		consumed += int64(len(b))
		if len(b) > 4<<20 || consumed > limit {
			return errors.New("source exceeds report read budget")
		}
		if e := strictjson.Decode(b, out, 4<<20); e != nil {
			return e
		}
		fmt.Fprintf(h, "%d:%s:%d:", len(name), name, len(b))
		_, _ = h.Write(b)
		return nil
	}
	if err = read("signet.json", &result.Signet); err != nil {
		return result, err
	}
	for _, base := range []string{"memory", "provenance", "foundlings"} {
		err = fs.WalkDir(r.FS(), base, func(name string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if e := ctx.Err(); e != nil {
				return e
			}
			if d.Type()&os.ModeSymlink != 0 {
				return errors.New("symlink in portable data")
			}
			if d.IsDir() {
				return nil
			}
			if d.Name() == ".gitkeep" && d.Type().IsRegular() {
				return nil
			}
			if !d.Type().IsRegular() {
				return errors.New("nonregular portable data")
			}
			var canonical string
			switch {
			case strings.HasPrefix(name, "memory/records/"):
				var v memory.Revision
				if e := read(name, &v); e != nil {
					return e
				}
				canonical = path.Join("memory/records", v.RecordID, v.ID+".json")
				result.Revisions = append(result.Revisions, v)
			case strings.HasPrefix(name, "memory/sources/"):
				var v memory.Source
				if e := read(name, &v); e != nil {
					return e
				}
				canonical = path.Join("memory/sources", v.ID+".json")
				result.Sources = append(result.Sources, v)
			case strings.HasPrefix(name, "memory/events/"):
				var v memory.JournalEntry
				if e := read(name, &v); e != nil {
					return e
				}
				at, e := time.Parse(time.RFC3339Nano, v.RecordedAt)
				if e != nil {
					return e
				}
				canonical = path.Join("memory/events", at.UTC().Format("2006/01"), v.ID+".json")
				result.Journal = append(result.Journal, v)
			case strings.HasPrefix(name, "provenance/devices/"):
				var v memory.Device
				if e := read(name, &v); e != nil {
					return e
				}
				canonical = path.Join("provenance/devices", v.ID+".json")
				result.Devices = append(result.Devices, v)
			case strings.HasPrefix(name, "foundlings/registrations/"):
				var v memory.FoundlingRegistration
				if e := read(name, &v); e != nil {
					return e
				}
				canonical = path.Join("foundlings/registrations", v.FoundlingID, v.ID+".json")
				result.Registrations = append(result.Registrations, v)
			default:
				return errors.New("unknown portable data")
			}
			if canonical != name {
				return errors.New("noncanonical portable object path")
			}
			return nil
		})
		if err != nil {
			return result, err
		}
	}
	if result.Signet != s.Signet {
		return result, memory.ErrIdentityChanged
	}
	if err = memory.ValidateReportSnapshot(result.Snapshot, result.Registrations); err != nil {
		return result, err
	}
	current, err := os.Stat(s.Root)
	if err != nil || !os.SameFile(identity, current) {
		return result, memory.ErrIdentityChanged
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	result.Digest = hex.EncodeToString(h.Sum(nil))
	return result, nil
}
