# Synthetic privacy lifetime probe

Basis: `a8b56b268ed48f47363d015ab6d5f89e847bee6c`.
Executed 2026-09-16 using pinned Go 1.26.4, macOS arm64; local Git only.
This is a temporary characterization test, not a shipped privacy feature.
No native agent/provider, live bank or remote service was used.

## Reproduce

Place the following code temporarily at
`internal/api/privacy_discovery_test.go` in a disposable checkout of the basis,
then run:

```sh
gofmt -w internal/api/privacy_discovery_test.go
mise exec -- go test ./internal/api -run '^TestPrivacyDiscoveryLifetime$' -count=1 -v
```

The test creates its own temporary bank and Git repository. It deletes only its
own synthetic predecessor file to test refusal; Go's test cleanup removes its
disposable fixtures. Do not substitute a real bank or a real credential.
Remove only the temporary test source afterward. O1 should promote these checks
into maintainable focused regression tests.

```go
package api

import (
 "context"
 "crypto/sha256"
 "encoding/json"
 "io/fs"
 "os"
 "path/filepath"
 "reflect"
 "strings"
 "testing"

 "github.com/acoz-labs/mandalore/internal/memory"
)

func TestPrivacyDiscoveryLifetime(t *testing.T) {
 a := fixture(t)
 ctx := context.Background()
 const marker = "SYNTHETIC-PRIVACY-CANARY-NOT-A-SECRET"
 first, err := a.service.Remember(memory.Write{Kind:"fact", Summary:"Privacy canary", Body:marker, Basis:"observation", Reason:"Synthetic probe", Sensitivity:"restricted"})
 if err != nil { t.Fatal(err) }
 if _, err := a.service.AppendJournal("probe", marker); err != nil { t.Fatal(err) }
 if out := a.Call(ctx, "memory_git_init", []byte("{}")); !out.OK { t.Fatal(out.Error) }
 oldHead := inlineGit(t, "-C", a.service.Root(), "rev-parse", "HEAD")
 relative := "memory/records/" + first.RecordID + "/" + first.ID + ".json"

 recall := a.Call(ctx, "memory_recall", []byte("{}"))
 raw, err := json.Marshal(recall)
 if err != nil || !recall.OK || !strings.Contains(string(raw), marker) { t.Fatal("restricted content was not recalled", err) }
 text, warning := contextText(t, a, "Privacy canary")
 if warning != "" || !strings.Contains(text, marker) { t.Fatal("restricted content was not in native context") }

 if _, err := a.service.Remember(memory.Write{Kind:"fact", Summary:"Privacy canary", Body:"Replacement without original marker.", Basis:"user-direction", Reason:"Synthetic correction", RecordID:first.RecordID, Supersedes:[]string{first.ID}}); err != nil { t.Fatal(err) }
 packet, err := a.service.Recall("", nil, 5, 8192)
 if err != nil || len(packet.Current) != 1 || packet.Current[0].Body == marker { t.Fatal("correction did not change current recall", err) }
 history, err := a.service.History(first.RecordID)
 if err != nil || len(history) != 2 { t.Fatal("missing history", err) }
 preserved := false
 for _, revision := range history { preserved = preserved || revision.Body == marker }
 if !preserved { t.Fatal("correction erased predecessor") }
 entries, err := a.service.Journal(marker, 5)
 if err != nil || len(entries) != 1 { t.Fatal("journal did not preserve independent copy", err) }
 if !strings.Contains(inlineGit(t, "-C", a.service.Root(), "show", oldHead+":"+relative), marker) { t.Fatal("old Git blob disappeared") }

 // Include operational state and .git as well as portable content.
 inventory := func() map[string][32]byte {
  out := map[string][32]byte{}
  err := filepath.WalkDir(a.service.Root(), func(path string, entry fs.DirEntry, err error) error {
   if err != nil { return err }
   if entry.IsDir() { return nil }
   data, err := os.ReadFile(path)
   if err != nil { return err }
   rel, err := filepath.Rel(a.service.Root(), path)
   if err != nil { return err }
   out[rel] = sha256.Sum256(data)
   return nil
  })
  if err != nil { t.Fatal(err) }
  return out
 }
 a.ReadOnly = true
 before := inventory()
 for _, operation := range []string{"memory_remember", "memory_journal_append", "memory_remember_and_sync", "memory_journal_append_and_sync", "memory_git_init", "memory_checkpoint", "memory_sync"} {
  out := a.Call(ctx, operation, []byte("{}"))
  if out.OK || out.Error == nil || out.Error.Code != "operation.read_only" || out.Error.WriteMayHaveOccurred { t.Fatal(operation, out.Error) }
 }
 // A no-save connection can still return historical content to the caller.
 if out := a.Call(ctx, "memory_journal", []byte("{}")); !out.OK { t.Fatal(out.Error) }
 if !reflect.DeepEqual(before, inventory()) { t.Fatal("read-only calls changed file names or bytes") }

 a.ReadOnly = false
 if out := a.Call(ctx, "memory_checkpoint", []byte("{}")); !out.OK { t.Fatal(out.Error) }
 checkpoint := inlineGit(t, "-C", a.service.Root(), "rev-parse", "HEAD")
 // Remove only this test-created predecessor in a disposable fixture.
 if err := os.Remove(filepath.Join(a.service.Root(), relative)); err != nil { t.Fatal(err) }
 if out := a.Call(ctx, "memory_checkpoint", []byte("{}")); out.OK { t.Fatal("deleted predecessor checkpointed") }
 if inlineGit(t, "-C", a.service.Root(), "rev-parse", "HEAD") != checkpoint { t.Fatal("rejected deletion moved HEAD") }
 if !strings.Contains(inlineGit(t, "-C", a.service.Root(), "show", oldHead+":"+relative), marker) { t.Fatal("local deletion erased old Git blob") }
}
```

## Observed result

```text
=== RUN   TestPrivacyDiscoveryLifetime
--- PASS: TestPrivacyDiscoveryLifetime (1.00s)
PASS
ok github.com/acoz-labs/mandalore/internal/api 1.397s
```

Assertions exercised actual service/API/Git paths: a restricted record reached
both recall and the native context packet; a correction changed current recall
without erasing history, the independent journal copy or an old Git blob;
read-only rejected seven mutation operations and left the complete fixture file
inventory/bytes unchanged; deleting the predecessor was refused without advancing
HEAD, and the original committed blob still existed.

Limits: the read-only inventory hashes names/content, not access times or OS
caches. Rejected deletion is graph-invalid and does not independently isolate the
append-only guard; the existing checkpoint rewrite test covers that separate
guard. No proof of cross-device erasure, provider behavior, native UI acceptance,
semantic classification or secret detection. This is not a benchmark.
