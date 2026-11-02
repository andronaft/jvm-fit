package rules

import "testing"

func TestJVM008(t *testing.T) {
	checkRule(t, "JVM008", []ruleCase{
		{"500m", spec{cpuLimit: "500m"}, true},
		{"0.25", spec{cpuLimit: "0.25"}, true},
		{"1 core", spec{cpuLimit: "1"}, false},
		{"no limit", spec{}, false},
	})
}
