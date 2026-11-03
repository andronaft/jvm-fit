package rules

import (
	"github.com/andronaft/jvm-fit/internal/detect"
	"github.com/andronaft/jvm-fit/internal/model"
	"github.com/andronaft/jvm-fit/internal/quantity"
)

func init() {
	register(Rule{
		ID:       "JVM005",
		Name:     "DefaultHeapSize",
		Severity: Low,
		Title:    "No heap setting: HotSpot uses only 25% of the limit",
		Explain: `With no -Xmx and no MaxRAMPercentage, HotSpot sets the max heap to 25% of
the memory limit. With a 2Gi limit the heap is 512Mi and about 1Gi is never
used: you pay for memory the JVM will not touch, or the heap is too small.

OpenJ9 has different defaults (50-75%), so this rule only applies to HotSpot.

Fix: -XX:MaxRAMPercentage=70 (or 75).`,
		Check: func(c *model.Container) *Finding {
			if c.HeapSource != model.HeapDefault || c.MemLimit == 0 || c.Detect.Runtime != detect.HotSpot || !c.ContainerSupport {
				return nil
			}
			return &Finding{
				Message: "No heap setting: max heap is " + quantity.Mi(c.HeapMax) + " of " + quantity.Human(c.MemLimit) + " (" + c.HeapDefaultNote + ")",
				Fix:     "add " + percentageFix(c),
			}
		},
	})
}
