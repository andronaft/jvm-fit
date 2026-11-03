package loader

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReleaseName(t *testing.T) {
	tests := map[string]string{
		"charts/payments":                                "payments",
		"charts/payments/":                               "payments",
		"bitnami/tomcat":                                 "tomcat",
		"oci://registry.example/charts/Spring-App":       "spring-app",
		"https://charts.example.com/tomcat-11.3.1.tgz":   "tomcat",
		"https://charts.example.com/app-v2.0.0-rc.1.tgz": "app",
		".":        "release",
		"my_chart": "release",
	}
	for in, want := range tests {
		if got := releaseName(in); got != want {
			t.Errorf("releaseName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHelmArgs(t *testing.T) {
	got := helmArgs("charts/app", Options{HelmValues: []string{"prod.yaml"}, HelmSet: []string{"a=--b"}, ChartVersion: "1.2.3"})
	want := []string{"template", "app", "charts/app", "--values=prod.yaml", "--set=a=--b", "--version=1.2.3"}
	if len(got) != len(want) {
		t.Fatalf("args = %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("args = %q, want %q", got, want)
			break
		}
	}
}

func TestParseHelm(t *testing.T) {
	out := []byte(`---
# Source: spring-app/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: spring-app
---
# Source: spring-app/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: spring-app
spec:
  template:
    spec:
      containers:
        - name: app
          image: eclipse-temurin:17-jre
---
# Source: spring-app/charts/worker/templates/job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: worker
spec:
  template:
    spec:
      containers:
        - name: worker
          image: eclipse-temurin:21-jre
`)
	dir := filepath.Join("testdata", "spring-app")
	ws, err := parseHelm(out, dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 2 {
		t.Fatalf("got %d workloads, want 2", len(ws))
	}
	if want := filepath.Join(dir, "templates", "deployment.yaml"); ws[0].File != want {
		t.Errorf("file = %q, want %q", ws[0].File, want)
	}
	// The subchart template does not exist on disk.
	if want := filepath.Join(dir, "Chart.yaml"); ws[1].File != want {
		t.Errorf("subchart file = %q, want %q", ws[1].File, want)
	}
	if ws[0].Line != 0 || ws[0].Spec.Containers[0].Line != 0 {
		t.Error("rendered line numbers should be dropped")
	}

	remote, err := parseHelm(out, "bitnami/spring-app", "")
	if err != nil {
		t.Fatal(err)
	}
	if remote[0].File != "spring-app/templates/deployment.yaml" {
		t.Errorf("remote file = %q", remote[0].File)
	}
}

func needTool(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not in PATH", name)
	}
}

func TestRenderHelm(t *testing.T) {
	needTool(t, "helm")
	dir := filepath.Join("testdata", "spring-app")
	ws, err := RenderHelm(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 1 || ws[0].Kind != "Deployment" || ws[0].Name != "spring-app" {
		t.Fatalf("workloads = %+v", ws)
	}
	c := ws[0].Spec.Containers[0]
	if c.Resources.Limits["memory"] != "768Mi" || c.Resources.Limits["cpu"] != "750m" {
		t.Errorf("preset limits = %v", c.Resources.Limits)
	}

	ws, err = RenderHelm(dir, Options{
		HelmSet:    []string{"resourcesPreset=large", "startupProbe.enabled=true"},
		HelmValues: []string{filepath.Join("testdata", "spring-values-prod.yaml")},
	})
	if err != nil {
		t.Fatal(err)
	}
	c = ws[0].Spec.Containers[0]
	if c.Resources.Limits["memory"] != "3072Mi" || c.StartupProbe == nil {
		t.Errorf("--set not applied: limits %v, startup probe %v", c.Resources.Limits, c.StartupProbe)
	}
	if c.Env[0].Value != "-XX:MaxRAMPercentage=70" {
		t.Errorf("values file not applied: env %+v", c.Env)
	}

	if _, err := RenderHelm(dir, Options{HelmSet: []string{"resourcesPreset=huge"}}); err == nil {
		t.Error("want an error when the chart fails to render")
	}
}

func TestRenderKustomize(t *testing.T) {
	if _, err := exec.LookPath("kustomize"); err != nil {
		needTool(t, "kubectl")
	}
	dir := filepath.Join("testdata", "kustomize", "overlays", "prod")
	ws, err := RenderKustomize(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 1 || ws[0].ID() != "prod/Deployment/billing" {
		t.Fatalf("workloads = %+v", ws)
	}
	if want := filepath.Join(dir, "kustomization.yaml"); ws[0].File != want {
		t.Errorf("file = %q, want %q", ws[0].File, want)
	}
	c := ws[0].Spec.Containers[0]
	if c.Resources.Limits["memory"] != "2Gi" || c.Env[0].Value != "-XX:MaxRAMPercentage=90" {
		t.Errorf("patch not applied: %+v", c)
	}
}

func TestLoadPathsDetectsChartAndKustomization(t *testing.T) {
	needTool(t, "helm")
	if _, err := exec.LookPath("kustomize"); err != nil {
		needTool(t, "kubectl")
	}
	ws, warnings, err := LoadPaths([]string{
		filepath.Join("testdata", "spring-app"),
		filepath.Join("testdata", "kustomize", "overlays", "prod"),
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 2 || len(warnings) != 0 {
		t.Errorf("workloads = %d, warnings = %v", len(ws), warnings)
	}
}

func TestLoadPathsNestedChart(t *testing.T) {
	needTool(t, "helm")
	// testdata holds the chart as a subdirectory: it is rendered with its
	// default values instead of parsing the templates as YAML.
	ws, _, err := LoadPaths([]string{"testdata"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range ws {
		if w.Name == "spring-app" {
			found = true
		}
	}
	if !found {
		t.Errorf("chart not rendered: %+v", ws)
	}
}

func TestLoadPathsWithoutTools(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, _, err := LoadPaths([]string{filepath.Join("testdata", "kustomize", "base")}, Options{}); err == nil {
		t.Error("a kustomization given explicitly needs kustomize or kubectl")
	}
	if _, _, err := LoadPaths([]string{filepath.Join("testdata", "spring-app")}, Options{}); err == nil {
		t.Error("a chart given explicitly needs helm")
	}
	// A chart found while walking is skipped with a warning.
	_, warnings, err := LoadPaths([]string{"testdata"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 {
		t.Errorf("want one warning for the skipped chart, got %v", warnings)
	}
}
