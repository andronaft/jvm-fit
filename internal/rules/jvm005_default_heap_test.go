package rules

import (
	"strings"
	"testing"
)

func TestJVM005(t *testing.T) {
	checkRule(t, "JVM005", []ruleCase{
		{"no heap flags", spec{memLimit: "2Gi"}, true},
		{"only Xms", spec{memLimit: "2Gi", opts: "-Xms512m"}, true},
		{"percentage set", spec{memLimit: "2Gi", opts: "-XX:MaxRAMPercentage=70"}, false},
		{"Xmx set", spec{memLimit: "2Gi", opts: "-Xmx1g"}, false},
		{"OpenJ9 has its own defaults", spec{image: "ibm-semeru-runtimes:open-21-jre", memLimit: "2Gi"}, false},
		{"no limit is JVM001", spec{}, false},
		{"small heap from MaxRAM", spec{memLimit: "2Gi", opts: "-XX:MaxRAM=512m"}, true},
		{"MaxRAM above the limit is JVM002", spec{memLimit: "1Gi", opts: "-XX:MaxRAM=4g"}, false},
	})
}

func TestJVM005MaxRAMFix(t *testing.T) {
	r, _ := Get("JVM005")
	f := r.Check(spec{memLimit: "2Gi", opts: "-XX:MaxRAM=512m"}.build(t))
	if f == nil || !strings.Contains(f.Message, "128Mi of 2Gi (25% of -XX:MaxRAM=512m (JVM default))") ||
		!strings.HasPrefix(f.Fix, "remove -XX:MaxRAM=512m") || !strings.Contains(f.Fix, "and add -XX:MaxRAMPercentage=") {
		t.Errorf("finding = %+v", f)
	}
}
