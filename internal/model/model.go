// Package model works out how the JVM sees its container: max heap, CPU
// count, the GC it picks and whether heap plus non-heap fits in the limit.
//
// The numbers follow HotSpot and OpenJ9 ergonomics. They are checked against
// real JVMs in docs/evidence.
package model

import (
	"fmt"
	"math"

	"github.com/andronaft/jvm-fit/internal/detect"
	"github.com/andronaft/jvm-fit/internal/flags"
	"github.com/andronaft/jvm-fit/internal/loader"
	"github.com/andronaft/jvm-fit/internal/quantity"
)

const (
	mi = quantity.MiB

	// DefaultNonHeap is the non-heap estimate for a typical Spring Boot service:
	// metaspace, thread stacks, code cache, direct buffers and GC structures.
	DefaultNonHeap = 350 * mi

	// HotSpot ergonomics (src/hotspot/share/runtime/arguments.cpp).
	hotspotMaxRAMPercentage = 25.0
	hotspotMinRAMPercentage = 50.0
	// hotspotDefaultMaxHeap is the MaxHeapSize flag default on 64-bit
	// (96M scaled for word size). Measured: 126M with a 256m limit.
	hotspotDefaultMaxHeap = 126 * mi
	// serverClassMemory: with less memory (or fewer than 2 CPUs) HotSpot
	// picks SerialGC. 2G - 256M, see os::is_server_class_machine.
	serverClassMemory = 1792 * mi
)

// HeapSource says how the max heap was decided.
type HeapSource string

const (
	HeapFromXmx        HeapSource = "xmx"
	HeapFromPercentage HeapSource = "percentage"
	HeapDefault        HeapSource = "default"
)

// Options tune the model.
type Options struct {
	// NonHeap is the memory the JVM needs outside the heap.
	NonHeap int64
}

// Container is the model of one Java container.
type Container struct {
	Workload  loader.Workload
	Container loader.Container
	Detect    detect.Result
	Flags     flags.Result

	MemLimit   int64 // bytes, 0 = no limit
	MemRequest int64 // bytes, 0 = not set
	CPULimit   int64 // millicores, 0 = no limit
	CPURequest int64 // millicores, 0 = not set
	// ResourceErrors lists quantities that could not be parsed.
	ResourceErrors []string

	ContainerSupport bool

	// HeapMax is the max heap in bytes, 0 if it depends on the node.
	HeapMax     int64
	HeapSource  HeapSource
	HeapPercent float64 // percentage used when HeapSource is not xmx
	// HeapDefaultNote explains a default heap, e.g. "25% of the limit".
	HeapDefaultNote string

	// CPUs is what the JVM uses as its processor count, 0 if it depends on the node.
	CPUs      int
	CPUSource string

	// GC is the collector name ("G1", "Serial", "gencon"...), "" if unknown.
	GC         string
	GCExplicit bool
	// GCReason says why the JVM picks this GC when it is not set explicitly.
	GCReason string

	NonHeap int64
}

// Needed is heap plus non-heap, or 0 when the heap is unknown.
func (c *Container) Needed() int64 {
	if c.HeapMax == 0 {
		return 0
	}
	return c.HeapMax + c.NonHeap
}

// Build models one container. It returns nil for containers that do not look like Java.
func Build(w loader.Workload, lc loader.Container, assumedEnv []string, opts Options) *Container {
	det := detect.Container(lc)
	if !det.Java {
		return nil
	}
	if opts.NonHeap == 0 {
		opts.NonHeap = DefaultNonHeap
	}
	c := &Container{Workload: w, Container: lc, Detect: det, NonHeap: opts.NonHeap}

	in := flags.Input{Command: lc.Command, Args: lc.Args, AssumedEnv: assumedEnv}
	for _, e := range lc.Env {
		in.Env = append(in.Env, flags.EnvVar{Name: e.Name, Value: e.Value, FromRef: e.ValueFrom != nil})
	}
	c.Flags = flags.Collect(in)

	c.MemLimit = c.bytes(lc.Resources.Limits, "limits.memory")
	c.MemRequest = c.bytes(lc.Resources.Requests, "requests.memory")
	c.CPULimit = c.milli(lc.Resources.Limits, "limits.cpu")
	c.CPURequest = c.milli(lc.Resources.Requests, "requests.cpu")

	f := &c.Flags.Flags
	c.ContainerSupport = !f.UseContainerSupport.Set || f.UseContainerSupport.V
	c.heap()
	c.cpus()
	c.gc()
	return c
}

func (c *Container) bytes(m map[string]string, field string) int64 {
	s, ok := m[fieldName(field)]
	if !ok {
		return 0
	}
	v, err := quantity.ParseBytes(s)
	if err != nil {
		c.ResourceErrors = append(c.ResourceErrors, fmt.Sprintf("%s: %v", field, err))
	}
	return v
}

func (c *Container) milli(m map[string]string, field string) int64 {
	s, ok := m[fieldName(field)]
	if !ok {
		return 0
	}
	v, err := quantity.ParseMilliCPU(s)
	if err != nil {
		c.ResourceErrors = append(c.ResourceErrors, fmt.Sprintf("%s: %v", field, err))
	}
	return v
}

func fieldName(f string) string {
	for i := len(f) - 1; i >= 0; i-- {
		if f[i] == '.' {
			return f[i+1:]
		}
	}
	return f
}

