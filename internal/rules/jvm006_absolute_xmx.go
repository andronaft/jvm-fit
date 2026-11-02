package rules

import (
	"github.com/andronaft/jvm-fit/internal/model"
)

func init() {
	register(Rule{
		ID:       "JVM006",
		Severity: Info,
		Title:    "Fixed -Xmx in absolute units",
		Explain: `-Xmx2g does not change when someone changes the memory limit. After a few
right-sizing rounds the two drift apart: the heap no longer fits, or most of
the limit is unused.

Fix: replace -Xmx with -XX:MaxRAMPercentage so the heap follows the limit.`,
		Check: func(c *model.Container) *Finding {
			if c.HeapSource != model.HeapFromXmx {
				return nil
			}
			src := c.Flags.Flags.MaxHeap.Source.Name
			return &Finding{
				Message: "Heap is fixed with " + c.Flags.Flags.MaxHeap.Raw + " (" + src + ") and will not follow changes to the limit",
				Fix:     "replace it with -XX:MaxRAMPercentage=70",
			}
		},
	})
}
