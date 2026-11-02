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
