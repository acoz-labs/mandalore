// Package sessionsync coordinates foreground transport under an explicitly
// selected, immutable session policy. Legacy connections never construct it.
package sessionsync

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

const PolicyVersion = 1
const EnabledMode = "enabled-session"

type Policy struct {
	SchemaVersion int    `json:"schema_version"`
	Mode          string `json:"mode"`
	Binding       string `json:"binding"`
	BindingSHA256 string `json:"binding_sha256"`
	SignetID      string `json:"signet_id"`
	Runtime       string `json:"runtime"`
	RuntimeSHA256 string `json:"runtime_sha256"`
	StateDir      string `json:"state_dir"`
}

type Selection struct {
	PolicySHA256 string
	Binding      string
	Guard        binding.Guard
	Runtime      string
}

type Coordinator struct {
	policyPath string
	selected   Selection
	policy     Policy
	root       string
	identity   string
}

func hash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func validDigest(v string) bool {
	raw, err := hex.DecodeString(v)
	return err == nil && len(raw) == 32 && strings.ToLower(v) == v
}
func realPath(path string) bool {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.IndexFunc(path, unicode.IsControl) >= 0 {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	return err == nil && resolved == path
}
func readFile(path string, limit int64) ([]byte, error) {
	if _, err := os.Lstat(path); err != nil {
		return nil, err
	}
	if !realPath(path) {
		return nil, errors.New("session selection path is missing or redirected")
	}
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > limit {
		return nil, errors.New("session selection file is invalid")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return nil, errors.New("session selection exceeds byte limit")
	}
	return raw, nil
}

// Open validates explicit policy and selection without writing or synchronizing.
func Open(path string, selected Selection) (*Coordinator, error) {
	if !validDigest(selected.PolicySHA256) || selected.Guard.SHA256 == "" || selected.Guard.SignetID == "" || selected.Guard.Validate() != nil {
		return nil, errors.New("session policy requires explicit identity pins")
	}
	raw, err := readFile(path, 16384)
	if err != nil || hash(raw) != selected.PolicySHA256 {
		return nil, errors.New("session policy does not match selected bytes")
	}
	var p Policy
	if strictjson.Decode(raw, &p, 16384) != nil || p.SchemaVersion != PolicyVersion || p.Mode != EnabledMode {
		return nil, errors.New("unsupported session transport policy")
	}
	if p.Binding != selected.Binding || p.BindingSHA256 != selected.Guard.SHA256 || p.SignetID != selected.Guard.SignetID || p.Runtime != selected.Runtime || !validDigest(p.RuntimeSHA256) || !realPath(p.StateDir) {
		return nil, errors.New("session policy differs from selected connection")
	}
	c := &Coordinator{policyPath: path, selected: selected, policy: p}
	s, err := binding.OpenGuarded(p.Binding, "session-policy", selected.Guard)
	if err != nil {
		return nil, errors.New("session policy binding is unavailable")
	}
	c.root = s.Root()
	if !realPath(c.root) {
		return nil, errors.New("session checkout is redirected")
	}
	// Coordination is per physical checkout, not logical signet alone. Every
	// caller validates its own policy before adopting overlapping work.
	c.identity = hash([]byte(c.root + "\n" + p.SignetID))
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Coordinator) Validate() error {
	if c == nil {
		return errors.New("session policy is absent")
	}
	raw, err := readFile(c.policyPath, 16384)
	if err != nil || hash(raw) != c.selected.PolicySHA256 {
		return errors.New("session policy changed")
	}
	raw, err = readFile(c.policy.Runtime, 128<<20)
	if err != nil || hash(raw) != c.policy.RuntimeSHA256 {
		return errors.New("session runtime changed")
	}
	s, err := binding.OpenGuarded(c.policy.Binding, "session-policy", c.selected.Guard)
	if err != nil || s.Root() != c.root || s.ID() != c.policy.SignetID {
		return errors.New("session binding changed")
	}
	if !realPath(c.root) || !realPath(c.policy.StateDir) {
		return errors.New("session checkout or installation state changed")
	}
	return nil
}

func (c *Coordinator) Matches(s *memory.Service) bool {
	return c != nil && s != nil && s.ID() == c.policy.SignetID && s.Root() == c.root
}
