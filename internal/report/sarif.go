package report

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"

	"github.com/andronaft/jvm-fit/internal/rules"
)

const (
	sarifSchema  = "https://json.schemastore.org/sarif-2.1.0.json"
	sarifVersion = "2.1.0"
	infoURI      = "https://github.com/andronaft/jvm-fit"
)

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version,omitempty"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	ShortDescription     sarifText       `json:"shortDescription"`
	FullDescription      sarifText       `json:"fullDescription"`
	Help                 sarifText       `json:"help"`
	HelpURI              string          `json:"helpUri"`
	DefaultConfiguration sarifRuleConfig `json:"defaultConfiguration"`
	Properties           sarifRuleProps  `json:"properties"`
}

type sarifRuleConfig struct {
	Level string `json:"level"`
}

type sarifRuleProps struct {
	Tags     []string `json:"tags"`
	Severity string   `json:"jvm-fit/severity"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	RuleIndex int             `json:"ruleIndex"`
	Level     string          `json:"level"`
	Message   sarifText       `json:"message"`
	Locations []sarifLocation `json:"locations"`
	// PartialFingerprints keep an alert stable when lines move.
	PartialFingerprints map[string]string `json:"partialFingerprints"`
	Properties          sarifResultProps  `json:"properties"`
}

type sarifResultProps struct {
	Severity  string `json:"jvm-fit/severity"`
	Workload  string `json:"jvm-fit/workload"`
	Container string `json:"jvm-fit/container"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           *sarifRegion  `json:"region,omitempty"`
}

type sarifArtifact struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId,omitempty"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

// sarifLevel maps severities to SARIF levels. GitHub code scanning shows
// "error" as Error, "warning" as Warning and "note" as Note.
func sarifLevel(s rules.Severity) string {
	switch {
	case s >= rules.High:
		return "error"
	case s == rules.Medium:
		return "warning"
	default:
		return "note"
	}
}

// SARIF writes a SARIF 2.1.0 log for GitHub code scanning and other tools.
// File paths are written relative to the current directory with forward slashes.
func SARIF(w io.Writer, r Result, version string) error {
	all := rules.All()
	index := map[string]int{}
	driver := sarifDriver{Name: "jvm-fit", Version: version, InformationURI: infoURI, Rules: []sarifRule{}}
	for i, rule := range all {
		index[rule.ID] = i
		driver.Rules = append(driver.Rules, sarifRule{
			ID:                   rule.ID,
			Name:                 rule.Name,
			ShortDescription:     sarifText{Text: rule.Title},
			FullDescription:      sarifText{Text: firstParagraph(rule.Explain)},
			Help:                 sarifText{Text: rule.Explain},
			HelpURI:              infoURI + "#rules",
			DefaultConfiguration: sarifRuleConfig{Level: sarifLevel(rule.Severity)},
			Properties:           sarifRuleProps{Tags: []string{"kubernetes", "jvm", "reliability"}, Severity: rule.Severity.String()},
		})
	}

	results := []sarifResult{}
	for _, c := range r.Containers {
		m := c.Model
		uri, base := artifactURI(m.Workload.File)
		loc := sarifPhysical{ArtifactLocation: sarifArtifact{URI: uri, URIBaseID: base}}
		if line := m.Container.Line; line > 0 {
			loc.Region = &sarifRegion{StartLine: line}
		} else if m.Workload.Line > 0 {
			loc.Region = &sarifRegion{StartLine: m.Workload.Line}
		}
		for _, f := range c.Findings {
			text := f.Message + " (" + m.Workload.ID() + ", container " + m.Container.Name + ")"
			if f.Fix != "" {
				text += ". Fix: " + f.Fix
			}
			results = append(results, sarifResult{
				RuleID:    f.RuleID,
				RuleIndex: index[f.RuleID],
				Level:     sarifLevel(f.Severity),
				Message:   sarifText{Text: text},
				Locations: []sarifLocation{{PhysicalLocation: loc}},
				PartialFingerprints: map[string]string{
					"jvmFit/v1": f.RuleID + "|" + uri + "|" + m.Workload.ID() + "|" + m.Container.Name,
				},
				Properties: sarifResultProps{Severity: f.Severity.String(), Workload: m.Workload.ID(), Container: m.Container.Name},
			})
		}
	}

	log := sarifLog{
		Schema:  sarifSchema,
		Version: sarifVersion,
		Runs:    []sarifRun{{Tool: sarifTool{Driver: driver}, Results: results}},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}

// artifactURI makes a repository-relative URI. Paths outside the current
// directory stay absolute (file://) and get no uriBaseId.
func artifactURI(file string) (uri, base string) {
	if file == "<stdin>" {
		return "stdin", ""
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return filepath.ToSlash(file), "%SRCROOT%"
	}
	if wd, err := filepath.Abs("."); err == nil {
		if rel, err := filepath.Rel(wd, abs); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel), "%SRCROOT%"
		}
	}
	return "file://" + filepath.ToSlash(abs), ""
}

func firstParagraph(s string) string {
	p, _, _ := strings.Cut(s, "\n\n")
	return strings.Join(strings.Fields(p), " ")
}
