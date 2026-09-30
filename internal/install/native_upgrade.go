package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var nativeVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)

// Remember the explicitly selected launcher before resolving its current target.
// Existing plans omit this field and retain their original content identity.
func prepareNative(o *Options) error {
	source := o.NativeBinary
	if o.NativeLauncher != "" {
		source = o.NativeLauncher
	}
	if !receiptPath(source) {
		return errors.New("native launcher requires an absolute path")
	}
	launcher := filepath.Clean(source)
	target, err := canonical(launcher)
	if err != nil {
		return err
	}
	if target != launcher || o.NativeLauncher != "" {
		o.NativeLauncher = launcher
	}
	o.NativeBinary = target
	return nil
}

// nativeSnapshot follows only the locator saved by the user or a recognized
// updater layout derived from the original executable. PATH is never searched.
func nativeSnapshot(harness string, o Options) (string, string, string, error) {
	source := o.NativeBinary
	if o.NativeLauncher != "" {
		source = o.NativeLauncher
	} else {
		launcher, err := legacyNativeLauncher(harness, source)
		if err != nil {
			return "", "", "", err
		}
		if launcher != "" {
			source = launcher
		}
	}
	target, err := canonical(source)
	if err != nil {
		return "", "", "", errors.New("native launcher is unavailable; restore its installation or explicitly select a trusted executable")
	}
	st, err := os.Stat(target)
	if err != nil || !st.Mode().IsRegular() || st.Mode()&0111 == 0 {
		return "", "", "", errors.New("selected native executable is unavailable or not executable; restore its installation or explicitly select a trusted executable")
	}
	digest, err := digestLimit(target, maxNativeBinary)
	return target, digest, source, err
}

// Older receipts resolved Claude's documented native-install symlink to a
// version file. Recover only its same-installation launcher and version family.
func legacyNativeLauncher(harness, original string) (string, error) {
	invalid := errors.New("native updater launcher is missing or redirected outside its installation; restore the native installation or explicitly select a trusted executable")
	if harness == "pi" {
		// mise's npm backend publishes a latest symlink inside the same tool root.
		marker := string(filepath.Separator) + filepath.Join("installs", "npm-earendil-works-pi-coding-agent") + string(filepath.Separator)
		if i := strings.LastIndex(original, marker); i >= 0 {
			root := original[:i+len(marker)-1]
			relative := strings.TrimPrefix(original, root+string(filepath.Separator))
			version, tail, ok := strings.Cut(relative, string(filepath.Separator))
			if ok && nativeVersionPattern.MatchString(version) && (tail == filepath.Join("node_modules", ".bin", "pi") || tail == filepath.Join("node_modules", "@earendil-works", "pi-coding-agent", "dist", "bundle", "cli.js")) {
				if _, err := os.Lstat(filepath.Join(root, "latest")); os.IsNotExist(err) {
					return "", nil
				}
				launcher := filepath.Join(root, "latest", tail)
				current, err := filepath.EvalSymlinks(filepath.Join(root, "latest"))
				target, targetErr := filepath.EvalSymlinks(launcher)
				if err == nil && targetErr == nil && filepath.Dir(current) == root && nativeVersionPattern.MatchString(filepath.Base(current)) && inside(filepath.Join(current, "node_modules"), target) {
					return launcher, nil
				}
				return "", invalid
			}
		}
	}
	if harness == "codex" && filepath.Base(original) == "codex" && filepath.Base(filepath.Dir(original)) == "bin" {
		release := filepath.Dir(filepath.Dir(original))
		releases := filepath.Dir(release)
		if filepath.Base(releases) == "releases" && filepath.Base(filepath.Dir(releases)) == "standalone" {
			if _, err := os.Lstat(filepath.Join(filepath.Dir(releases), "current")); os.IsNotExist(err) {
				return "", nil
			}
			launcher := filepath.Join(filepath.Dir(releases), "current", "bin", "codex")
			target, err := filepath.EvalSymlinks(launcher)
			if err == nil && filepath.Dir(filepath.Dir(filepath.Dir(target))) == releases && filepath.Base(target) == "codex" && filepath.Base(filepath.Dir(target)) == "bin" {
				return launcher, nil
			}
			return "", invalid
		}
	}
	if harness != "claude-code" || !nativeVersionPattern.MatchString(filepath.Base(original)) {
		return "", nil
	}
	versions := filepath.Dir(original)
	suffix := filepath.Join("share", "claude", "versions")
	if !strings.HasSuffix(versions, string(filepath.Separator)+suffix) {
		return "", nil
	}
	base := strings.TrimSuffix(versions, suffix)
	launcher := filepath.Join(base, "bin", "claude")
	if _, err := os.Lstat(launcher); os.IsNotExist(err) {
		return "", nil
	}
	target, err := filepath.EvalSymlinks(launcher)
	if err == nil && filepath.Dir(target) == versions && nativeVersionPattern.MatchString(filepath.Base(target)) {
		return launcher, nil
	}
	return "", invalid
}

