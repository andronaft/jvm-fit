package rules

import (
	"fmt"

	"github.com/andronaft/jvm-fit/internal/model"
)

func init() {
	register(Rule{
		ID:       "JVM012",
		Name:     "NoStartupProbe",
		Severity: Medium,
		Title:    "No startup probe and liveness starts early",
		Explain: `A Java service can take 20-60s or more to start, longer with a low CPU limit.
If the liveness probe starts before the app is up, the kubelet restarts the
container again and again (CrashLoopBackOff).

Fix: add a startupProbe with enough failureThreshold × periodSeconds for a
slow start. Liveness checks begin only after the startup probe succeeds.`,
		Refs: []string{refProbes, refProbesHowTo},
		Check: func(c *model.Container) *Finding {
			lp := c.Container.LivenessProbe
			if lp == nil || c.Container.StartupProbe != nil || lp.InitialDelaySeconds >= 30 {
				return nil
			}
			return &Finding{
				Message: fmt.Sprintf("No startup probe; liveness starts after %ds", lp.InitialDelaySeconds),
				Fix:     "add a startupProbe (for example periodSeconds: 5, failureThreshold: 24 for up to 120s)",
			}
		},
	})
}
