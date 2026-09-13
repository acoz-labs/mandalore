package memory

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestMemoryDependencyBoundary(t *testing.T) {
	paths, err := filepath.Glob("*.go")
	if err != nil || len(paths) == 0 {
		t.Fatal("missing package files", err)
	}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			name, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			// net/url parses identifiers; it performs no network requests.
			if name == "os/exec" || name == "net" || (strings.HasPrefix(name, "net/") && name != "net/url") || strings.HasPrefix(name, "github.com/acoz-labs/") {
				t.Errorf("memory must not depend on process/network/assistant packages: %s imports %s", path, name)
			}
		}
	}
}
