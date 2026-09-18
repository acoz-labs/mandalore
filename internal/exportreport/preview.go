package exportreport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	"golang.org/x/sys/unix"
)

type Request struct {
	BindingPath string    `json:"binding_path"`
	Destination string    `json:"destination"`
	Selection   Selection `json:"selection"`
}

// Plan is reviewable metadata, itself potentially sensitive. It never contains
// projected memory bodies. Apply must reconstruct it, not trust supplied hashes.
type Plan struct {
	Version           int        `json:"version"`
	Request           Request    `json:"request"`
	SignetID          string     `json:"signet_id"`
	Source            string     `json:"source"`
	SourceIdentity    string     `json:"source_identity"`
	ParentIdentity    string     `json:"parent_identity"`
	BindingSHA256     string     `json:"binding_sha256"`
	SourceSHA256      string     `json:"source_sha256"`
	ConnectionsSHA256 string     `json:"connections_sha256"`
	Projection        Projection `json:"projection"`
	ReportBytes       int        `json:"report_bytes"`
	ReportSHA256      string     `json:"report_sha256"`
	Effects           []string   `json:"effects"`
	Notice            string     `json:"notice"`
}

var errPreview = errors.New("cannot safely prepare export; inspect explicit binding, source, reference configuration and fresh destination")

func Preview(ctx context.Context, in Request) (Plan, error) {
	p, _, err := prepare(ctx, in)
	return p, err
}

func prepare(ctx context.Context, in Request) (result Plan, report []byte, err error) {
	defer func() {
		if err != nil {
			result = Plan{}
			report = nil
			if ctx.Err() != nil {
				err = ctx.Err()
			} else {
				err = errPreview
			}
		}
	}()
	if err = ctx.Err(); err != nil {
		return
	}
	if !validPath(in.BindingPath) || !validPath(in.Destination) {
		err = errPreview
		return
	}
	// The binding leaf must itself be a regular file; ancestor aliases may be
	// canonicalized. Hash the same bytes used to select the guarded connection.
	var b binding.Binding
	var bindingHash string
	in.BindingPath, b, bindingHash, err = readBinding(in.BindingPath)
	if err != nil {
		return
	}
	service, e := binding.OpenGuarded(in.BindingPath, "export", binding.Guard{SHA256: bindingHash, SignetID: b.SignetID})
	if e != nil {
		err = e
		return
	}
	source, e := filepath.EvalSymlinks(service.Root())
	if e != nil {
		err = e
		return
	}
	sourceID, e := directoryIdentity(source)
	if e != nil {
		err = e
		return
	}
	connections, e := foundlings.New(service).ConfiguredRoots(ctx)
	if e != nil {
		err = e
		return
	}
	protected := append([]string{source, filepath.Dir(in.BindingPath)}, connections.Roots...)
	in.Destination, err = destination(in.Destination, protected)
	if err != nil {
		return
	}
	parentID, e := directoryIdentity(filepath.Dir(in.Destination))
	if e != nil {
		err = e
		return
	}
	snapshot, e := ReadReportSnapshot(ctx, source)
	if e != nil {
		err = e
		return
	}
	if snapshot.Signet.ID != service.ID() {
		err = errPreview
		return
	}
	projection, data, e := project(snapshot, in.Selection)
	if e != nil {
		err = e
		return
	}
	// Fresh reads detect cooperative updates without writing a source lockfile.
	second, e := ReadReportSnapshot(ctx, source)
	if e != nil {
		err = e
		return
	}
	if snapshot.Digest != second.Digest {
		err = errPreview
		return
	}
	again, e := foundlings.New(service).ConfiguredRoots(ctx)
	if e != nil {
		err = e
		return
	}
	if !reflect.DeepEqual(connections, again) {
		err = errPreview
		return
	}
	_, _, latest, e := readBinding(in.BindingPath)
	if e != nil || latest != bindingHash {
		err = errPreview
		return
	}
	currentSource, e := directoryIdentity(source)
	if e != nil || sourceID != currentSource {
		err = errPreview
		return
	}
	currentParent, e := directoryIdentity(filepath.Dir(in.Destination))
	if e != nil || parentID != currentParent {
		err = errPreview
		return
	}
	if _, e = destination(in.Destination, protected); e != nil {
		err = e
		return
	}
	result = Plan{Version: 1, Request: in, SignetID: snapshot.Signet.ID, Source: source, SourceIdentity: sourceID, ParentIdentity: parentID, BindingSHA256: bindingHash, SourceSHA256: snapshot.Digest, ConnectionsSHA256: connections.SHA256, Projection: projection, ReportBytes: len(data), ReportSHA256: digest(data), Effects: []string{"create private sibling staging directory (0700)", "write selected report.json exclusively (0600)", "publish fresh destination without replacement"}, Notice: "Sensitive review metadata, not report contents. Source is unchanged. Derived report is not a backup or safe-to-publish certification. Partial output is retained on failure."}
	encoded, e := json.Marshal(result)
	if e != nil || len(encoded) > 24<<10 {
		err = errPreview
		return
	}
	if err = ctx.Err(); err != nil {
		return
	}
	return result, data, nil
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func validPath(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && len(path) <= 4096 && utf8.ValidString(path) && strings.IndexFunc(path, func(r rune) bool { return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) }) < 0
}

func readBinding(path string) (string, binding.Binding, string, error) {
	var b binding.Binding
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 16384 {
		return "", b, "", errPreview
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", b, "", errPreview
	}
	f, err := os.Open(canonical)
	if err != nil {
		return "", b, "", errPreview
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return "", b, "", errPreview
	}
	data, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil {
		return "", b, "", errPreview
	}
	if err = strictjson.Decode(data, &b, 16384); err != nil {
		return "", b, "", errPreview
	}
	return canonical, b, digest(data), nil
}

func directoryIdentity(path string) (string, error) {
	var st unix.Stat_t
	if err := unix.Stat(path, &st); err != nil {
		return "", errPreview
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR {
		return "", errPreview
	}
	return fmt.Sprintf("%d:%d", st.Dev, st.Ino), nil
}

func destination(path string, protected []string) (string, error) {
	if !validPath(path) {
		return "", errPreview
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		return "", errPreview
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", errPreview
	}
	if _, err := directoryIdentity(parent); err != nil {
		return "", errPreview
	}
	for _, root := range protected {
		id, err := directoryIdentity(root)
		if err != nil {
			return "", errPreview
		}
		// Identity traversal also handles aliases on case-insensitive volumes.
		for ancestor := parent; ; ancestor = filepath.Dir(ancestor) {
			other, err := directoryIdentity(ancestor)
			if err != nil || id == other {
				return "", errPreview
			}
			if filepath.Dir(ancestor) == ancestor {
				break
			}
		}
	}
	canonical := filepath.Join(parent, filepath.Base(path))
	if _, err := os.Lstat(canonical); !os.IsNotExist(err) {
		return "", errPreview
	}
	return canonical, nil
}
