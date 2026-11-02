package rules

import (
	"github.com/andronaft/jvm-fit/internal/model"
	"github.com/andronaft/jvm-fit/internal/quantity"
)

func init() {
	register(Rule{
		ID:       "JVM008",
		Severity: Medium,
		Title:    "CPU limit below 1 core",
		Explain: `The JVM is busy with many threads at once during startup (class loading, JIT
compilation) and during GC. A CPU limit below 1 core is enforced by CFS
quota: the container is throttled for the rest of each 100ms period. Startup
gets slow (and probes start failing) and GC pauses get longer.

Fix: raise the limit to at least 1 core, or set only a CPU request and no
CPU limit.`,
		Check: func(c *model.Container) *Finding {
			if c.CPULimit == 0 || c.CPULimit >= 1000 {
				return nil
			}
			return &Finding{
				Message: "CPU limit " + quantity.CPU(c.CPULimit) + " is below 1 core → throttling during startup, JIT and GC",
				Fix:     "raise the CPU limit to ≥ 1, or keep only resources.requests.cpu",
			}
		},
	})
}
