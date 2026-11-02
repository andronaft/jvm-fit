package rules

import (
	"github.com/andronaft/jvm-fit/internal/model"
	"github.com/andronaft/jvm-fit/internal/quantity"
)

func init() {
	register(Rule{
		ID:       "JVM002",
		Severity: Critical,
		Title:    "-Xmx is equal to or larger than the memory limit",
		Explain: `The heap alone can grow to the whole memory limit, and the JVM always needs
memory outside the heap too. When the heap fills up, the kernel kills the
container (OOMKilled) before the JVM can throw OutOfMemoryError.

Fix: make -Xmx smaller than the limit minus non-heap, or raise the limit.
Better: drop -Xmx and use -XX:MaxRAMPercentage so the heap follows the limit.`,
		Check: func(c *model.Container) *Finding {
			if c.HeapSource != model.HeapFromXmx || c.MemLimit == 0 || c.HeapMax < c.MemLimit {
				return nil
			}
			return &Finding{
				Message: "Heap " + c.Flags.Flags.MaxHeap.Raw + " is not smaller than the memory limit " + quantity.Human(c.MemLimit) + " → OOMKilled when the heap fills up",
				Fix:     "set -Xmx below " + quantity.Mi(c.MemLimit-c.NonHeap) + " or raise the memory limit to " + quantity.Human(model.RecommendedLimit(c.HeapMax, c.NonHeap)),
			}
		},
	})
}
