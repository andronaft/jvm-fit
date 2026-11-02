package rules

import "testing"

func TestJVM004(t *testing.T) {
	checkRule(t, "JVM004", []ruleCase{
		{"90%", spec{memLimit: "4Gi", opts: "-XX:MaxRAMPercentage=90.0"}, true},
		{"MaxRAMFraction=1", spec{memLimit: "4Gi", opts: "-XX:MaxRAMFraction=1"}, true},
		{"80% is fine", spec{memLimit: "4Gi", opts: "-XX:MaxRAMPercentage=80"}, false},
		{"75%", spec{memLimit: "4Gi", opts: "-XX:MaxRAMPercentage=75.0"}, false},
		{"Xmx wins over percentage", spec{memLimit: "4Gi", opts: "-XX:MaxRAMPercentage=90 -Xmx2g"}, false},
	})
}
