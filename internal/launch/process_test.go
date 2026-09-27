package launch

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestProcessReplacementPreservesLiteralArgsCWDStatusAndSignals(t *testing.T) {
	if target := os.Getenv("MANDALORE_TEST_EXEC_TARGET"); target != "" {
		if err := replaceProcess(target, []string{"one space", "$(echo literal)", "`literal`", "$HOME"}, os.Environ()); err != nil {
			os.Exit(90)
		}
		return
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "native with spaces")
	body := "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\"\nexit 23\n"
	if e := os.WriteFile(script, []byte(body), 0700); e != nil {
		t.Fatal(e)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestProcessReplacementPreservesLiteralArgsCWDStatusAndSignals$")
	child.Dir = dir
	child.Env = append(os.Environ(), "MANDALORE_TEST_EXEC_TARGET="+script)
	raw, e := child.Output()
	exit, ok := e.(*exec.ExitError)
	if !ok || exit.ExitCode() != 23 {
		t.Fatal("status not preserved", e)
	}
	realDir, _ := filepath.EvalSymlinks(dir)
	want := realDir + "\none space\n$(echo literal)\n`literal`\n$HOME\n"
	if string(raw) != want {
		t.Fatalf("literal args/cwd lost: %q want %q", raw, want)
	}
	// Exec retains the child PID, so a signal to the launcher PID reaches native.
	if e = os.WriteFile(script, []byte("#!/bin/sh\necho ready\nexec sleep 30\n"), 0700); e != nil {
		t.Fatal(e)
	}
	child = exec.Command(os.Args[0], "-test.run=^TestProcessReplacementPreservesLiteralArgsCWDStatusAndSignals$")
	child.Env = append(os.Environ(), "MANDALORE_TEST_EXEC_TARGET="+script)
	out, e := child.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	if e = child.Start(); e != nil {
		t.Fatal(e)
	}
	defer child.Process.Kill()
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(out).ReadString('\n'); ready <- line }()
	select {
	case line := <-ready:
		if strings.TrimSpace(line) != "ready" {
			t.Fatal(line)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("native start timed out")
	}
	if e = child.Process.Signal(syscall.SIGTERM); e != nil {
		t.Fatal(e)
	}
	if e = child.Wait(); e == nil {
		t.Fatal("termination reported success")
	}
	if status, ok := child.ProcessState.Sys().(syscall.WaitStatus); !ok || status.Signal() != syscall.SIGTERM {
		t.Fatal("signal did not reach native process", child.ProcessState)
	}
}
