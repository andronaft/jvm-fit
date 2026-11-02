package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/andronaft/jvm-fit/internal/model"
	"github.com/andronaft/jvm-fit/internal/quantity"
	"github.com/andronaft/jvm-fit/internal/rules"
)

// Text writes the human-readable report.
func Text(w io.Writer, r Result) error {
	p := &printer{w: w}
	if len(r.Containers) == 0 {
		p.f("No Java containers found in %d workloads.\n", r.Workloads)
		return p.err
	}
	for i, c := range r.Containers {
		if i > 0 {
			p.f("\n")
		}
		p.container(c)
	}
	p.f("\n%s\n", summary(r))
	return p.err
}

type printer struct {
	w   io.Writer
	err error
}

func (p *printer) f(format string, args ...any) {
	if p.err != nil {
		return
	}
	_, p.err = fmt.Fprintf(p.w, format, args...)
}

func (p *printer) container(c Container) {
	m := c.Model
	wl := m.Workload
	name := wl.Name
	if wl.Namespace != "" {
		name = wl.Namespace + "/" + name
	}
	p.f("%s (%s, container: %s)  image: %s\n", name, wl.Kind, m.Container.Name, m.Container.Image)
	p.f("  %s  limits: %s   flags: %s\n", location(m), limits(m), flagList(m))

	p.budget(m)
	p.cpu(m)
	for _, n := range notes(m) {
		p.f("  note: %s\n", n)
	}
	p.f("\n")
	if len(c.Findings) == 0 {
		p.f("  ✅ no findings\n")
		return
	}
	for _, f := range c.Findings {
		p.f("  %-8s %s  %s\n", strings.ToUpper(f.Severity.String()), f.RuleID, f.Message)
		if f.Fix != "" {
			p.f("  %-8s %s  fix: %s\n", "", strings.Repeat(" ", len(f.RuleID)), f.Fix)
		}
	}
}

func location(m *model.Container) string {
	if m.Container.Line > 0 {
		return fmt.Sprintf("%s:%d", m.Workload.File, m.Container.Line)
	}
	return m.Workload.File
}

func limits(m *model.Container) string {
	mem, cpu := "none", "none"
	if v, ok := m.Container.Resources.Limits["memory"]; ok {
		mem = v
	}
	if v, ok := m.Container.Resources.Limits["cpu"]; ok {
		cpu = v
	}
	return "memory=" + mem + " cpu=" + cpu
}

func flagList(m *model.Container) string {
	seen := m.Flags.Flags.Seen
	if len(seen) == 0 {
		return "none"
	}
	// Group options by source, keeping the order sources first appear in.
	var order []string
	groups := map[string][]string{}
	for _, o := range seen {
		key := o.Source.Name
		if o.Source.Assumed {
			key += ", assumed"
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], o.Raw)
	}
	parts := make([]string, 0, len(order))
	for _, k := range order {
		parts = append(parts, strings.Join(groups[k], " ")+" ("+k+")")
	}
	return strings.Join(parts, "; ")
}

func heapLabel(m *model.Container) string {
	switch m.HeapSource {
	case model.HeapFromXmx:
		return "heap max (-Xmx)"
	case model.HeapFromPercentage:
		return fmt.Sprintf("heap max (%s%%)", trimFloat(m.HeapPercent))
	default:
		return "heap max (default)"
	}
}

func (p *printer) budget(m *model.Container) {
	p.f("\n  Memory budget\n")
	const row = "    %-20s %8s%s\n"
	limit := "none"
	if m.MemLimit > 0 {
		limit = quantity.Mi(m.MemLimit)
	}
	p.f(row, "container limit", limit, "")
	if m.HeapMax == 0 {
		reason := "depends on node memory"
		if !m.ContainerSupport {
			reason = "container support is off"
		}
		p.f(row, heapLabel(m), "?", "   ("+reason+")")
		return
	}
	note := ""
	if m.HeapSource == model.HeapDefault && m.HeapDefaultNote != "" {
		note = "   (" + m.HeapDefaultNote + ")"
	}
	p.f(row, heapLabel(m), quantity.Mi(m.HeapMax), note)
	p.f(row, "non-heap estimate", "~"+quantity.Mi(m.NonHeap), "")
	p.f("    %s\n", strings.Repeat("─", 29))
	verdict := ""
	if m.MemLimit > 0 {
		if diff := m.MemLimit - m.Needed(); diff >= 0 {
			verdict = "   ✅ fits (~" + quantity.Mi(diff) + " free)"
		} else {
			verdict = "   ❌ over limit by ~" + quantity.Mi(-diff)
		}
	}
	p.f(row, "needed", "~"+quantity.Mi(m.Needed()), verdict)
}

func (p *printer) cpu(m *model.Container) {
	cpus := "? (depends on node)"
	if m.CPUs > 0 {
		cpus = fmt.Sprintf("%d (%s)", m.CPUs, m.CPUSource)
	}
	gc := m.GC
	switch {
	case gc == "":
		gc = "? (depends on node)"
	case m.GCExplicit:
		gc += " (set explicitly)"
	case m.GCReason != "":
		gc += " (" + m.GCReason + ")"
	}
	p.f("\n  CPU   JVM sees %s\n", cpus)
	p.f("  GC    %s\n", gc)
}

func notes(m *model.Container) []string {
	var out []string
	for _, name := range m.Flags.Unresolved {
		out = append(out, name+" comes from a ConfigMap or Secret; its flags are not checked")
	}
	for _, w := range m.Flags.Warnings {
		out = append(out, "could not parse "+w.String())
	}
	for _, e := range m.ResourceErrors {
		out = append(out, "could not parse "+e)
	}
	return out
}

func summary(r Result) string {
	counts := r.Counts()
	total := 0
	var parts []string
	for s := rules.Critical; s >= rules.Info; s-- {
		if n := counts[s]; n > 0 {
			total += n
			parts = append(parts, fmt.Sprintf("%d %s", n, s))
		}
	}
	if total == 0 {
		return fmt.Sprintf("No findings in %d Java containers (%d workloads scanned)", len(r.Containers), r.Workloads)
	}
	return fmt.Sprintf("%d %s (%s) in %d of %d workloads",
		total, plural(total, "finding"), strings.Join(parts, ", "), r.WorkloadsWithFindings(), r.Workloads)
}

func plural(n int, s string) string {
	if n == 1 {
		return s
	}
	return s + "s"
}

func trimFloat(f float64) string {
	s := fmt.Sprintf("%.2f", f)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return s
}

// RuleList writes all rules as a table, for "jvm-fit explain" without arguments.
func RuleList(w io.Writer) error {
	p := &printer{w: w}
	all := rules.All()
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	for _, r := range all {
		p.f("%s  %-8s  %s\n", r.ID, r.Severity, r.Title)
	}
	return p.err
}
