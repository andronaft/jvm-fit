package rules

import "testing"

func TestJVM005(t *testing.T) {
	checkRule(t, "JVM005", []ruleCase{
		{"no heap flags", spec{memLimit: "2Gi"}, true},
		{"only Xms", spec{memLimit: "2Gi", opts: "-Xms512m"}, true},
		{"percentage set", spec{memLimit: "2Gi", opts: "-XX:MaxRAMPercentage=70"}, false},
		{"Xmx set", spec{memLimit: "2Gi", opts: "-Xmx1g"}, false},
		{"OpenJ9 has its own defaults", spec{image: "ibm-semeru-runtimes:open-21-jre", memLimit: "2Gi"}, false},
		{"no limit is JVM001", spec{}, false},
	})
}
