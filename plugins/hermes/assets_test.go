package hermes

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonAndEngineAgreeOnPackageIdentity(t *testing.T) {
	files, err := PackageFiles()
	if err != nil {
		t.Fatal(err)
	}
	info, err := Inspect()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for name, raw := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"connection.json", "session-policy.json"} {
		if _, ok := files[name]; ok {
			t.Fatal("private connection material in public package")
		}
	}
	// Python is a required adapter test dependency, not a silently skipped gate.
	cmd := exec.Command("python3", "-B", "-c", `import importlib.util,sys
from pathlib import Path
root=Path(sys.argv[1])
spec=importlib.util.spec_from_file_location("mandalore_identity_test",root/"__init__.py",submodule_search_locations=[str(root)])
module=importlib.util.module_from_spec(spec)
sys.modules[spec.name]=module
spec.loader.exec_module(module)
print(sys.modules[spec.name+".connection"].package_digest(root))`, root)
	out, err := cmd.CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != info.SHA256 {
		t.Fatalf("Python and engine package identity disagree: %s %v", out, err)
	}
}
