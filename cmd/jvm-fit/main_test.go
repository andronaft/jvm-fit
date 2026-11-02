package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
