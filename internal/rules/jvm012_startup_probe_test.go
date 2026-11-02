package rules

import (
	"testing"

	"github.com/andronaft/jvm-fit/internal/loader"
)

func TestJVM012(t *testing.T) {
	checkRule(t, "JVM012", []ruleCase{
		{"liveness after 10s", spec{liveness: &loader.Probe{InitialDelaySeconds: 10}}, true},
		{"liveness with no delay", spec{liveness: &loader.Probe{}}, true},
		{"liveness after 60s", spec{liveness: &loader.Probe{InitialDelaySeconds: 60}}, false},
		{"startup probe", spec{liveness: &loader.Probe{InitialDelaySeconds: 10}, startup: &loader.Probe{FailureThreshold: 30}}, false},
		{"no liveness", spec{}, false},
	})
}
