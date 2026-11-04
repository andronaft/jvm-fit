package rules

import (
	"github.com/andronaft/jvm-fit/internal/model"
)

func init() {
	register(Rule{
		ID:       "JVM009",
		Name:     "ContainerSupportDisabled",
		Severity: High,
		Title:    "-XX:-UseContainerSupport",
		Explain: `With container support off, the JVM ignores cgroup limits and sizes the heap,
thread pools and GC from the memory and CPUs of the whole node.

Measured on JDK 11, 17 and 21 with a 1Gi limit: the heap is 25% of the
host memory and the GC is G1, as if there were no container.

Fix: remove -XX:-UseContainerSupport (it is on by default since JDK 10 and 8u191).`,
		Refs: []string{refContainerSupport, refJavaMan, refEvidence},
		Check: func(c *model.Container) *Finding {
			if c.ContainerSupport {
				return nil
			}
			v := c.Flags.Flags.UseContainerSupport
			return &Finding{
				Message: v.Raw + " (" + v.Source.Name + ") → the JVM does not see the container limits",
				Fix:     "remove -XX:-UseContainerSupport",
			}
		},
	})
}
