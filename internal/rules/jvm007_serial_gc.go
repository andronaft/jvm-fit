package rules

import (
	"github.com/andronaft/jvm-fit/internal/model"
)

func init() {
	register(Rule{
		ID:       "JVM007",
		Severity: Medium,
		Title:    "The JVM will pick SerialGC",
		Explain: `When no GC is set, HotSpot picks G1 only on a "server class machine": at least
2 CPUs and at least 1792Mi of memory. In a container both come from the
limits. With a CPU limit below 2 cores, or a memory limit below 1792Mi, the
JVM picks SerialGC: one GC thread and long stop-the-world pauses on bigger heaps.

Measured on JDK 11, 17 and 21: 1Gi + 2 CPUs gives SerialGC, and so does
-XX:ActiveProcessorCount=2 with 1Gi.

Fix: set the GC explicitly (-XX:+UseG1GC, or -XX:+UseParallelGC for small
batch jobs), or raise the limits.`,
		Check: func(c *model.Container) *Finding {
			if c.GCExplicit || c.GC != "Serial" {
				return nil
			}
			return &Finding{
				Message: c.GCReason + ", no GC set → SerialGC (single-threaded GC)",
				Fix:     "add -XX:+UseG1GC, or raise the limits to ≥ 2 CPUs and ≥ 1792Mi",
			}
		},
	})
}