// visibleMemory is the memory the JVM treats as physical RAM, 0 if unknown.
func (c *Container) visibleMemory() int64 {
	f := &c.Flags.Flags
	mem := int64(0)
	if c.ContainerSupport {
		mem = c.MemLimit
	}
	if f.MaxRAM.Set && (mem == 0 || f.MaxRAM.V < mem) {
		mem = f.MaxRAM.V
	}
	return mem
}

func (c *Container) heap() {
	f := &c.Flags.Flags
	if f.MaxHeap.Set {
		c.HeapMax, c.HeapSource = f.MaxHeap.V, HeapFromXmx
		return
	}
	mem := c.visibleMemory()
	if f.MaxRAMPercentage.Set {
		c.HeapSource, c.HeapPercent = HeapFromPercentage, f.MaxRAMPercentage.V
		c.HeapMax = int64(float64(mem) * c.HeapPercent / 100)
		return
	}
	c.HeapSource = HeapDefault
	if mem == 0 {
		c.HeapPercent = hotspotMaxRAMPercentage
		if c.Detect.Runtime == detect.OpenJ9 {
			c.HeapPercent = 0
		}
		return
	}
	if c.Detect.Runtime == detect.OpenJ9 {
		c.HeapMax, c.HeapDefaultNote = openj9DefaultHeap(mem)
		c.HeapPercent = math.Round(float64(c.HeapMax) / float64(mem) * 100)
		return
	}
	c.HeapMax, c.HeapPercent, c.HeapDefaultNote = hotspotDefaultHeap(mem, f)
}

// hotspotDefaultHeap mirrors Arguments::set_heap_size when no heap flag is set.
func hotspotDefaultHeap(mem int64, f *flags.Flags) (int64, float64, string) {
	minPct := hotspotMinRAMPercentage
	if f.MinRAMPercentage.Set {
		minPct = f.MinRAMPercentage.V
	}
	small := int64(float64(mem) * minPct / 100)
	if small < hotspotDefaultMaxHeap {
		return small, minPct, fmt.Sprintf("%.0f%% of the limit (MinRAMPercentage, small container)", minPct)
	}
	heap := int64(float64(mem) * hotspotMaxRAMPercentage / 100)
	if heap < hotspotDefaultMaxHeap {
		return hotspotDefaultMaxHeap, hotspotMaxRAMPercentage, "JVM minimum default of 126Mi"
	}
	return heap, hotspotMaxRAMPercentage, "25% of the limit (JVM default)"
}

// openj9DefaultHeap follows the OpenJ9 container defaults: 50% below 1 GiB,
// limit - 512 MiB from 1 to 2 GiB, 75% above 2 GiB.
func openj9DefaultHeap(mem int64) (int64, string) {
	switch {
	case mem < 1024*mi:
		return mem / 2, "50% of the limit (OpenJ9 default below 1Gi)"
	case mem <= 2048*mi:
		return mem - 512*mi, "limit - 512Mi (OpenJ9 default for 1-2Gi)"
	default:
		return mem * 3 / 4, "75% of the limit (OpenJ9 default above 2Gi)"
	}
}

func (c *Container) cpus() {
	f := &c.Flags.Flags
	switch {
	case f.ActiveProcessorCount.Set:
		c.CPUs, c.CPUSource = f.ActiveProcessorCount.V, "-XX:ActiveProcessorCount"
	case c.ContainerSupport && c.CPULimit > 0:
		c.CPUs = int(math.Ceil(float64(c.CPULimit) / 1000))
		c.CPUSource = "CPU limit " + quantity.CPU(c.CPULimit)
	}
}

func (c *Container) gc() {
	f := &c.Flags.Flags
	if f.GC.Set {
		c.GC, c.GCExplicit = f.GC.V, true
		return
	}
	if c.Detect.Runtime == detect.OpenJ9 {
		c.GC, c.GCReason = "gencon", "OpenJ9 default"
		return
	}
	server := "G1"
	if c.Detect.Major == 8 {
		server = "Parallel"
	}
	mem := c.visibleMemory()
	switch {
	case c.CPUs == 1:
		c.GC, c.GCReason = "Serial", c.CPUSource+" → the JVM sees 1 CPU"
	case mem > 0 && mem < serverClassMemory:
		c.GC, c.GCReason = "Serial", fmt.Sprintf("memory limit %s is below %s", quantity.Human(mem), quantity.Mi(serverClassMemory))
	case c.CPUs >= 2 && mem >= serverClassMemory:
		c.GC, c.GCReason = server, "the JVM sees 2+ CPUs and 1792Mi+ memory"
	}
}

// RecommendedLimit is the memory limit that fits heap plus non-heap, rounded
// up to 64Mi.
func RecommendedLimit(heap, nonHeap int64) int64 {
	const step = 64 * mi
	need := heap + nonHeap
	return (need + step - 1) / step * step
}

// RecommendedPercentage is the MaxRAMPercentage that leaves nonHeap free in
// the limit, rounded down to 5 and capped at 75.
func RecommendedPercentage(limit, nonHeap int64) int {
	if limit <= nonHeap {
		return 0
	}
	p := int(float64(limit-nonHeap) / float64(limit) * 100)
	p = p / 5 * 5
	if p > 75 {
		p = 75
	}
	return p
}
