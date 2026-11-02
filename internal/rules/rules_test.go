package rules

import (
	"fmt"
	"testing"
)

func TestAllRulesRegistered(t *testing.T) {
	all := All()
	if len(all) != 12 {
		t.Fatalf("got %d rules, want 12", len(all))
	}
	for i, r := range all {
		if want := fmt.Sprintf("JVM%03d", i+1); r.ID != want {
			t.Errorf("rule %d is %s, want %s", i, r.ID, want)
		}
		if r.Title == "" || r.Explain == "" || r.Check == nil {
			t.Errorf("%s is missing title, explain or check", r.ID)
		}
	}
}

func TestRunSortsBySeverity(t *testing.T) {
	fs := Run(spec{memLimit: "1Gi", cpuLimit: "500m", opts: "-Xmx900m"}.build(t))
	if len(fs) == 0 {
		t.Fatal("no findings")
	}
	for i := 1; i < len(fs); i++ {
		if fs[i].Severity > fs[i-1].Severity {
			t.Errorf("findings not sorted: %v", fs)
		}
	}
	if fs[0].RuleID != "JVM003" {
		t.Errorf("first finding = %s, want JVM003", fs[0].RuleID)
	}
}

func TestParseSeverity(t *testing.T) {
	for _, s := range []string{"critical", "HIGH", "Medium", "low", "info"} {
		if _, err := ParseSeverity(s); err != nil {
			t.Errorf("ParseSeverity(%q): %v", s, err)
		}
	}
	if _, err := ParseSeverity("urgent"); err == nil {
		t.Error("want error")
	}
}