func verifyNativePreview(o Options, want string) error {
	// A launcher retargeted between preview and apply invalidates the preview even
	// if the former target remains on disk.
	target := o.NativeBinary
	if o.NativeLauncher != "" {
		resolved, err := canonical(o.NativeLauncher)
		if err != nil || resolved != target {
			return errors.New("selected native launcher changed after preview")
		}
	}
	got, err := digestLimit(target, maxNativeBinary)
	if err != nil || got != want {
		return errors.New("selected native executable changed after preview")
	}
	return nil
}

// Help establishes the CLI surfaces Mandalore uses, not loaded model context.
// Missing interfaces fail with the capability name and leave native state alone.
func nativeCapabilities(ctx context.Context, harness string, o Options, run runner) error {
	probes := []struct{ args, tokens []string }{}
	switch harness {
	case "claude-code":
		probes = append(probes, struct{ args, tokens []string }{[]string{"plugin", "--help"}, []string{"install", "uninstall", "list", "marketplace"}})
		probes = append(probes, struct{ args, tokens []string }{[]string{"plugin", "marketplace", "--help"}, []string{"add", "remove", "list"}})
	case "pi":
		probes = append(probes, struct{ args, tokens []string }{[]string{"--help"}, []string{"--mode", "--extension", "--no-session", "install", "remove"}})
	}
	for _, probe := range probes {
		raw, err := run(ctx, o, probe.args...)
		if err != nil {
			return fmt.Errorf("incompatible %s CLI capability %s; restore a compatible native installation: %w", harness, strings.Join(probe.args, " "), err)
		}
		for _, token := range probe.tokens {
			if !regexp.MustCompile(`(^|[^A-Za-z0-9_-])` + regexp.QuoteMeta(token) + `([^A-Za-z0-9_-]|$)`).Match(raw) {
				return fmt.Errorf("incompatible %s CLI: required %s capability is missing; restore a compatible native installation", harness, token)
			}
		}
	}
	return nil
}

func selectedNative(harness string, o Options, selected string) (Options, string, string, error) {
	target, hash, locator, err := nativeSnapshot(harness, o)
	if err != nil {
		return o, "", "", err
	}
	if selected != o.NativeBinary && selected != o.NativeLauncher && selected != target {
		return o, "", "", errors.New("native executable belongs to a different installation; explicitly select that installation")
	}
	o.NativeBinary = target
	o.NativeLauncher = ""
	return o, hash, locator, nil
}

func verifyNativeObservation(harness string, o Options, target, digest, locator string) error {
	currentTarget, currentDigest, currentLocator, err := nativeSnapshot(harness, o)
	if err != nil {
		return err
	}
	if currentTarget != target || currentDigest != digest || currentLocator != locator {
		return errors.New("native executable changed during inspection; retry")
	}
	return nil
}
