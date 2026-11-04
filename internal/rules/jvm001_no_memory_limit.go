package rules

import "github.com/andronaft/jvm-fit/internal/model"

func init() {
	register(Rule{
		ID:       "JVM001",
		Name:     "NoMemoryLimit",
		Severity: High,
		Title:    "No memory limit and the heap is a percentage of RAM",
		Explain: `Without a memory limit the JVM sees the memory of the whole node. With a
percentage heap (MaxRAMPercentage, or the default 25%) the heap then depends
on the node size: the same pod gets 1Gi of heap on a 4Gi node and 16Gi on a
64Gi node, and the scheduler does not know about it.

Fix: set resources.limits.memory. The heap is then a percentage of the limit.`,
		Refs: []string{refK8sResources, refJavaMan, refEvidence},
		Check: func(c *model.Container) *Finding {
			if c.MemLimit > 0 || c.HeapSource == model.HeapFromXmx || c.Flags.Flags.MaxRAM.Set {
				return nil
			}
			msg := "No memory limit, so the heap is a share of the node memory and changes with the node"
			if c.HeapPercent > 0 {
				msg = "No memory limit, so the heap is " + pct(c.HeapPercent) + " of the node memory and changes with the node"
			}
			return &Finding{
				Message: msg,
				Fix:     "set resources.limits.memory (for example 1Gi) and keep -XX:MaxRAMPercentage=70",
			}
		},
	})
}
