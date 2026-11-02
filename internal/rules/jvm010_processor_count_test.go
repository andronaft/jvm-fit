package rules

import "testing"

func TestJVM010(t *testing.T) {
	checkRule(t, "JVM010", []ruleCase{
		{"4 on 1 CPU", spec{cpuLimit: "1", opts: "-XX:ActiveProcessorCount=4"}, true},
		{"2 on 500m", spec{cpuLimit: "500m", opts: "-XX:ActiveProcessorCount=2"}, true},
		{"2 on 1500m", spec{cpuLimit: "1500m", opts: "-XX:ActiveProcessorCount=2"}, false},
		{"2 on 2", spec{cpuLimit: "2", opts: "-XX:ActiveProcessorCount=2"}, false},
		{"no CPU limit", spec{opts: "-XX:ActiveProcessorCount=8"}, false},
		{"not set", spec{cpuLimit: "1"}, false},
	})
}
