package flags

import (
	"reflect"
	"testing"
)

const (
	ki = int64(1) << 10
	mi = int64(1) << 20
	gi = int64(1) << 30
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"512m", 512 * mi, false},
		{"512M", 512 * mi, false},
		{"1g", gi, false},
		{"1G", gi, false},
		{"2048k", 2048 * ki, false},
		{"2048K", 2048 * ki, false},
		{"1t", 1 << 40, false},
		{"1073741824", gi, false},
		{"0", 0, false},
		{"", 0, true},
		{"m", 0, true},
		{"1.5g", 0, true},
		{"1gb", 0, true},
		{"-1g", 0, true},
		{"1Gi", 0, true},
		{"99999999999999999999", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseSize(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseSize(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestApply(t *testing.T) {
	src := Source{Name: "test"}
	tests := []struct {
		name  string
		in    string
		check func(t *testing.T, f Flags)
	}{
		{"Xmx", "-Xmx900m", func(t *testing.T, f Flags) { wantI(t, f.MaxHeap, 900*mi) }},
		{"MaxHeapSize", "-XX:MaxHeapSize=2g", func(t *testing.T, f Flags) { wantI(t, f.MaxHeap, 2*gi) }},
		{"last Xmx wins", "-Xmx1g -Xmx512m", func(t *testing.T, f Flags) { wantI(t, f.MaxHeap, 512*mi) }},
		{"Xms", "-Xms256m", func(t *testing.T, f Flags) { wantI(t, f.InitialHeap, 256*mi) }},
		{"Xss", "-Xss512k", func(t *testing.T, f Flags) { wantI(t, f.ThreadStack, 512*ki) }},
		{"ThreadStackSize is KiB", "-XX:ThreadStackSize=1024", func(t *testing.T, f Flags) { wantI(t, f.ThreadStack, mi) }},
		{"ThreadStackSize with unit", "-XX:ThreadStackSize=2m", func(t *testing.T, f Flags) { wantI(t, f.ThreadStack, 2*mi) }},
		{"MaxRAMPercentage", "-XX:MaxRAMPercentage=75.0", func(t *testing.T, f Flags) { wantF(t, f.MaxRAMPercentage, 75) }},
		{"MaxRAMPercentage int", "-XX:MaxRAMPercentage=80", func(t *testing.T, f Flags) { wantF(t, f.MaxRAMPercentage, 80) }},
		{"MaxRAMFraction", "-XX:MaxRAMFraction=2", func(t *testing.T, f Flags) { wantF(t, f.MaxRAMPercentage, 50) }},
		{"InitialRAMPercentage", "-XX:InitialRAMPercentage=50", func(t *testing.T, f Flags) { wantF(t, f.InitialRAMPercentage, 50) }},
		{"MaxMetaspaceSize", "-XX:MaxMetaspaceSize=256m", func(t *testing.T, f Flags) { wantI(t, f.MaxMetaspace, 256*mi) }},
		{"ReservedCodeCacheSize", "-XX:ReservedCodeCacheSize=240m", func(t *testing.T, f Flags) { wantI(t, f.ReservedCodeCache, 240*mi) }},
		{"MaxDirectMemorySize", "-XX:MaxDirectMemorySize=128m", func(t *testing.T, f Flags) { wantI(t, f.MaxDirectMemory, 128*mi) }},
		{"MaxRAM", "-XX:MaxRAM=4g", func(t *testing.T, f Flags) { wantI(t, f.MaxRAM, 4*gi) }},
		{"ActiveProcessorCount", "-XX:ActiveProcessorCount=4", func(t *testing.T, f Flags) {
			if !f.ActiveProcessorCount.Set || f.ActiveProcessorCount.V != 4 {
				t.Errorf("ActiveProcessorCount = %+v", f.ActiveProcessorCount)
			}
		}},
		{"ActiveProcessorCount -1 resets", "-XX:ActiveProcessorCount=4 -XX:ActiveProcessorCount=-1", func(t *testing.T, f Flags) {
			if f.ActiveProcessorCount.Set {
				t.Errorf("ActiveProcessorCount should be unset")
			}
		}},
		{"container support off", "-XX:-UseContainerSupport", func(t *testing.T, f Flags) {
			if !f.UseContainerSupport.Set || f.UseContainerSupport.V {
				t.Errorf("UseContainerSupport = %+v", f.UseContainerSupport)
			}
		}},
		{"G1", "-XX:+UseG1GC", func(t *testing.T, f Flags) { wantS(t, f.GC, "G1") }},
		{"Parallel", "-XX:+UseParallelGC", func(t *testing.T, f Flags) { wantS(t, f.GC, "Parallel") }},
		{"ZGC", "-XX:+UseZGC", func(t *testing.T, f Flags) { wantS(t, f.GC, "Z") }},
		{"last GC wins", "-XX:+UseSerialGC -XX:+UseG1GC", func(t *testing.T, f Flags) { wantS(t, f.GC, "G1") }},
		{"GC turned off", "-XX:+UseG1GC -XX:-UseG1GC", func(t *testing.T, f Flags) {
			if f.GC.Set {
				t.Errorf("GC should be unset, got %+v", f.GC)
			}
		}},
		{"OpenJ9 gcpolicy", "-Xgcpolicy:balanced", func(t *testing.T, f Flags) { wantS(t, f.GC, "balanced") }},
		{"unrelated flags ignored", "-Dspring.profiles.active=prod -XX:+HeapDumpOnOutOfMemoryError -verbose:gc", func(t *testing.T, f Flags) {
			if len(f.Seen) != 0 {
				t.Errorf("Seen = %v, want none", f.Seen)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f Flags
			if w := f.Apply(Split(tt.in), src); len(w) > 0 {
				t.Fatalf("warnings: %v", w)
			}
			tt.check(t, f)
		})
	}
}

func TestApplyInvalid(t *testing.T) {
	for _, in := range []string{"-Xmx", "-Xmx1.5g", "-Xmxlots", "-XX:MaxRAMPercentage=abc", "-XX:MaxRAMPercentage=150", "-XX:ActiveProcessorCount=0", "-XX:MaxRAMFraction=0"} {
		var f Flags
		if w := f.Apply([]string{in}, Source{Name: "test"}); len(w) != 1 {
			t.Errorf("Apply(%q) warnings = %v, want 1", in, w)
		}
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"-Xmx1g  -Xms1g", []string{"-Xmx1g", "-Xms1g"}},
		{`-Dname="a b" -Xss1m`, []string{"-Dname=a b", "-Xss1m"}},
		{`'-Da=x y' z`, []string{"-Da=x y", "z"}},
		{`a\ b`, []string{"a b"}},
		{"\n -Xmx1g\n", []string{"-Xmx1g"}},
		{"", nil},
	}
	for _, tt := range tests {
		if got := Split(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Split(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestExpandKube(t *testing.T) {
	env := map[string]string{"HEAP": "-Xmx1g"}
	tests := map[string]string{
		"$(HEAP)":           "-Xmx1g",
		"java $(HEAP) -jar": "java -Xmx1g -jar",
		"$$(HEAP)":          "$(HEAP)",
		"$(MISSING)":        "$(MISSING)",
		"$HEAP":             "$HEAP",
	}
	for in, want := range tests {
		if got := ExpandKube(in, env); got != want {
			t.Errorf("ExpandKube(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCollect(t *testing.T) {
	tests := []struct {
		name       string
		in         Input
		wantHeap   int64
		wantSource Source
	}{
		{
			name:       "JAVA_TOOL_OPTIONS",
			in:         Input{Env: []EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-Xmx900m"}}},
			wantHeap:   900 * mi,
			wantSource: Source{Name: "JAVA_TOOL_OPTIONS"},
		},
		{
			name: "command line beats JAVA_TOOL_OPTIONS",
			in: Input{
				Env:     []EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-Xmx900m"}},
				Command: []string{"java", "-Xmx512m", "-jar", "app.jar", "-Xmx2g"},
			},
			wantHeap:   512 * mi,
			wantSource: Source{Name: "command"},
		},
		{
			name: "JDK_JAVA_OPTIONS beats JAVA_TOOL_OPTIONS",
			in: Input{Env: []EnvVar{
				{Name: "JDK_JAVA_OPTIONS", Value: "-Xmx700m"},
				{Name: "JAVA_TOOL_OPTIONS", Value: "-Xmx900m"},
			}},
			wantHeap:   700 * mi,
			wantSource: Source{Name: "JDK_JAVA_OPTIONS"},
		},
		{
			name:       "JAVA_OPTS is assumed",
			in:         Input{Env: []EnvVar{{Name: "JAVA_OPTS", Value: "-Xmx1g"}}},
			wantHeap:   gi,
			wantSource: Source{Name: "JAVA_OPTS", Assumed: true},
		},
		{
			name: "JAVA_OPTS used by sh -c is not assumed",
			in: Input{
				Env:     []EnvVar{{Name: "JAVA_OPTS", Value: "-Xmx1g"}},
				Command: []string{"sh", "-c", "exec java $JAVA_OPTS -jar /app.jar"},
			},
			wantHeap:   gi,
			wantSource: Source{Name: "command"},
		},
		{
			name: "kube $(VAR) in args",
			in: Input{
				Env:     []EnvVar{{Name: "HEAP_OPTS", Value: "-Xmx640m"}},
				Command: []string{"java"},
				Args:    []string{"$(HEAP_OPTS)", "-cp", "app.jar", "com.example.Main"},
			},
			wantHeap:   640 * mi,
			wantSource: Source{Name: "command"},
		},
		{
			name:       "args to image entrypoint",
			in:         Input{Args: []string{"-Xmx300m", "--server.port=8080"}},
			wantHeap:   300 * mi,
			wantSource: Source{Name: "command"},
		},
		{
			name:     "app args after main class are ignored",
			in:       Input{Command: []string{"/opt/java/bin/java", "-cp", "/app", "Main", "-Xmx4g"}},
			wantHeap: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Collect(tt.in)
			got := res.Flags.MaxHeap
			if got.V != tt.wantHeap {
				t.Fatalf("MaxHeap = %d, want %d (%+v)", got.V, tt.wantHeap, got)
			}
			if tt.wantHeap != 0 && got.Source != tt.wantSource {
				t.Errorf("Source = %+v, want %+v", got.Source, tt.wantSource)
			}
		})
	}
}

func TestCollectUnresolved(t *testing.T) {
	res := Collect(Input{Env: []EnvVar{{Name: "JAVA_TOOL_OPTIONS", FromRef: true}, {Name: "DB_PASSWORD", FromRef: true}}})
	if !reflect.DeepEqual(res.Unresolved, []string{"JAVA_TOOL_OPTIONS"}) {
		t.Errorf("Unresolved = %v", res.Unresolved)
	}
}

func TestCollectCustomAssumedEnv(t *testing.T) {
	res := Collect(Input{
		Env:        []EnvVar{{Name: "JAVA_OPTS", Value: "-Xmx1g"}, {Name: "APP_JVM", Value: "-Xmx2g"}},
		AssumedEnv: []string{"APP_JVM"},
	})
	if res.Flags.MaxHeap.V != 2*gi {
		t.Errorf("MaxHeap = %d, want 2g from APP_JVM", res.Flags.MaxHeap.V)
	}
}

func wantI(t *testing.T, v Value[int64], want int64) {
	t.Helper()
	if !v.Set || v.V != want {
		t.Errorf("got %+v, want %d", v, want)
	}
}

func wantF(t *testing.T, v Value[float64], want float64) {
	t.Helper()
	if !v.Set || v.V != want {
		t.Errorf("got %+v, want %v", v, want)
	}
}

func wantS(t *testing.T, v Value[string], want string) {
	t.Helper()
	if !v.Set || v.V != want {
		t.Errorf("got %+v, want %q", v, want)
	}
}
