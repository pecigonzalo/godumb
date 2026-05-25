package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBuildInvokesGoBuild(t *testing.T) {
	tmpDir := t.TempDir()
	sourcePath := writeMinimalGoDumbMain(t, tmpDir)
	outputPath := filepath.Join(tmpDir, "hello-bin")

	var capturedArgs []string
	var generatedGoPath string
	goBuildRunner = func(args []string, _, _ io.Writer) error {
		capturedArgs = append([]string{}, args...)
		generatedGoPath = args[len(args)-1]

		data, err := os.ReadFile(generatedGoPath)
		if err != nil {
			return err
		}
		if !strings.Contains(string(data), "package main") {
			t.Fatalf("generated source missing package declaration:\n%s", string(data))
		}
		return nil
	}
	t.Cleanup(func() {
		goBuildRunner = runGoBuild
	})

	var stdout, stderr bytes.Buffer
	if err := runBuild([]string{"-o", outputPath, sourcePath}, &stdout, &stderr); err != nil {
		t.Fatalf("runBuild failed: %v", err)
	}

	wantPrefix := []string{"build", "-o", outputPath}
	if len(capturedArgs) != 4 {
		t.Fatalf("unexpected arg count: got %d args: %v", len(capturedArgs), capturedArgs)
	}
	for i := range wantPrefix {
		if capturedArgs[i] != wantPrefix[i] {
			t.Fatalf("unexpected go arg %d: got %q want %q", i, capturedArgs[i], wantPrefix[i])
		}
	}

	if _, err := os.Stat(generatedGoPath); !os.IsNotExist(err) {
		t.Fatalf("temporary generated go file should be removed, stat err: %v", err)
	}
}

func TestRunBuildRequiresSingleInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runBuild([]string{}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing input")
	}

	err = runBuild([]string{"a.gdb", "b.gdb"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for multiple inputs")
	}
}

func TestRunRunBuildsAndExecutes(t *testing.T) {
	tmpDir := t.TempDir()
	sourcePath := writeMinimalGoDumbMain(t, tmpDir)

	var (
		capturedBuildArgs []string
		generatedGoPath   string
		generatedBinPath  string
		runPath           string
		runArgs           []string
		runStdin          string
	)

	goBuildRunner = func(args []string, _, _ io.Writer) error {
		capturedBuildArgs = append([]string{}, args...)
		if len(args) != 4 || args[0] != "build" || args[1] != "-o" {
			return fmt.Errorf("unexpected build args: %v", args)
		}

		generatedBinPath = args[2]
		generatedGoPath = args[3]
		return os.WriteFile(generatedBinPath, []byte("fake-binary"), 0o755)
	}
	binaryRunner = func(path string, args []string, stdin io.Reader, stdout, _ io.Writer) error {
		runPath = path
		runArgs = append([]string{}, args...)

		data, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		runStdin = string(data)

		_, err = io.WriteString(stdout, "program output\n")
		return err
	}
	t.Cleanup(func() {
		goBuildRunner = runGoBuild
		binaryRunner = runBinary
	})

	var stdout, stderr bytes.Buffer
	if err := runRun([]string{sourcePath, "arg1", "arg2"}, strings.NewReader("stdin payload"), &stdout, &stderr); err != nil {
		t.Fatalf("runRun failed: %v", err)
	}

	if len(capturedBuildArgs) != 4 {
		t.Fatalf("unexpected build args length: %d (%v)", len(capturedBuildArgs), capturedBuildArgs)
	}
	if capturedBuildArgs[0] != "build" || capturedBuildArgs[1] != "-o" {
		t.Fatalf("unexpected build args: %v", capturedBuildArgs)
	}
	if runPath != generatedBinPath {
		t.Fatalf("binary runner used wrong path: got %q want %q", runPath, generatedBinPath)
	}
	if strings.Join(runArgs, ",") != "arg1,arg2" {
		t.Fatalf("unexpected run args: %v", runArgs)
	}
	if runStdin != "stdin payload" {
		t.Fatalf("unexpected stdin payload: %q", runStdin)
	}
	if stdout.String() != "program output\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}

	if _, err := os.Stat(generatedGoPath); !os.IsNotExist(err) {
		t.Fatalf("temporary go file should be removed, stat err: %v", err)
	}
	if _, err := os.Stat(generatedBinPath); !os.IsNotExist(err) {
		t.Fatalf("temporary binary should be removed, stat err: %v", err)
	}
}

func TestRunRunRequiresInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runRun([]string{}, strings.NewReader(""), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing input")
	}
}

func writeMinimalGoDumbMain(t *testing.T, dir string) string {
	t.Helper()

	sourcePath := filepath.Join(dir, "hello.gdb")
	source := `package
main
func
main
(
)
{
}
`

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	return sourcePath
}
