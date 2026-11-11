package model

import (
	"testing"

	"github.com/andronaft/jvm-fit/internal/loader"
)

func container(image, mem, cpu, opts string) loader.Container {
	c := loader.Container{Name: "app", Image: image, Resources: loader.Resources{Limits: map[string]string{}}}
	if mem != "" {
		c.Resources.Limits["memory"] = mem
	}
	if cpu != "" {
		c.Resources.Limits["cpu"] = cpu
	}
	if opts != "" {
		c.Env = []loader.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: opts}}
	}
	return c
}

// TestEvidence checks the model against docs/evidence/results.md: the same
// limits and flags must give the same heap, CPU count and GC as a real JVM.
// Rows above the Docker VM memory (2g, 4g) are left out: there the JVM saw
// the VM memory, not the limit.
func TestEvidence(t *testing.T) {
	tests := []struct {
		image  string
		mem    string
		cpu    string
		flags  string
		heapMi int64
		cpus   int
		gc     string
	}{
		// eclipse-temurin 11, 17 and 21 gave the same values.
		{"eclipse-temurin:17-jre", "256Mi", "500m", "", 126, 1, "Serial"},
		{"eclipse-temurin:17-jre", "512Mi", "500m", "", 128, 1, "Serial"},
		{"eclipse-temurin:17-jre", "512Mi", "1", "", 128, 1, "Serial"},
		{"eclipse-temurin:17-jre", "1Gi", "1", "", 256, 1, "Serial"},
		{"eclipse-temurin:17-jre", "1Gi", "1", "-XX:MaxRAMPercentage=75.0", 768, 1, "Serial"},
		{"eclipse-temurin:17-jre", "1Gi", "1", "-XX:ActiveProcessorCount=2", 256, 2, "Serial"},
		{"eclipse-temurin:17-jre", "1Gi", "2", "", 256, 2, "Serial"},
		{"eclipse-temurin:11-jre", "1Gi", "2", "", 256, 2, "Serial"},
		{"eclipse-temurin:21-jre", "1Gi", "1", "", 256, 1, "Serial"},
		// OpenJ9 (ibm-semeru-runtimes:open-21-jre).
		{"ibm-semeru-runtimes:open-21-jre", "256Mi", "500m", "", 128, 1, "gencon"},
		{"ibm-semeru-runtimes:open-21-jre", "512Mi", "500m", "", 256, 1, "gencon"},
		{"ibm-semeru-runtimes:open-21-jre", "1Gi", "1", "", 512, 1, "gencon"},
		{"ibm-semeru-runtimes:open-21-jre", "1Gi", "1", "-XX:MaxRAMPercentage=75.0", 768, 1, "gencon"},
		{"ibm-semeru-runtimes:open-21-jre", "1Gi", "1", "-XX:ActiveProcessorCount=2", 512, 2, "gencon"},
	}
	for _, tt := range tests {
		name := tt.image + " " + tt.mem + " " + tt.cpu + " " + tt.flags
		t.Run(name, func(t *testing.T) {
			c := Build(loader.Workload{}, container(tt.image, tt.mem, tt.cpu, tt.flags), nil, Options{})
			if c == nil {
				t.Fatal("not detected as Java")
			}
			if got := c.HeapMax / mi; got != tt.heapMi {
				t.Errorf("heap = %dMi, want %dMi", got, tt.heapMi)
			}
			if c.CPUs != tt.cpus {
				t.Errorf("cpus = %d, want %d", c.CPUs, tt.cpus)
			}
			if c.GC != tt.gc {
				t.Errorf("gc = %q, want %q", c.GC, tt.gc)
			}
		})
	}
}

func TestContainerSupportOff(t *testing.T) {
	c := Build(loader.Workload{}, container("eclipse-temurin:17", "1Gi", "1", "-XX:-UseContainerSupport"), nil, Options{})
	if c.ContainerSupport {
		t.Fatal("container support should be off")
	}
	// Measured: the JVM sizes the heap from host memory, not the limit.
	if c.HeapMax != 0 || c.CPUs != 0 {
		t.Errorf("heap %d, cpus %d: want unknown (depends on the node)", c.HeapMax, c.CPUs)
	}
}

func TestNoLimit(t *testing.T) {
	c := Build(loader.Workload{}, container("eclipse-temurin:17", "", "", ""), nil, Options{})
	if c.HeapMax != 0 || c.HeapSource != HeapDefault || c.HeapPercent != 25 {
		t.Errorf("heap = %d %s %v", c.HeapMax, c.HeapSource, c.HeapPercent)
	}
	if c.GC != "" {
		t.Errorf("gc = %q, want unknown", c.GC)
	}
}

