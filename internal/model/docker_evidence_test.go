//go:build evidence

package model

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/andronaft/jvm-fit/internal/loader"
)

// TestDockerEvidence starts real JVMs in Docker with the same limits and
// flags as a manifest and checks that the model predicts the same max heap,
// CPU count and GC. Run with:
//
//	go test -tags evidence -run TestDockerEvidence -v ./internal/model/
//
// Cases with a memory limit above the Docker host memory are skipped: the
// JVM then sees the host memory, not the limit.
func TestDockerEvidence(t *testing.T) {
	hostMem := dockerHostMemory(t)
	images := []string{"eclipse-temurin:11-jre", "eclipse-temurin:17-jre", "eclipse-temurin:21-jre"}
	cases := []struct {
		mem, cpu, flags string
	}{
		{"256Mi", "500m", ""},
		{"512Mi", "500m", ""},
		{"1Gi", "1", ""},
		{"1Gi", "2", ""},
		{"1Gi", "1", "-XX:MaxRAMPercentage=75.0"},
		{"1Gi", "1", "-XX:ActiveProcessorCount=2"},
		{"1Gi", "1", "-XX:+UseG1GC"},
		{"1Gi", "1", "-Xmx600m"},
		{"2Gi", "2", ""},
		{"2Gi", "1", "-XX:ActiveProcessorCount=2"},
		{"4Gi", "4", ""},
		{"4Gi", "4", "-XX:MaxRAMPercentage=70"},
	}
	for _, image := range images {
		for _, tc := range cases {
			name := fmt.Sprintf("%s/%s/%s/%s", image, tc.mem, tc.cpu, tc.flags)
			t.Run(name, func(t *testing.T) {
				c := container(image, tc.mem, tc.cpu, tc.flags)
				m := Build(loader.Workload{}, c, nil, Options{})
				if m.MemLimit > hostMem {
					t.Skipf("limit %d is above Docker host memory %d", m.MemLimit, hostMem)
				}
				heap, cpus, gc := runHotSpot(t, image, m.MemLimit, m.CPULimit, tc.flags)
				// The JVM aligns the heap to its region/page size; allow 2%.
				if diff := heap - m.HeapMax; diff*50 > m.HeapMax || -diff*50 > m.HeapMax {
					t.Errorf("heap: JVM %dMi, model %dMi", heap/mi, m.HeapMax/mi)
				}
				if cpus != m.CPUs {
					t.Errorf("cpus: JVM %d, model %d", cpus, m.CPUs)
				}
				if gc != m.GC {
					t.Errorf("gc: JVM %s, model %s", gc, m.GC)
				}
			})
		}
	}
}

func dockerHostMemory(t *testing.T) int64 {
	out, err := exec.Command("docker", "info", "--format", "{{.MemTotal}}").Output()
	if err != nil {
		t.Skipf("docker is not available: %v", err)
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		t.Fatalf("docker info: %v", err)
	}
	return n
}

var (
	reHeap = regexp.MustCompile(`\bMaxHeapSize\s+=\s+(\d+)`)
	reAPC  = regexp.MustCompile(`\bActiveProcessorCount\s+=\s+(-?\d+)`)
	reGC   = regexp.MustCompile(`\bUse(Serial|Parallel|G1|Z|Shenandoah)GC\s+=\s+true`)
	reCPU  = regexp.MustCompile(`Effective CPU Count: (\d+)`)
)

func runHotSpot(t *testing.T, image string, mem, milliCPU int64, flags string) (heap int64, cpus int, gc string) {
	t.Helper()
	args := []string{"run", "--rm", fmt.Sprintf("--memory=%d", mem), fmt.Sprintf("--cpus=%s", strconv.FormatFloat(float64(milliCPU)/1000, 'f', -1, 64)), image,
		"sh", "-c", "java " + flags + " -XX:+PrintFlagsFinal -version; java " + flags + " -XshowSettings:system -version"}
	var out bytes.Buffer
	cmd := exec.Command("docker", args...)
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("docker %v: %v\n%s", args, err, out.String())
	}
	s := out.String()
	m := reHeap.FindStringSubmatch(s)
	if m == nil {
		t.Fatalf("no MaxHeapSize in output:\n%s", s)
	}
	heap, _ = strconv.ParseInt(m[1], 10, 64)
	if m := reAPC.FindStringSubmatch(s); m != nil && m[1] != "-1" {
		cpus, _ = strconv.Atoi(m[1])
	} else if m := reCPU.FindStringSubmatch(s); m != nil {
		cpus, _ = strconv.Atoi(m[1])
	}
	if m := reGC.FindStringSubmatch(s); m != nil {
		gc = m[1]
	}
	return heap, cpus, gc
}
