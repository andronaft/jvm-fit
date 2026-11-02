// Package report renders scan results as text or JSON.
package report

import (
	"github.com/andronaft/jvm-fit/internal/loader"
	"github.com/andronaft/jvm-fit/internal/model"
	"github.com/andronaft/jvm-fit/internal/rules"
)

// Result is the outcome of one scan.
type Result struct {
	Workloads  int
	Containers []Container
}

// Container is one Java container and what the rules found in it.
type Container struct {
	Model    *model.Container
	Findings []rules.Finding
}

// Scan models every container in the workloads and runs the rules.
func Scan(ws []loader.Workload, assumedEnv []string, opts model.Options) Result {
	res := Result{Workloads: len(ws)}
	for _, w := range ws {
		for _, c := range w.Spec.Containers {
			m := model.Build(w, c, assumedEnv, opts)
			if m == nil {
				continue
			}
			res.Containers = append(res.Containers, Container{Model: m, Findings: rules.Run(m)})
		}
	}
	return res
}

// Max returns the highest severity found and whether there was any finding.
func (r Result) Max() (rules.Severity, bool) {
	var max rules.Severity
	found := false
	for _, c := range r.Containers {
		for _, f := range c.Findings {
			if !found || f.Severity > max {
				max, found = f.Severity, true
			}
		}
	}
	return max, found
}

// Counts returns the number of findings per severity.
func (r Result) Counts() map[rules.Severity]int {
	out := map[rules.Severity]int{}
	for _, c := range r.Containers {
		for _, f := range c.Findings {
			out[f.Severity]++
		}
	}
	return out
}

// WorkloadsWithFindings counts distinct workloads that have at least one finding.
func (r Result) WorkloadsWithFindings() int {
	seen := map[string]bool{}
	for _, c := range r.Containers {
		if len(c.Findings) > 0 {
			w := c.Model.Workload
			seen[w.File+"|"+w.ID()] = true
		}
	}
	return len(seen)
}
