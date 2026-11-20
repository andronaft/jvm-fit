package rules

import (
	"strings"
	"testing"
)

func TestJVM002(t *testing.T) {
	checkRule(t, "JVM002", []ruleCase{
		{"Xmx equals limit", spec{memLimit: "1Gi", opts: "-Xmx1g"}, true},
		{"Xmx above limit", spec{memLimit: "1Gi", opts: "-Xmx2g"}, true},
		{"MaxHeapSize above limit", spec{memLimit: "512Mi", opts: "-XX:MaxHeapSize=600m"}, true},
		{"Xmx below limit", spec{memLimit: "1Gi", opts: "-Xmx600m"}, false},
		{"no limit", spec{opts: "-Xmx4g"}, false},
		{"percentage", spec{memLimit: "1Gi", opts: "-XX:MaxRAMPercentage=100"}, false},
		{"MaxRAM above limit", spec{memLimit: "1Gi", opts: "-XX:MaxRAM=4g"}, true},
		{"MaxRAM and percentage", spec{memLimit: "1Gi", opts: "-XX:MaxRAM=2g -XX:MaxRAMPercentage=60"}, true},
		{"MaxRAM below limit", spec{memLimit: "1Gi", opts: "-XX:MaxRAM=512m"}, false},
		{"OpenJ9 ignores MaxRAM", spec{image: "ibm-semeru-runtimes:open-21-jre", memLimit: "1Gi", opts: "-XX:MaxRAM=4g"}, false},
	})
}

func TestJVM002MaxRAMMessage(t *testing.T) {
	r, _ := Get("JVM002")
	f := r.Check(spec{memLimit: "1Gi", opts: "-XX:MaxRAM=4g"}.build(t))
	if f == nil || !strings.Contains(f.Message, "Heap max 1024Mi (25% of -XX:MaxRAM=4g (JVM default))") ||
		!strings.HasPrefix(f.Fix, "remove -XX:MaxRAM=4g") || !strings.Contains(f.Fix, "-XX:MaxRAMPercentage=65") {
		t.Errorf("finding = %+v", f)
	}
}
