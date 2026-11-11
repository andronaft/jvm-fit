package rules

import (
	"fmt"

	"github.com/andronaft/jvm-fit/internal/model"
	"github.com/andronaft/jvm-fit/internal/quantity"
)

func init() {
	register(Rule{
		ID:       "JVM003",
		Name:     "MemoryBudgetExceeded",
		Severity: High,
		Title:    "Heap plus non-heap estimate does not fit in the memory limit",
		Explain: `The JVM uses memory outside the heap: metaspace (loaded classes), a stack per
thread, the JIT code cache, direct buffers (Netty, NIO) and GC structures. For
a typical Spring Boot service this is 300-400Mi. The container is killed when
heap + non-heap goes over the limit, even if the heap looks fine.

The non-heap number is an estimate. Change it with --nonheap-estimate.

Fix: leave room for non-heap (MaxRAMPercentage 70-75 for most services) or
raise the memory limit.`,
		Refs: []string{refNMT, refK8sResources},
		Check: func(c *model.Container) *Finding {
			if c.MemLimit == 0 || c.HeapMax == 0 || c.Needed() <= c.MemLimit {
				return nil
			}
			// JVM002 already covers a heap that alone does not fit.
			if c.HeapSource == model.HeapFromXmx && c.HeapMax >= c.MemLimit {
				return nil
			}
			over := c.Needed() - c.MemLimit
			newLimit := model.RecommendedLimit(c.HeapMax, c.NonHeap)
			fix := fmt.Sprintf("raise the memory limit to %s", quantity.Human(newLimit))
			if c.HeapSource == model.HeapFromXms {
				// -Xms sets the heap here, so a percentage would not lower it.
				return &Finding{
					Message: fmt.Sprintf("Heap + non-heap needs ~%s, limit is %s (over by ~%s) → OOMKilled under load",
						quantity.Mi(c.Needed()), quantity.Mi(c.MemLimit), quantity.Mi(over)),
					Fix: fmt.Sprintf("%s raises the max heap: lower it or %s", c.Flags.Flags.InitialHeap.Raw, fix),
				}
			}
			if p := model.RecommendedPercentage(newLimit, c.NonHeap); p > 0 {
				fix = fmt.Sprintf("-XX:MaxRAMPercentage=%d and memory limit %s", p, quantity.Human(newLimit))
			}
			if p := model.RecommendedPercentage(c.MemLimit, c.NonHeap); p >= 50 {
				fix += fmt.Sprintf(", or keep the limit and use -XX:MaxRAMPercentage=%d", p)
			}
			return &Finding{
				Message: fmt.Sprintf("Heap + non-heap needs ~%s, limit is %s (over by ~%s) → OOMKilled under load",
					quantity.Mi(c.Needed()), quantity.Mi(c.MemLimit), quantity.Mi(over)),
				Fix: fix,
			}
		},
	})
}
