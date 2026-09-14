package migration

// The supported public source-change structure and checks are adapted from
// pinned My Friday portable/changes.go; see NOTICE. These operational Git-index
// observations are validated and kept only in original/, never current memory.
import (
	"errors"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

type sourceVersion struct {
	ObjectID string `json:"git_object_id"`
	Mode     string `json:"git_mode"`
}
type sourceChange struct {
	Version    int                `json:"schema_version"`
	ID         string             `json:"id"`
	RecordedAt string             `json:"recorded_at"`
	BaseCommit string             `json:"base_commit"`
	Observer   *memory.Authorship `json:"checkpoint_observer"`
	Files      []struct {
		Path   string         `json:"path"`
		Before *sourceVersion `json:"before"`
		After  *sourceVersion `json:"after"`
	} `json:"files"`
}

var objectID = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

func validateChange(name string, b []byte, devices []memory.Device) error {
	fail := errors.New("unsupported source-change provenance: " + name)
	var c sourceChange
	if strictjson.Decode(b, &c, MaxFileBytes) != nil || c.Version != 1 || !idPattern.MatchString(c.ID) || !strings.HasPrefix(c.ID, "change-") || name != "provenance/changes/"+c.ID+".json" || len(c.Files) == 0 {
		return fail
	}
	if _, err := time.Parse(time.RFC3339Nano, c.RecordedAt); err != nil {
		return fail
	}
	if c.BaseCommit != "" && !objectID.MatchString(c.BaseCommit) {
		return fail
	}
	if c.Observer != nil {
		known := false
		for _, d := range devices {
			if d.ID == c.Observer.DeviceID {
				known = true
			}
		}
		if !known || strings.TrimSpace(c.Observer.Actor) == "" || strings.TrimSpace(c.Observer.Harness) == "" {
			return fail
		}
	}
	previous := ""
	for _, f := range c.Files {
		if f.Path <= previous || strings.ContainsAny(f.Path, "\x00\\") || path.IsAbs(f.Path) || path.Clean(f.Path) != f.Path || f.Path == "." || f.Path == ".." || strings.HasPrefix(f.Path, "../") {
			return fail
		}
		for _, prefix := range []string{".git", "memory", "provenance"} {
			if f.Path == prefix || strings.HasPrefix(f.Path, prefix+"/") {
				return fail
			}
		}
		previous = f.Path
		if f.Before == nil && f.After == nil || f.Before != nil && f.After != nil && *f.Before == *f.After {
			return fail
		}
		for _, v := range []*sourceVersion{f.Before, f.After} {
			if v == nil {
				continue
			}
			if !objectID.MatchString(v.ObjectID) || strings.Trim(v.ObjectID, "0") == "" {
				return fail
			}
			switch v.Mode {
			case "100644", "100755", "120000", "160000":
			default:
				return fail
			}
		}
	}
	return nil
}
