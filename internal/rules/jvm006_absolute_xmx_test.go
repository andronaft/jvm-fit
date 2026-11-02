package rules

import "testing"

func TestJVM006(t *testing.T) {
	checkRule(t, "JVM006", []ruleCase{
		{"Xmx", spec{memLimit: "2Gi", opts: "-Xmx1g"}, true},
		{"MaxHeapSize", spec{memLimit: "2Gi", opts: "-XX:MaxHeapSize=1g"}, true},
		{"percentage", spec{memLimit: "2Gi", opts: "-XX:MaxRAMPercentage=70"}, false},
		{"nothing", spec{memLimit: "2Gi"}, false},
	})
}