func TestServerClass(t *testing.T) {
	c := Build(loader.Workload{}, container("eclipse-temurin:17", "2Gi", "2", ""), nil, Options{})
	if c.GC != "G1" {
		t.Errorf("gc = %q, want G1", c.GC)
	}
	c = Build(loader.Workload{}, container("openjdk:8-jre", "2Gi", "2", ""), nil, Options{})
	if c.GC != "Parallel" {
		t.Errorf("JDK 8 gc = %q, want Parallel", c.GC)
	}
}

func TestXmxAndBudget(t *testing.T) {
	c := Build(loader.Workload{}, container("eclipse-temurin:17", "1Gi", "500m", "-Xmx900m"), nil, Options{})
	if c.HeapMax != 900*mi || c.HeapSource != HeapFromXmx {
		t.Fatalf("heap = %d %s", c.HeapMax, c.HeapSource)
	}
	if c.Needed() != 1250*mi {
		t.Errorf("needed = %dMi, want 1250Mi", c.Needed()/mi)
	}
}

func TestMaxRAM(t *testing.T) {
	c := Build(loader.Workload{}, container("eclipse-temurin:17", "", "", "-XX:MaxRAM=2g -XX:MaxRAMPercentage=50"), nil, Options{})
	if c.HeapMax != 1024*mi {
		t.Errorf("heap = %dMi, want 1024Mi", c.HeapMax/mi)
	}
}

func TestOpenJ9DefaultLarge(t *testing.T) {
	// From the OpenJ9 docs; not yet measured (see docs/evidence).
	for mem, want := range map[string]int64{"1536Mi": 1024, "4Gi": 3072} {
		c := Build(loader.Workload{}, container("ibm-semeru-runtimes:open-17-jre", mem, "", ""), nil, Options{})
		if c.HeapMax/mi != want {
			t.Errorf("%s: heap = %dMi, want %dMi", mem, c.HeapMax/mi, want)
		}
	}
}

func TestNotJava(t *testing.T) {
	if Build(loader.Workload{}, container("nginx:1.27", "1Gi", "1", ""), nil, Options{}) != nil {
		t.Error("nginx detected as Java")
	}
}

func TestBadQuantity(t *testing.T) {
	c := Build(loader.Workload{}, container("eclipse-temurin:17", "lots", "", ""), nil, Options{})
	if len(c.ResourceErrors) != 1 {
		t.Errorf("errors = %v", c.ResourceErrors)
	}
}

func TestRecommend(t *testing.T) {
	if got := RecommendedLimit(900*mi, 350*mi); got != 1280*mi {
		t.Errorf("RecommendedLimit = %dMi, want 1280Mi", got/mi)
	}
	tests := map[int64]int{1024 * mi: 65, 1280 * mi: 70, 4096 * mi: 75, 300 * mi: 0}
	for limit, want := range tests {
		if got := RecommendedPercentage(limit, 350*mi); got != want {
			t.Errorf("RecommendedPercentage(%dMi) = %d, want %d", limit/mi, got, want)
		}
	}
}

// Measured on JDK 17 with -XX:MaxRAM=1g: -Xms768m gives MaxHeapSize 768M,
// with or without -XX:MaxRAMPercentage=50; without -Xms it is 256M.
func TestXmsRaisesMaxHeap(t *testing.T) {
	tests := []struct {
		opts   string
		heapMi int64
		source HeapSource
	}{
		{"-Xms768m", 768, HeapFromXms},
		{"-Xms768m -XX:MaxRAMPercentage=50", 768, HeapFromXms},
		{"-XX:InitialHeapSize=768m", 768, HeapFromXms},
		{"-Xms128m", 256, HeapDefault},
		{"-Xms256m -XX:MaxRAMPercentage=50", 512, HeapFromPercentage},
		{"-Xms768m -Xmx600m", 600, HeapFromXmx},
	}
	for _, tt := range tests {
		c := Build(loader.Workload{}, container("eclipse-temurin:17", "1Gi", "1", tt.opts), nil, Options{})
		if c.HeapMax != tt.heapMi*mi || c.HeapSource != tt.source {
			t.Errorf("%s: heap = %dMi %s, want %dMi %s", tt.opts, c.HeapMax/mi, c.HeapSource, tt.heapMi, tt.source)
		}
	}
	// OpenJ9 has its own rules; the model leaves it alone.
	c := Build(loader.Workload{}, container("ibm-semeru-runtimes:open-17-jre", "1Gi", "1", "-Xms768m"), nil, Options{})
	if c.HeapSource == HeapFromXms {
		t.Errorf("OpenJ9: heap source = %s", c.HeapSource)
	}
}
