package rules

import (
	"strings"
	"testing"
)

func TestJVM006(t *testing.T) {
	checkRule(t, "JVM006", []ruleCase{
		{"Xmx", spec{memLimit: "2Gi", opts: "-Xmx1g"}, true},
		{"MaxHeapSize", spec{memLimit: "2Gi", opts: "-XX:MaxHeapSize=1g"}, true},
		{"percentage", spec{memLimit: "2Gi", opts: "-XX:MaxRAMPercentage=70"}, false},
		{"nothing", spec{memLimit: "2Gi"}, false},
	})
}

func TestPercentageFix(t *testing.T) {
	tests := []struct {
		name string
		spec spec
		want string
	}{
		{"room for 70", spec{memLimit: "2Gi", opts: "-Xmx1g"}, "-XX:MaxRAMPercentage=70"},
		{"small limit", spec{memLimit: "832Mi", opts: "-Xmx400m"}, "-XX:MaxRAMPercentage=55"},
		{"no limit", spec{opts: "-Xmx1g"}, "-XX:MaxRAMPercentage=70"},
		{"over budget", spec{memLimit: "512Mi", opts: "-Xmx450m"}, "memory budget fix"},
	}
	r, _ := Get("JVM006")
	for _, tt := range tests {
		f := r.Check(tt.spec.build(t))
		if f == nil || !strings.Contains(f.Fix, tt.want) {
			t.Errorf("%s: fix = %+v, want %q", tt.name, f, tt.want)
		}
	}
}
