package rules

import (
	"github.com/andronaft/jvm-fit/internal/model"
	"github.com/andronaft/jvm-fit/internal/quantity"
)

func init() {
	register(Rule{
		ID:       "JVM011",
		Severity: Low,
		Title:    "Memory request is lower than the limit",
		Explain: `The JVM grows the heap toward its max and rarely gives memory back to the OS.
A Java pod therefore ends up using close to its limit. With request < limit
the scheduler counts only the request, so under node memory pressure the pod
is among the first to be evicted.

Fix: set the memory request equal to the limit.`,
		Check: func(c *model.Container) *Finding {
			if c.MemRequest == 0 || c.MemLimit == 0 || c.MemRequest >= c.MemLimit {
				return nil
			}
			return &Finding{
				Message: "Memory request " + quantity.Human(c.MemRequest) + " < limit " + quantity.Human(c.MemLimit) + "; the JVM keeps memory it once used",
				Fix:     "set resources.requests.memory to " + quantity.Human(c.MemLimit),
			}
		},
	})
}
