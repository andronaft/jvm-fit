package rules

import (
	"fmt"
	"math"

	"github.com/andronaft/jvm-fit/internal/model"
	"github.com/andronaft/jvm-fit/internal/quantity"
)

func init() {
	register(Rule{
		ID:       "JVM010",
		Name:     "ProcessorCountOverLimit",
		Severity: Medium,
		Title:    "ActiveProcessorCount is higher than the CPU limit",
		Explain: `-XX:ActiveProcessorCount sets how many CPUs the JVM thinks it has. GC threads,
JIT threads and the common ForkJoinPool are sized from it. When it is above
the CPU limit, these threads compete for less CPU time than they expect and
the container is throttled.

Fix: make ActiveProcessorCount at most the CPU limit (rounded up), or raise
the limit.`,
		Refs: []string{refContainerSupport, refJavaMan},
		Check: func(c *model.Container) *Finding {
			apc := c.Flags.Flags.ActiveProcessorCount
			if !apc.Set || c.CPULimit == 0 {
				return nil
			}
			max := int(math.Ceil(float64(c.CPULimit) / 1000))
			if apc.V <= max {
				return nil
			}
			return &Finding{
				Message: fmt.Sprintf("ActiveProcessorCount=%d but the CPU limit is %s → more threads than CPU time, throttling", apc.V, quantity.CPU(c.CPULimit)),
				Fix:     fmt.Sprintf("-XX:ActiveProcessorCount=%d, or raise the CPU limit to %d", max, apc.V),
			}
		},
	})
}
