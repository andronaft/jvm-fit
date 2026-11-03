package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func run(args ...string) (string, error) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var e exitError
	if errors.As(err, &e) {
		return e.code
	}
	return 2
}

func TestScanText(t *testing.T) {
	out, err := run("scan", "../../testdata")
	if code := exitCode(err); code != 1 {
		t.Fatalf("exit code = %d (%v), want 1", code, err)
	}
	for _, want := range []string{
		"payments-api (Deployment, container: app)",
		"needed                ~1250Mi   ❌ over limit by ~226Mi",
		"HIGH     JVM003",
		"in 3 of 4 workloads",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "nginx") {
		t.Error("non-Java container in the report")
	}
}

func TestScanGoodFile(t *testing.T) {
	out, err := run("scan", "../../testdata/orders-good.yaml")
	if err != nil {
		t.Fatalf("err = %v\n%s", err, out)
	}
	if !strings.Contains(out, "No findings") {
		t.Errorf("output:\n%s", out)
	}
}

func TestScanJSON(t *testing.T) {
	out, err := run("scan", "../../testdata", "--format", "json", "--fail-on", "none")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	var doc struct {
		Summary struct {
			Workloads      int            `json:"workloads"`
			JavaContainers int            `json:"javaContainers"`
			BySeverity     map[string]int `json:"bySeverity"`
		} `json:"summary"`
		Containers []struct {
			Findings []struct {
				Rule     string `json:"rule"`
				Severity string `json:"severity"`
			} `json:"findings"`
		} `json:"containers"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if doc.Summary.Workloads != 4 || doc.Summary.JavaContainers != 5 {
		t.Errorf("summary = %+v", doc.Summary)
	}
	if doc.Summary.BySeverity["high"] != 3 {
		t.Errorf("bySeverity = %v", doc.Summary.BySeverity)
	}
}

func TestFailOn(t *testing.T) {
	tests := map[string]int{"critical": 0, "high": 1, "info": 1, "none": 0}
	for level, want := range tests {
		_, err := run("scan", "../../testdata", "--fail-on", level)
		if got := exitCode(err); got != want {
			t.Errorf("--fail-on %s: exit %d, want %d", level, got, want)
		}
	}
}

func TestNonHeapEstimate(t *testing.T) {
	out, _ := run("scan", "../../testdata/orders-good.yaml", "--nonheap-estimate", "600Mi")
	if !strings.Contains(out, "JVM003") {
		t.Errorf("a larger non-heap estimate should make orders not fit:\n%s", out)
	}
}

func TestFlagsEnv(t *testing.T) {
	out, _ := run("scan", "../../testdata/batch-cronjob.yaml", "--flags-env", "EXTRA")
	if !strings.Contains(out, "JVM009") {
		t.Errorf("output:\n%s", out)
	}
}

func TestBadInput(t *testing.T) {
	for _, args := range [][]string{
		{"scan", "../../testdata", "--format", "xml"},
		{"scan", "../../testdata", "--fail-on", "urgent"},
		{"scan", "../../testdata", "--nonheap-estimate", "lots"},
		{"scan", "does-not-exist"},
		{"explain", "JVM999"},
	} {
		if _, err := run(args...); exitCode(err) != 2 {
			t.Errorf("%v: err = %v, want a usage error", args, err)
		}
	}
}

func TestExplain(t *testing.T) {
	out, err := run("explain", "jvm007")
	if err != nil || !strings.Contains(out, "SerialGC") {
		t.Errorf("explain: %v\n%s", err, out)
	}
	out, err = run("explain")
	if err != nil || strings.Count(out, "\n") != 12 {
		t.Errorf("rule list: %v\n%s", err, out)
	}
}

func TestScanSARIFToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.sarif")
	out, err := run("scan", "../../testdata", "--format", "sarif", "-o", path)
	if exitCode(err) != 1 {
		t.Fatalf("exit = %d (%v), want 1: the exit code does not depend on the format", exitCode(err), err)
	}
	if out != "" {
		t.Errorf("stdout should be empty with --output, got %q", out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"version": "2.1.0"`) {
		t.Errorf("not SARIF: %.200s", data)
	}
}

func TestOutputNotWrittenOnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")
	if _, err := run("scan", "does-not-exist", "--format", "json", "-o", path); exitCode(err) != 2 {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("output file should not exist")
	}
}

func needTool(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not in PATH", name)
	}
}

func TestScanHelmChart(t *testing.T) {
	needTool(t, "helm")
	chart := "../../internal/loader/testdata/spring-app"
	out, err := run("scan", chart)
	if exitCode(err) != 1 {
		t.Fatalf("exit = %d (%v)\n%s", exitCode(err), err, out)
	}
	for _, want := range []string{
		"spring-app (Deployment, container: spring-app)",
		"templates/deployment.yaml  limits: memory=768Mi cpu=750m",
		"HIGH     JVM003",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %q:\n%s", want, out)
		}
	}

	// The same chart through --helm, with values that make it fit.
	out, err = run("scan", "--helm", chart,
		"--set", "resourcesPreset=large", "--set", "startupProbe.enabled=true",
		"--set", "javaOpts=-XX:MaxRAMPercentage=70 -XX:+UseG1GC")
	if exitCode(err) != 0 {
		t.Errorf("exit = %d (%v)\n%s", exitCode(err), err, out)
	}
}

func TestScanKustomize(t *testing.T) {
	if _, err := exec.LookPath("kustomize"); err != nil {
		needTool(t, "kubectl")
	}
	out, _ := run("scan", "../../internal/loader/testdata/kustomize/overlays/prod")
	if !strings.Contains(out, "prod/billing (Deployment, container: app)") || !strings.Contains(out, "JVM004") {
		t.Errorf("output:\n%s", out)
	}
}

func TestScanNoInput(t *testing.T) {
	if _, err := run("scan"); exitCode(err) != 2 {
		t.Errorf("err = %v, want a usage error", err)
	}
}
