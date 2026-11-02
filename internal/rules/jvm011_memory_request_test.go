package rules

import "testing"

func TestJVM011(t *testing.T) {
	checkRule(t, "JVM011", []ruleCase{
		{"request below limit", spec{memRequest: "512Mi", memLimit: "1Gi"}, true},
		{"request equals limit", spec{memRequest: "1Gi", memLimit: "1Gi"}, false},
		{"request in other units", spec{memRequest: "1024Mi", memLimit: "1Gi"}, false},
		{"no request (defaults to limit)", spec{memLimit: "1Gi"}, false},
		{"no limit", spec{memRequest: "1Gi"}, false},
	})
}
