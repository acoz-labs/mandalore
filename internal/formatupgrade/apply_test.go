package formatupgrade

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

func TestPreparedRoundTrip(t *testing.T) {
	in, _ := fixture(t)
	p, err := Preview(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	want := prepared{Plan: p, OriginalManifest: "original", Manifest: "next"}
	b, err := encoded(want)
	if err != nil {
		t.Fatal(err)
	}
	var got prepared
	if err := strictjson.Decode(b, &got, preparedLimit); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatal("preparation changed during round trip")
	}
}

func TestRecoveryCannotStartUpgrade(t *testing.T) {
	in, service := fixture(t)
	p, err := Preview(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Recover(context.Background(), ApplyRequest{Plan: p, StoppedWriters: true})
	if err == nil || r.StagingDirectory != "" || r.Activated {
		t.Fatal(r, err)
	}
	if _, err := os.Lstat(filepath.Join(service.Root(), stage)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("recovery created preparation", err)
	}
}

func TestPreparedDriftStopsBeforePublishingEvidence(t *testing.T) {
	for _, target := range []string{"binding", "source"} {
		t.Run(target, func(t *testing.T) {
			in, service := fixture(t)
			p, err := Preview(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			r, err := apply(context.Background(), ApplyRequest{Plan: p, StoppedWriters: true}, false, func(phase string) error {
				if phase != "prepared" {
					return nil
				}
				file := in.BindingPath
				if target == "source" {
					file = filepath.Join(service.Root(), "signet.json")
				}
				b, err := os.ReadFile(file)
				if err != nil {
					return err
				}
				return os.WriteFile(file, append(b, '\n'), 0600)
			})
			if err == nil || r.EvidencePublished || r.Activated {
				t.Fatal(r, err)
			}
			if _, err := os.Lstat(filepath.Join(service.Root(), "provenance/upgrades", r.EvidenceID+".json")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("source drift published evidence", err)
			}
		})
	}
}

func TestCanceledUpgradeRecoversWithoutRepeatingEvidence(t *testing.T) {
	for _, phase := range []string{"stage-created", "prepared", "receipt-linked", "receipt-durable", "manifest-replaced", "activated"} {
		t.Run(phase, func(t *testing.T) {
			in, _ := fixture(t)
			p, err := Preview(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			request := ApplyRequest{Plan: p, StoppedWriters: true}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r, err := apply(ctx, request, false, func(current string) error {
				if current == phase {
					cancel()
					return ctx.Err()
				}
				return nil
			})
			if !errors.Is(err, context.Canceled) || r.DurableLocally {
				t.Fatal(r, err)
			}
			again, err := Recover(context.Background(), request)
			if err != nil || !again.DurableLocally || (r.EvidencePublished && r.EvidenceID != again.EvidenceID) {
				t.Fatal(again, err)
			}
		})
	}
}

func TestApplyRequiresAcknowledgementAndPreservesHistory(t *testing.T) {
	in, service := fixture(t)
	originals := map[string][]byte{}
	if err := filepath.WalkDir(service.Root(), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err == nil {
			originals[path] = data
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	p, err := Preview(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), ApplyRequest{Plan: p}); err == nil {
		t.Fatal("missing stopped-writer acknowledgement accepted")
	}
	receipt, err := Apply(context.Background(), ApplyRequest{Plan: p, StoppedWriters: true})
	if err != nil || !receipt.Activated || !receipt.EvidencePublished || !receipt.DurableLocally || receipt.Checkpointed || receipt.Delivered {
		t.Fatal(receipt, err)
	}
	s, err := memory.Open(service.Root())
	if err != nil || s.Signet.Version != 2 {
		t.Fatal(s, err)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	for path, want := range originals {
		if path == filepath.Join(service.Root(), "signet.json") {
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("original evidence or Git metadata changed: %s: %v", path, err)
		}
	}
	if _, err := Apply(context.Background(), ApplyRequest{Plan: p, StoppedWriters: true}); err == nil {
		t.Fatal("blind replay accepted")
	}
	again, err := Recover(context.Background(), ApplyRequest{Plan: p, StoppedWriters: true})
	if err != nil || !again.Activated || !again.DurableLocally || again.EvidenceID != receipt.EvidenceID {
		t.Fatal(again, err)
	}
}

func TestUpgradeInterruptionsRecoverExplicitly(t *testing.T) {
	for _, phase := range []string{"stage-created", "prepared", "receipt-linked", "receipt-durable", "manifest-replaced", "activated"} {
		t.Run(phase, func(t *testing.T) {
			in, service := fixture(t)
			p, err := Preview(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			request := ApplyRequest{Plan: p, StoppedWriters: true}
			r, err := apply(context.Background(), request, false, func(current string) error {
				if current == phase {
					return errors.New("synthetic interruption")
				}
				return nil
			})
			if err == nil || r.DurableLocally {
				t.Fatal("interruption reported completed durability", r, err)
			}
			recovered, err := Recover(context.Background(), request)
			if err != nil || !recovered.Activated || !recovered.DurableLocally {
				t.Fatal(recovered, err)
			}
			if r.EvidencePublished && r.EvidenceID != recovered.EvidenceID {
				t.Fatal("recovery generated another portable receipt")
			}
			s, err := memory.Open(service.Root())
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRecoveryRefusesSourceAndBindingDrift(t *testing.T) {
	for _, target := range []string{"binding", "source"} {
		t.Run(target, func(t *testing.T) {
			in, service := fixture(t)
			p, err := Preview(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			request := ApplyRequest{Plan: p, StoppedWriters: true}
			if _, err := apply(context.Background(), request, false, func(phase string) error {
				if phase == "receipt-durable" {
					return errors.New("synthetic pause")
				}
				return nil
			}); err == nil {
				t.Fatal("fault missing")
			}
			file := in.BindingPath
			if target == "source" {
				file = filepath.Join(service.Root(), "signet.json")
			}
			b, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, append(b, '\n'), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Recover(context.Background(), request); err == nil {
				t.Fatal("recovery ignored changed pins")
			}
			s, err := memory.Open(service.Root())
			if err != nil || s.Signet.Version != 1 {
				t.Fatal("failed recovery activated format", err)
			}
		})
	}
}
