package rules

import "testing"

func TestJVM001(t *testing.T) {
	checkRule(t, "JVM001", []ruleCase{
		{"no limit, default heap", spec{}, true},
		{"no limit, percentage", spec{opts: "-XX:MaxRAMPercentage=75"}, true},
		{"no limit, OpenJ9", spec{image: "ibm-semeru-runtimes:open-17-jre"}, true},
		{"limit set", spec{memLimit: "1Gi"}, false},
		{"no limit but Xmx", spec{opts: "-Xmx512m"}, false},
		{"no limit but MaxRAM", spec{opts: "-XX:MaxRAM=1g"}, false},
	})
}
