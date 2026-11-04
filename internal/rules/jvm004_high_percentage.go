package rules

import (
	"strconv"

	"github.com/andronaft/jvm-fit/internal/model"
)

func init() {
	register(Rule{
		ID:       "JVM004",
		Name:     "HighMaxRAMPercentage",
		Severity: Medium,
		Title:    "MaxRAMPercentage is above 80",
		Explain: `With MaxRAMPercentage above 80 less than 20% of the limit is left for
metaspace, threads, code cache and direct memory. On small limits this is
much less than a Spring Boot service needs.

Fix: use 70-75 for a normal service. Go higher only with a large limit and a
measured small non-heap.`,
		Refs: []string{refJavaMan},
		Check: func(c *model.Container) *Finding {
			if c.HeapSource != model.HeapFromPercentage || c.HeapPercent <= 80 {
				return nil
			}
			return &Finding{
				Message: "MaxRAMPercentage=" + strconv.FormatFloat(c.HeapPercent, 'f', -1, 64) + " leaves little room for non-heap memory",
				Fix:     "-XX:MaxRAMPercentage=75",
			}
		},
	})
}
