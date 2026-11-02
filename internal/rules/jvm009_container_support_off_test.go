package rules

import "testing"

func TestJVM009(t *testing.T) {
	checkRule(t, "JVM009", []ruleCase{
		{"off", spec{memLimit: "1Gi", opts: "-XX:-UseContainerSupport"}, true},
		{"on explicitly", spec{memLimit: "1Gi", opts: "-XX:+UseContainerSupport"}, false},
		{"default", spec{memLimit: "1Gi"}, false},
		{"off then on", spec{memLimit: "1Gi", opts: "-XX:-UseContainerSupport -XX:+UseContainerSupport"}, false},
	})
}
