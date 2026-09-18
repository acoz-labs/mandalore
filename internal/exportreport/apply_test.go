package exportreport

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPublishesOnlyReviewedReportPrivately(t *testing.T) {
	in, s := previewFixture(t)
	before, err := ReadReportSnapshot(context.Background(), s.Root())
	if err != nil {
		t.Fatal(err)
	}
	p, err := Preview(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Apply(context.Background(), p)
	if err != nil || !r.Published || !r.Durable || r.BytesWritten != p.ReportBytes {
		t.Fatal(r, err)
	}
	data, err := os.ReadFile(filepath.Join(p.Request.Destination, "report.json"))
	if err != nil || digest(data) != p.ReportSHA256 {
		t.Fatal("report differs", err)
	}
	for path, mode := range map[string]os.FileMode{p.Request.Destination: 0700, filepath.Join(p.Request.Destination, "report.json"): 0600} {
		st, err := os.Stat(path)
		if err != nil || st.Mode().Perm() != mode {
			t.Fatal("unsafe permissions", err)
		}
	}
	after, err := ReadReportSnapshot(context.Background(), s.Root())
	if err != nil || before.Digest != after.Digest {
		t.Fatal("source changed", err)
	}
	if _, err := Apply(context.Background(), p); err == nil {
		t.Fatal("replay overwrote report")
	}
}

func TestApplyRejectsTamperedAndStalePlanBeforeWrites(t *testing.T) {
	for _, mode := range []string{"digest", "notice", "source", "parent"} {
		t.Run(mode, func(t *testing.T) {
			in, s := previewFixture(t)
			p, err := Preview(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "digest":
				p.ReportSHA256 = "invented"
			case "notice":
				p.Notice = "invented"
			case "source":
				if _, err := s.AppendJournal("test", "changed"); err != nil {
					t.Fatal(err)
				}
			case "parent":
				parent := filepath.Dir(in.Destination)
				if err := os.Rename(parent, parent+"-moved"); err != nil {
					t.Fatal(err)
				}
				defer os.Rename(parent+"-moved", parent)
				if err := os.Mkdir(parent, 0700); err != nil {
					t.Fatal(err)
				}
			}
			r, err := Apply(context.Background(), p)
			if err == nil || r.StagingDirectory != "" || r.Published {
				t.Fatal("unsafe plan wrote output", r, err)
			}
		})
	}
}

func TestApplyRetainsUnconfirmedOutputOnCancellationAndDrift(t *testing.T) {
	for _, mode := range []string{"cancel", "source", "destination", "fault"} {
		t.Run(mode, func(t *testing.T) {
			in, s := previewFixture(t)
			p, err := Preview(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r, err := apply(ctx, p, func(phase string) error {
				if phase != "before_publish" {
					return nil
				}
				switch mode {
				case "cancel":
					cancel()
				case "source":
					_, e := s.AppendJournal("test", "changed")
					return e
				case "destination":
					return os.Mkdir(in.Destination, 0700)
				case "fault":
					return errors.New("PRIVATE_PROVIDER_DETAIL")
				}
				return nil
			})
			if err == nil || r.Published || r.Durable || r.StagingDirectory == "" {
				t.Fatal("partial output hidden", r, err)
			}
			if _, err := os.Stat(filepath.Join(r.StagingDirectory, "report.json")); err != nil {
				t.Fatal("partial output removed", err)
			}
			if mode == "destination" {
				entries, err := os.ReadDir(in.Destination)
				if err != nil || len(entries) != 0 {
					t.Fatal("concurrent destination changed", err)
				}
			}
		})
	}
}

func TestApplyNoReplacePublicationAndStageRedirection(t *testing.T) {
	for _, mode := range []string{"publish-race", "stage-link", "parent-moved"} {
		t.Run(mode, func(t *testing.T) {
			in, _ := previewFixture(t)
			p, err := Preview(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			foreign := t.TempDir()
			r, err := apply(context.Background(), p, func(phase string) error {
				if mode == "publish-race" && phase == "publishing" {
					return os.Mkdir(in.Destination, 0700)
				}
				if phase != "before_write" {
					return nil
				}
				parent := filepath.Dir(in.Destination)
				if mode == "parent-moved" {
					if err := os.Rename(parent, parent+"-moved"); err != nil {
						return err
					}
					return os.Mkdir(parent, 0700)
				}
				if mode == "stage-link" {
					entries, err := os.ReadDir(parent)
					if err != nil {
						return err
					}
					for _, entry := range entries {
						if strings.HasPrefix(entry.Name(), ".mandalore-export-") {
							path := filepath.Join(parent, entry.Name())
							if err := os.Rename(path, path+"-saved"); err != nil {
								return err
							}
							return os.Symlink(foreign, path)
						}
					}
				}
				return nil
			})
			if err == nil || r.Published || r.Durable {
				t.Fatal("race accepted", r, err)
			}
			entries, e := os.ReadDir(foreign)
			if e != nil || len(entries) != 0 {
				t.Fatal("wrote redirected stage", e)
			}
			if mode == "publish-race" {
				entries, e := os.ReadDir(in.Destination)
				if e != nil || len(entries) != 0 {
					t.Fatal("overwrote racing destination", e)
				}
			} else if r.BytesWritten != 0 {
				t.Fatal("wrote after directory replacement", r)
			}
		})
	}
}
