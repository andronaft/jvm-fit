package rules

import "testing"

func TestJVM002(t *testing.T) {
	checkRule(t, "JVM002", []ruleCase{
		{"Xmx equals limit", spec{memLimit: "1Gi", opts: "-Xmx1g"}, true},
		{"Xmx above limit", spec{memLimit: "1Gi", opts: "-Xmx2g"}, true},
		{"MaxHeapSize above limit", spec{memLimit: "512Mi", opts: "-XX:MaxHeapSize=600m"}, true},
		{"Xmx below limit", spec{memLimit: "1Gi", opts: "-Xmx600m"}, false},
		{"no limit", spec{opts: "-Xmx4g"}, false},
		{"percentage", spec{memLimit: "1Gi", opts: "-XX:MaxRAMPercentage=100"}, false},
	})
}
