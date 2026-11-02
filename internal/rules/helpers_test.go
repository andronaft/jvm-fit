package rules

import (
	"testing"

	"github.com/andronaft/jvm-fit/internal/loader"
	"github.com/andronaft/jvm-fit/internal/model"
)

// spec describes a container for rule tests.
type spec struct {
	image      string
	memLimit   string
	memRequest string
	cpuLimit   string
	opts       string // JAVA_TOOL_OPTIONS
	liveness   *loader.Probe
	startup    *loader.Probe
}

func (s spec) build(t *testing.T) *model.Container {
	t.Helper()
	img := s.image
	if img == "" {
		img = "eclipse-temurin:17-jre"
	}
	c := loader.Container{
		Name:  "app",
		Image: img,
		Resources: loader.Resources{
			Limits:   map[string]string{},
			Requests: map[string]string{},
		},
		LivenessProbe: s.liveness,
		StartupProbe:  s.startup,
	}
	if s.memLimit != "" {
		c.Resources.Limits["memory"] = s.memLimit
	}
	if s.memRequest != "" {
		c.Resources.Requests["memory"] = s.memRequest
	}
	if s.cpuLimit != "" {
		c.Resources.Limits["cpu"] = s.cpuLimit
	}
	if s.opts != "" {
		c.Env = []loader.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: s.opts}}
	}
	m := model.Build(loader.Workload{Kind: "Deployment", Name: "test"}, c, nil, model.Options{})
	if m == nil {
		t.Fatal("container not detected as Java")
	}
	return m
}

type ruleCase struct {
	name string
	spec spec
	want bool
}

func checkRule(t *testing.T, id string, cases []ruleCase) {
	t.Helper()
	r, ok := Get(id)
	if !ok {
		t.Fatalf("rule %s is not registered", id)
	}
	var good, bad bool
	for _, tc := range cases {
		if tc.want {
			bad = true
		} else {
			good = true
		}
		t.Run(tc.name, func(t *testing.T) {
			f := r.Check(tc.spec.build(t))
			if (f != nil) != tc.want {
				t.Fatalf("finding = %+v, want finding: %v", f, tc.want)
			}
			if f != nil && (f.Message == "" || f.Fix == "") {
				t.Errorf("finding needs a message and a fix: %+v", f)
			}
		})
	}
	if !good || !bad {
		t.Errorf("rule %s needs at least one good and one bad example", id)
	}
}
