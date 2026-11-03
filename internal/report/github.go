package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/andronaft/jvm-fit/internal/rules"
)

// GitHub writes findings as GitHub Actions workflow commands
// (::error, ::warning, ::notice), which show up as annotations on the files
// in a pull request and in the run summary.
func GitHub(w io.Writer, r Result) error {
	for _, c := range r.Containers {
		m := c.Model
		props := ""
		if uri, base := artifactURI(m.Workload.File); base != "" {
			props = "file=" + escapeProperty(uri)
			if line := m.Container.Line; line > 0 {
				props += fmt.Sprintf(",line=%d", line)
			} else if m.Workload.Line > 0 {
				props += fmt.Sprintf(",line=%d", m.Workload.Line)
			}
			props += ","
		}
		for _, f := range c.Findings {
			title := f.RuleID
			if rule, ok := rules.Get(f.RuleID); ok {
				title += " " + rule.Name
			}
			msg := f.Message + "\n" + m.Workload.ID() + ", container " + m.Container.Name
			if f.Fix != "" {
				msg += "\nfix: " + f.Fix
			}
			if _, err := fmt.Fprintf(w, "::%s %stitle=%s::%s\n", githubLevel(f.Severity), props, escapeProperty(title), escapeData(msg)); err != nil {
				return err
			}
		}
	}
	return nil
}

// githubLevel uses the same mapping as SARIF: critical and high are errors,
// medium is a warning, low and info are notices.
func githubLevel(s rules.Severity) string {
	switch {
	case s >= rules.High:
		return "error"
	case s == rules.Medium:
		return "warning"
	default:
		return "notice"
	}
}

// Escaping rules of the workflow command format.
var (
	dataEscaper     = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	propertyEscaper = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")
)

func escapeData(s string) string     { return dataEscaper.Replace(s) }
func escapeProperty(s string) string { return propertyEscaper.Replace(s) }
