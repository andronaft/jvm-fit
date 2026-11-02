package report

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/andronaft/jvm-fit/internal/loader"
	"github.com/andronaft/jvm-fit/internal/model"
)

func scanTestdata(t *testing.T) Result {
	t.Helper()
	ws, err := loader.LoadPaths([]string{filepath.Join("..", "..", "testdata")})
	if err != nil {
		t.Fatal(err)
	}
	return Scan(ws, nil, model.Options{})
}

// TestSARIFSchema validates the output against the official SARIF 2.1.0
// schema from OASIS (testdata/sarif-schema-2.1.0.json).
func TestSARIFSchema(t *testing.T) {
	var buf bytes.Buffer
	if err := SARIF(&buf, scanTestdata(t), "test"); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join("testdata", "sarif-schema-2.1.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	schemaDoc, err := jsonschema.UnmarshalJSON(f)
	_ = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("sarif.json", schemaDoc); err != nil {
		t.Fatal(err)
	}
	schema, err := c.Compile("sarif.json")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(doc); err != nil {
		t.Fatalf("SARIF does not match the schema: %v", err)
	}
}

func TestSARIFContent(t *testing.T) {
	var buf bytes.Buffer
	if err := SARIF(&buf, scanTestdata(t), "1.2.3"); err != nil {
		t.Fatal(err)
	}
	var log struct {
		Runs []struct {
			Tool struct {
				Driver struct {
					Version string `json:"version"`
					Rules   []struct {
						ID string `json:"id"`
					} `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID    string `json:"ruleId"`
				RuleIndex int    `json:"ruleIndex"`
				Level     string `json:"level"`
				Message   struct {
					Text string `json:"text"`
				} `json:"message"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI       string `json:"uri"`
							URIBaseID string `json:"uriBaseId"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatal(err)
	}
	run := log.Runs[0]
	if run.Tool.Driver.Version != "1.2.3" || len(run.Tool.Driver.Rules) != 12 {
		t.Errorf("driver = %+v", run.Tool.Driver)
	}
	if len(run.Results) != 10 {
		t.Errorf("got %d results, want 10", len(run.Results))
	}
	levels := map[string]int{}
	for _, r := range run.Results {
		levels[r.Level]++
		if run.Tool.Driver.Rules[r.RuleIndex].ID != r.RuleID {
			t.Errorf("ruleIndex %d does not point to %s", r.RuleIndex, r.RuleID)
		}
		loc := r.Locations[0].PhysicalLocation
		// Relative to the package directory the tests run in.
		if !strings.HasPrefix(loc.ArtifactLocation.URI, "file://") && loc.ArtifactLocation.URIBaseID != "%SRCROOT%" {
			t.Errorf("relative uri without uriBaseId: %+v", loc.ArtifactLocation)
		}
		if loc.Region.StartLine == 0 {
			t.Errorf("%s has no line", r.RuleID)
		}
	}
	// 3 high → error, 5 medium → warning, 1 low + 1 info → note.
	if levels["error"] != 3 || levels["warning"] != 5 || levels["note"] != 2 {
		t.Errorf("levels = %v", levels)
	}
}

func TestArtifactURI(t *testing.T) {
	if uri, base := artifactURI("deploy/app.yaml"); uri != "deploy/app.yaml" || base != "%SRCROOT%" {
		t.Errorf("relative: %s %s", uri, base)
	}
	if uri, base := artifactURI("<stdin>"); uri != "stdin" || base != "" {
		t.Errorf("stdin: %s %s", uri, base)
	}
	if uri, _ := artifactURI("/elsewhere/app.yaml"); !strings.HasPrefix(uri, "file:///") {
		t.Errorf("outside: %s", uri)
	}
}

func TestTextNoJava(t *testing.T) {
	var buf bytes.Buffer
	if err := Text(&buf, Result{Workloads: 2}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No Java containers found in 2 workloads") {
		t.Errorf("output: %s", buf.String())
	}
}
