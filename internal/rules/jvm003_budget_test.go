package rules

import (
	"strings"
	"testing"
)

func TestJVM003(t *testing.T) {
	checkRule(t, "JVM003", []ruleCase{
		{"Xmx 900m in 1Gi", spec{memLimit: "1Gi", opts: "-Xmx900m"}, true},
		{"90% of 1Gi", spec{memLimit: "1Gi", opts: "-XX:MaxRAMPercentage=90"}, true},
		{"default heap in 256Mi", spec{memLimit: "256Mi"}, true},
		{"Xmx 600m in 1Gi", spec{memLimit: "1Gi", opts: "-Xmx600m"}, false},
		{"70% of 2Gi", spec{memLimit: "2Gi", opts: "-XX:MaxRAMPercentage=70"}, false},
		{"Xmx over limit is JVM002", spec{memLimit: "1Gi", opts: "-Xmx1g"}, false},
		{"no limit", spec{opts: "-Xmx4g"}, false},
		{"Xms 768m alone in 1Gi", spec{memLimit: "1Gi", opts: "-Xms768m"}, true},
		{"Xms 768m over 50% of 1Gi", spec{memLimit: "1Gi", opts: "-Xms768m -XX:MaxRAMPercentage=50"}, true},
		{"Xms 256m alone in 2Gi", spec{memLimit: "2Gi", opts: "-Xms256m"}, false},
		{"Xms with a smaller Xmx", spec{memLimit: "1Gi", opts: "-Xms768m -Xmx600m"}, false},
	})
}

func TestJVM003Message(t *testing.T) {
	r, _ := Get("JVM003")
	f := r.Check(spec{memLimit: "1Gi", cpuLimit: "500m", opts: "-Xmx900m"}.build(t))
	if !strings.Contains(f.Message, "~1250Mi") || !strings.Contains(f.Message, "~226Mi") {
		t.Errorf("message = %q", f.Message)
	}
	if !strings.Contains(f.Fix, "-XX:MaxRAMPercentage=70 and memory limit 1.25Gi") {
		t.Errorf("fix = %q", f.Fix)
	}
}

func TestJVM003XmsFix(t *testing.T) {
	r, _ := Get("JVM003")
	f := r.Check(spec{memLimit: "1Gi", opts: "-Xms768m"}.build(t))
	if !strings.Contains(f.Message, "~1118Mi") {
		t.Errorf("message = %q", f.Message)
	}
	if !strings.Contains(f.Fix, "-Xms768m raises the max heap") || strings.Contains(f.Fix, "MaxRAMPercentage") {
		t.Errorf("fix = %q", f.Fix)
	}
}
