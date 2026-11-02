package rules

import "testing"

func TestJVM007(t *testing.T) {
	checkRule(t, "JVM007", []ruleCase{
		{"500m CPU", spec{memLimit: "2Gi", cpuLimit: "500m"}, true},
		{"1 CPU", spec{memLimit: "4Gi", cpuLimit: "1"}, true},
		{"2 CPUs but 1Gi", spec{memLimit: "1Gi", cpuLimit: "2"}, true},
		{"ActiveProcessorCount=2 but 1Gi", spec{memLimit: "1Gi", cpuLimit: "1", opts: "-XX:ActiveProcessorCount=2"}, true},
		{"2 CPUs and 2Gi", spec{memLimit: "2Gi", cpuLimit: "2"}, false},
		{"G1 set", spec{memLimit: "1Gi", cpuLimit: "500m", opts: "-XX:+UseG1GC"}, false},
		{"Serial set on purpose", spec{memLimit: "1Gi", cpuLimit: "500m", opts: "-XX:+UseSerialGC"}, false},
		{"no limits", spec{}, false},
		{"OpenJ9", spec{image: "ibm-semeru-runtimes:open-21-jre", memLimit: "1Gi", cpuLimit: "500m"}, false},
	})
}
