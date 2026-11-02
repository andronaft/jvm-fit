package report

import (
	"encoding/json"
	"io"

	"github.com/andronaft/jvm-fit/internal/model"
	"github.com/andronaft/jvm-fit/internal/rules"
)

type jsonReport struct {
	Version    string          `json:"version"`
	Summary    jsonSummary     `json:"summary"`
	Containers []jsonContainer `json:"containers"`
}

type jsonSummary struct {
	Workloads             int            `json:"workloads"`
	JavaContainers        int            `json:"javaContainers"`
	WorkloadsWithFindings int            `json:"workloadsWithFindings"`
	Findings              int            `json:"findings"`
	BySeverity            map[string]int `json:"bySeverity"`
}

type jsonContainer struct {
	Workload jsonWorkload      `json:"workload"`
	Name     string            `json:"container"`
	Image    string            `json:"image"`
	Runtime  string            `json:"runtime"`
	Java     int               `json:"javaVersion,omitempty"`
	Limits   map[string]string `json:"limits"`
	Requests map[string]string `json:"requests"`
	Flags    []jsonFlag        `json:"flags"`
	Heap     jsonHeap          `json:"heap"`
	NonHeap  int64             `json:"nonHeapEstimateBytes"`
	Needed   int64             `json:"neededBytes,omitempty"`
	CPUs     int               `json:"jvmCpus,omitempty"`
	GC       string            `json:"gc,omitempty"`
	GCSet    bool              `json:"gcExplicit"`
	Notes    []string          `json:"notes,omitempty"`
	Findings []rules.Finding   `json:"findings"`
}

type jsonWorkload struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	File      string `json:"file"`
	Line      int    `json:"line,omitempty"`
}

type jsonFlag struct {
	Option  string `json:"option"`
	Source  string `json:"source"`
	Assumed bool   `json:"assumed,omitempty"`
}

type jsonHeap struct {
	MaxBytes int64            `json:"maxBytes,omitempty"`
	Source   model.HeapSource `json:"source"`
	Percent  float64          `json:"percent,omitempty"`
}

// JSON writes the machine-readable report.
func JSON(w io.Writer, r Result, version string) error {
	out := jsonReport{
		Version:    version,
		Containers: []jsonContainer{},
		Summary: jsonSummary{
			Workloads:             r.Workloads,
			JavaContainers:        len(r.Containers),
			WorkloadsWithFindings: r.WorkloadsWithFindings(),
			BySeverity:            map[string]int{},
		},
	}
	for s, n := range r.Counts() {
		out.Summary.BySeverity[s.String()] = n
		out.Summary.Findings += n
	}
	for _, c := range r.Containers {
		m := c.Model
		jc := jsonContainer{
			Workload: jsonWorkload{
				Kind: m.Workload.Kind, Name: m.Workload.Name, Namespace: m.Workload.Namespace,
				File: m.Workload.File, Line: m.Container.Line,
			},
			Name:     m.Container.Name,
			Image:    m.Container.Image,
			Runtime:  string(m.Detect.Runtime),
			Java:     m.Detect.Major,
			Limits:   orEmpty(m.Container.Resources.Limits),
			Requests: orEmpty(m.Container.Resources.Requests),
			Flags:    []jsonFlag{},
			Heap:     jsonHeap{MaxBytes: m.HeapMax, Source: m.HeapSource, Percent: m.HeapPercent},
			NonHeap:  m.NonHeap,
			Needed:   m.Needed(),
			CPUs:     m.CPUs,
			GC:       m.GC,
			GCSet:    m.GCExplicit,
			Notes:    notes(m),
			Findings: c.Findings,
		}
		if jc.Findings == nil {
			jc.Findings = []rules.Finding{}
		}
		for _, o := range m.Flags.Flags.Seen {
			jc.Flags = append(jc.Flags, jsonFlag{Option: o.Raw, Source: o.Source.Name, Assumed: o.Source.Assumed})
		}
		out.Containers = append(out.Containers, jc)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func orEmpty(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}
