package rules

import (
	"fmt"

	"github.com/andronaft/jvm-fit/internal/model"
	"github.com/andronaft/jvm-fit/internal/quantity"
)

func init() {
	register(Rule{
		ID:       "JVM002",
		Name:     "XmxOverLimit",
		Severity: Critical,
		Title:    "Max heap is equal to or larger than the memory limit",
		Explain: `The heap alone can grow to the whole memory limit, and the JVM always needs
memory outside the heap too. When the heap fills up, the kernel kills the
container (OOMKilled) before the JVM can throw OutOfMemoryError.

-XX:MaxRAM does the same: the JVM sizes the heap from MaxRAM instead of the
limit, also when MaxRAM is larger. With -XX:MaxRAM=4g and a 1Gi limit the
default heap is 1Gi.

Fix: make -Xmx smaller than the limit minus non-heap, or raise the limit.
Better: drop -Xmx (and -XX:MaxRAM) and use -XX:MaxRAMPercentage so the heap
follows the limit.`,
		Refs: []string{refJavaMan, refK8sResources},
		Check: func(c *model.Container) *Finding {
			if !heapOverLimit(c) {
				return nil
			}
			if c.HeapSource != model.HeapFromXmx {
				fix := dropMaxRAM(c)
				if p := model.RecommendedPercentage(c.MemLimit, c.NonHeap); p > 0 {
					fix += fmt.Sprintf(" and use -XX:MaxRAMPercentage=%d", p)
				}
				return &Finding{
					Message: "Heap max " + quantity.Mi(c.HeapMax) + " (" + c.HeapDefaultNote + ") is not smaller than the memory limit " + quantity.Human(c.MemLimit) + " → OOMKilled when the heap fills up",
					Fix:     fix,
				}
			}
			return &Finding{
				Message: "Heap " + c.Flags.Flags.MaxHeap.Raw + " is not smaller than the memory limit " + quantity.Human(c.MemLimit) + " → OOMKilled when the heap fills up",
				Fix:     "set -Xmx below " + quantity.Mi(c.MemLimit-c.NonHeap) + " or raise the memory limit to " + quantity.Human(model.RecommendedLimit(c.HeapMax, c.NonHeap)),
			}
		},
	})
}
