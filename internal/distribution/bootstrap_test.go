package distribution

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBootstrapVerifiedHandoffAndRefusals(t *testing.T) {
	for _, scenario := range []string{"valid", "bad-digest", "duplicate", "missing", "download-error", "redirect", "redirect-loop", "old-curl", "unsupported", "invalid-version"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			bin, scratch := filepath.Join(root, "bin"), filepath.Join(root, "scratch")
			for _, dir := range []string{bin, scratch} {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			payload := []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$TEST_OUTPUT\"\n")
			sums := Digest(payload) + "  mandalore_1.0.0_linux_arm64\n"
			switch scenario {
			case "bad-digest":
				sums = strings.Repeat("0", 64) + "  mandalore_1.0.0_linux_arm64\n"
			case "duplicate":
				sums += sums
			case "missing":
				sums = Digest(payload) + "  another-file\n"
			}
			for name, data := range map[string][]byte{"payload": payload, "sums": []byte(sums)} {
				if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			curl := `#!/bin/sh
[ "$1" = -q ] || exit 90
shift
if [ "$1" = --version ]; then
  if [ "$SCENARIO" = old-curl ]; then echo 'curl 7.88.1'; else echo 'curl 8.7.1'; fi
  exit 0
fi
[ "$SCENARIO" != download-error ] || exit 22
while [ "$#" -gt 0 ]; do
  case "$1" in --output) output=$2; shift;; esac
  url=$1
  shift
done
printf '%s\n' "$url" >> "$TEST_FIXTURES/requests"
case "$url" in
  https://github.com/acoz-labs/mandalore/releases/download/v1.0.0/SHA256SUMS) cp "$TEST_FIXTURES/sums" "$output";;
  https://github.com/acoz-labs/mandalore/releases/download/v1.0.0/mandalore_1.0.0_linux_arm64) cp "$TEST_FIXTURES/payload" "$output";;
  *) exit 91;;
esac
if [ "$SCENARIO" = redirect ]; then printf '302\nhttps://untrusted.example/file'
elif [ "$SCENARIO" = redirect-loop ]; then printf '302\n%s' "$url"
else printf '200\n'; fi
`
			uname := "#!/bin/sh\nif [ \"$SCENARIO\" = unsupported ]; then echo Unsupported; elif [ \"$1\" = -s ]; then echo Linux; else echo aarch64; fi\n"
			for name, data := range map[string]string{"curl": curl, "uname": uname} {
				if err := os.WriteFile(filepath.Join(bin, name), []byte(data), 0700); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			v := "1.0.0"
			if scenario == "invalid-version" {
				v = "1.0.0/../../escape"
			}
			cmd := exec.CommandContext(ctx, "/bin/sh", "../../packaging/install.sh", v)
			cmd.Env = []string{"PATH=" + bin + ":/usr/bin:/bin", "TMPDIR=" + scratch, "TEST_FIXTURES=" + root, "TEST_OUTPUT=" + filepath.Join(root, "executed"), "SCENARIO=" + scenario}
			out, err := cmd.CombinedOutput()
			if scenario == "valid" {
				if err != nil {
					t.Fatalf("valid bootstrap failed: %v %s", err, out)
				}
				args, err := os.ReadFile(filepath.Join(root, "executed"))
				if err != nil || string(args) != "release\ninstall\n--version\n1.0.0\n" {
					t.Fatalf("wrong handoff: %q %v", args, err)
				}
				requests, err := os.ReadFile(filepath.Join(root, "requests"))
				if err != nil || len(strings.Fields(string(requests))) != 2 {
					t.Fatal("bootstrap must account for its two independent asset downloads", err, string(requests))
				}
			} else {
				if err == nil {
					t.Fatal("invalid bootstrap unexpectedly succeeded")
				}
				if _, err := os.Stat(filepath.Join(root, "executed")); !os.IsNotExist(err) {
					t.Fatal("unverified payload executed")
				}
			}
			entries, err := os.ReadDir(scratch)
			if err != nil || len(entries) != 0 {
				t.Fatal("bootstrap did not clean its own temporary directory")
			}
		})
	}
}
