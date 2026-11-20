package rules

import (
	"strings"
	"testing"

	"github.com/andronaft/jvm-fit/internal/loader"
	"gopkg.in/yaml.v3"
)

func TestJVM013(t *testing.T) {
	// 2Gi limit, 75% heap: 1536Mi + 350Mi non-heap leaves ~162Mi.
	tight := spec{memLimit: "2Gi", opts: "-XX:MaxRAMPercentage=75"}
	with := func(s spec, f func(*spec)) spec { f(&s); return s }
	env := func(name, value string) func(*spec) {
		return func(s *spec) { s.env = []loader.EnvVar{{Name: name, Value: value}} }
	}
	checkRule(t, "JVM013", []ruleCase{
		{"no CPU limit, tight", tight, true},
		{"8 CPUs, tight", with(tight, func(s *spec) { s.cpuLimit = "8" }), true},
		{"4 CPUs, tight", with(tight, func(s *spec) { s.cpuLimit = "3500m" }), true},
		{"over budget", spec{memLimit: "1Gi", opts: "-Xmx900m"}, true},
		{"ActiveProcessorCount=16 on 2 CPUs", with(tight, func(s *spec) { s.cpuLimit = "2"; s.opts += " -XX:ActiveProcessorCount=16" }), true},
		{"other LD_PRELOAD", with(tight, env("LD_PRELOAD", "/usr/lib/libfoo.so")), true},
		{"unknown app image", with(tight, func(s *spec) { s.image = "ghcr.io/acme/payments:1.4.2" }), true},
		{"2 CPUs", with(tight, func(s *spec) { s.cpuLimit = "2" }), false},
		{"ActiveProcessorCount=2, no limit", with(tight, func(s *spec) { s.opts += " -XX:ActiveProcessorCount=2" }), false},
		{"room to spare", spec{memLimit: "2Gi", opts: "-XX:MaxRAMPercentage=50"}, false},
		{"no memory limit", spec{opts: "-Xmx1g"}, false},
		{"alpine image", with(tight, func(s *spec) { s.image = "eclipse-temurin:21-jre-alpine" }), false},
		{"MALLOC_ARENA_MAX set", with(tight, env("MALLOC_ARENA_MAX", "2")), false},
		{"MALLOC_ARENA_MAX from ConfigMap", with(tight, func(s *spec) {
			s.env = []loader.EnvVar{{Name: "MALLOC_ARENA_MAX", ValueFrom: &yaml.Node{}}}
		}), false},
		{"jemalloc", with(tight, env("LD_PRELOAD", "/usr/lib/x86_64-linux-gnu/libjemalloc.so.2")), false},
		{"tcmalloc", with(tight, env("LD_PRELOAD", "libtcmalloc_minimal.so.4")), false},
		{"LD_PRELOAD from Secret", with(tight, func(s *spec) {
			s.env = []loader.EnvVar{{Name: "LD_PRELOAD", ValueFrom: &yaml.Node{}}}
		}), false},
		{"exported in sh -c", with(tight, func(s *spec) {
			s.command = []string{"sh", "-c", "export MALLOC_ARENA_MAX=2; exec java -jar app.jar"}
		}), false},
	})
}

func TestJVM013Message(t *testing.T) {
	r, _ := Get("JVM013")
	f := r.Check(spec{memLimit: "2Gi", cpuLimit: "8", opts: "-XX:MaxRAMPercentage=75"}.build(t))
	if f == nil {
		t.Fatal("no finding")
	}
	for _, want := range []string{"sees 8 CPUs", "~162Mi", "not measured"} {
		if !strings.Contains(f.Message, want) {
			t.Errorf("message %q does not contain %q", f.Message, want)
		}
	}
	if !strings.Contains(f.Fix, "MALLOC_ARENA_MAX=2") {
		t.Errorf("fix %q does not name MALLOC_ARENA_MAX=2", f.Fix)
	}
}
