// Package rules holds the checks. Each rule lives in its own file and looks
// at one modeled container.
package rules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/andronaft/jvm-fit/internal/model"
)

// Severity orders findings from info to critical.
type Severity int

const (
	Info Severity = iota
	Low
	Medium
	High
	Critical
)

var severityNames = []string{"info", "low", "medium", "high", "critical"}

func (s Severity) String() string { return severityNames[s] }

// MarshalText makes severities readable in JSON.
func (s Severity) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

// ParseSeverity accepts a severity name, case-insensitive.
func ParseSeverity(s string) (Severity, error) {
	for i, n := range severityNames {
		if strings.EqualFold(s, n) {
			return Severity(i), nil
		}
	}
	return 0, fmt.Errorf("unknown severity %q (want one of %s)", s, strings.Join(severityNames, ", "))
}

// Rule is one check.
type Rule struct {
	ID string
	// Name is a short PascalCase name, used in SARIF.
	Name     string
	Severity Severity
	// Title is a short summary for rule lists.
	Title string
	// Explain is the long text for "jvm-fit explain".
	Explain string
	// Check returns a finding or nil.
	Check func(c *model.Container) *Finding
}

// Finding is a problem found in one container.
type Finding struct {
	RuleID   string   `json:"rule"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Fix      string   `json:"fix,omitempty"`
}

var registry = map[string]Rule{}

func register(r Rule) {
	if _, dup := registry[r.ID]; dup {
		panic("duplicate rule " + r.ID)
	}
	registry[r.ID] = r
}

// All returns every rule sorted by ID.
func All() []Rule {
	out := make([]Rule, 0, len(registry))
	for _, r := range registry {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get returns a rule by ID, case-insensitive.
func Get(id string) (Rule, bool) {
	r, ok := registry[strings.ToUpper(id)]
	return r, ok
}

// Run applies all rules to a container. Findings are sorted by severity, then ID.
func Run(c *model.Container) []Finding {
	var out []Finding
	for _, r := range All() {
		f := r.Check(c)
		if f == nil {
			continue
		}
		f.RuleID, f.Severity = r.ID, r.Severity
		out = append(out, *f)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Severity > out[j].Severity })
	return out
}
