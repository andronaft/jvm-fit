package rules

import (
	"fmt"
	"strings"

	"github.com/andronaft/jvm-fit/internal/model"
	"github.com/andronaft/jvm-fit/internal/quantity"
)

const (
	// arenaMinCPUs: JVM013 fires when the JVM sees at least this many CPUs,
	// or has no CPU limit and sees the whole node. Below it the JVM starts few
	// GC, JIT and ForkJoinPool threads.
	arenaMinCPUs = 4
	// arenaSpare: JVM013 fires when the limit leaves less than this above
	// heap max plus the non-heap estimate. A few extra arenas take tens to
	// hundreds of MiB of RSS.
	arenaSpare = 256 * quantity.MiB

	envMallocArenaMax = "MALLOC_ARENA_MAX"
	refMallopt        = "https://man7.org/linux/man-pages/man3/mallopt.3.html"
	refMallocTunables = "https://sourceware.org/glibc/manual/latest/html_node/Memory-Allocation-Tunables.html"
)

func init() {
	register(Rule{
		ID:       "JVM013",
		Name:     "MallocArenaMaxNotSet",
		Severity: Low,
		Title:    "MALLOC_ARENA_MAX not set on glibc with many CPUs and a tight limit",
		Explain: `glibc malloc gives threads that allocate at the same time separate arenas,
up to 8 × the CPU cores (64Mi of address space each on 64-bit). The cap comes
from the cores the process may run on (all cores of the node unless the pod
is pinned to a cpuset), not from the CPU limit. The JVM allocates native
memory from many threads (GC, JIT, NIO, JNI, zip and other compression
libraries), so on a many-core node it can touch many arenas, and the
container RSS grows past the heap plus what NMT reports. The more threads
the JVM starts (it sizes GC, JIT and ForkJoinPool threads from the CPUs it
sees), the more arenas get used.

This is a hint. jvm-fit does not measure it: the growth depends on the app,
the node and the glibc version. It fires only when all of these hold:
- the image is not Alpine/musl (by its name or tag);
- MALLOC_ARENA_MAX is not set in the container env, and LD_PRELOAD does not
  load jemalloc or tcmalloc;
- the JVM sees 4 or more CPUs, or has no CPU limit (it sees the whole node);
- the memory limit leaves less than 256Mi above heap max plus the non-heap
  estimate.

Fix: set MALLOC_ARENA_MAX=2 (or 4) in the container env. The trade-off is
more malloc lock contention between threads. Other options: an Alpine/musl
image, or jemalloc/tcmalloc via LD_PRELOAD. If the image or an envFrom
source already sets MALLOC_ARENA_MAX, ignore this finding.`,
		Refs: []string{refMallopt, refMallocTunables, refNMT},
		Check: func(c *model.Container) *Finding {
			if c.Detect.Musl || c.MemLimit == 0 || c.HeapMax == 0 {
				return nil
			}
			if c.CPUs != 0 && c.CPUs < arenaMinCPUs {
				return nil
			}
			if otherMalloc(c) {
				return nil
			}
			spare := c.MemLimit - c.Needed()
			if spare >= arenaSpare {
				return nil
			}
			cpus := "no CPU limit, so the JVM sees every core of the node"
			if c.CPUs != 0 {
				cpus = fmt.Sprintf("the JVM sees %d CPUs (%s)", c.CPUs, c.CPUSource)
			}
			room := "nothing"
			if spare > 0 {
				room = "only ~" + quantity.Mi(spare)
			}
			return &Finding{
				Message: fmt.Sprintf("MALLOC_ARENA_MAX is not set and %s; glibc can create up to 8 malloc arenas per node core, and the limit leaves %s above heap + non-heap for that growth (hint, not measured)", cpus, room),
				Fix:     "env MALLOC_ARENA_MAX=2 (or 4) to cap glibc malloc arenas; ignore if the image or envFrom already sets it",
			}
		},
	})
}

// otherMalloc reports whether the container env or command sets
// MALLOC_ARENA_MAX, preloads jemalloc or tcmalloc, or sets LD_PRELOAD from a
// ConfigMap or Secret (so jvm-fit cannot tell).
func otherMalloc(c *model.Container) bool {
	for _, e := range c.Container.Env {
		switch e.Name {
		case envMallocArenaMax:
			return true
		case "LD_PRELOAD":
			if e.ValueFrom != nil || namesAllocator(e.Value) {
				return true
			}
		}
	}
	for _, s := range append(append([]string{c.Container.Image}, c.Container.Command...), c.Container.Args...) {
		if strings.Contains(s, envMallocArenaMax) || namesAllocator(s) {
			return true
		}
	}
	return false
}

func namesAllocator(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "jemalloc") || strings.Contains(s, "tcmalloc")
}
