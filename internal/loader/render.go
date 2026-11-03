package loader

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Options change how Helm charts are rendered.
type Options struct {
	// HelmValues (values files), HelmSet (key=value) and ChartVersion are
	// passed to `helm template` for charts given explicitly, not for charts
	// found while walking a directory.
	HelmValues, HelmSet []string
	ChartVersion        string
}

// IsChart reports whether dir is a Helm chart.
func IsChart(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "Chart.yaml"))
	return err == nil
}

var kustomizationFiles = []string{"kustomization.yaml", "kustomization.yml", "Kustomization"}

// kustomizationFile returns the kustomization file in dir, or "".
func kustomizationFile(dir string) string {
	for _, name := range kustomizationFiles {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// helmArgs returns the `helm template` arguments. Values are passed as
// --flag=value so none of them can become another helm flag.
func helmArgs(chart string, o Options) []string {
	args := []string{"template", releaseName(chart), chart}
	for _, v := range o.HelmValues {
		args = append(args, "--values="+v)
	}
	for _, s := range o.HelmSet {
		args = append(args, "--set="+s)
	}
	if o.ChartVersion != "" {
		args = append(args, "--version="+o.ChartVersion)
	}
	return args
}

// releaseName is the chart name, so charts that put the release and chart
// names together (Bitnami-style fullname) give the plain chart name.
// "https://.../tomcat-11.3.1.tgz" gives "tomcat".
func releaseName(chart string) string {
	name := path.Base(filepath.ToSlash(strings.TrimRight(chart, "/")))
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".tgz"), ".tar.gz")
	name = chartVersionSuffix.ReplaceAllString(name, "")
	name = strings.ToLower(name)
	if !releaseNameRe.MatchString(name) {
		return "release"
	}
	return name
}

var (
	chartVersionSuffix = regexp.MustCompile(`-v?\d+\.\d+\.\d+.*$`)
	releaseNameRe      = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
)

// RenderHelm renders a chart with `helm template`. chart is a local chart
// directory or anything helm accepts: repo/chart, an OCI reference, a URL.
func RenderHelm(chart string, o Options) ([]Workload, error) {
	if _, err := exec.LookPath("helm"); err != nil {
		return nil, fmt.Errorf("%s: rendering a Helm chart needs `helm` in PATH", chart)
	}
	out, err := run(exec.Command("helm", helmArgs(chart, o)...)) // #nosec G204 -- no shell; --flag=value arguments
	if err != nil {
		return nil, err
	}
	local := ""
	if info, err := os.Stat(chart); err == nil && info.IsDir() {
		local = chart
	}
	return parseHelm(out, chart, local)
}

var sourceRe = regexp.MustCompile(`(?m)^# Source: (.+)$`)

// parseHelm splits `helm template` output at "---" lines and points each
// workload at the template it came from. For a local chart that is the file
// on disk. Line numbers are dropped: they are lines of the rendered output,
// not of the template.
func parseHelm(out []byte, chart, localDir string) ([]Workload, error) {
	var all []Workload
	for _, doc := range splitDocs(out) {
		file := chart
		if m := sourceRe.FindSubmatch(doc); m != nil {
			file = templatePath(string(m[1]), chart, localDir)
		}
		ws, err := Parse(doc, file)
		if err != nil {
			return nil, fmt.Errorf("helm template %s: %w", chart, err)
		}
		all = append(all, ws...)
	}
	return clearLines(all), nil
}

// templatePath turns "spring-app/templates/deployment.yaml" into the path of
// that file inside a local chart. Templates of packed subcharts do not exist
// on disk; they point at the chart's Chart.yaml.
func templatePath(source, chart, localDir string) string {
	if localDir == "" {
		return source
	}
	_, rest, ok := strings.Cut(source, "/")
	if ok {
		p := filepath.Join(localDir, filepath.FromSlash(rest))
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(localDir, "Chart.yaml")
}

var docSep = regexp.MustCompile(`(?m)^---\s*$`)

func splitDocs(data []byte) [][]byte {
	var docs [][]byte
	for _, d := range docSep.Split(string(data), -1) {
		if strings.TrimSpace(d) != "" {
			docs = append(docs, []byte(d))
		}
	}
	return docs
}

// RenderKustomize builds a Kustomize directory with `kustomize build`, or
// `kubectl kustomize` when kustomize is not installed. Workloads point at
// the kustomization file; the output has no line numbers of the sources.
func RenderKustomize(dir string) ([]Workload, error) {
	file := kustomizationFile(dir)
	if file == "" {
		return nil, fmt.Errorf("%s: no kustomization.yaml", dir)
	}
	var cmd *exec.Cmd
	switch {
	case lookPath("kustomize"):
		cmd = exec.Command("kustomize", "build", dir) // #nosec G204 -- no shell; plugins stay disabled
	case lookPath("kubectl"):
		cmd = exec.Command("kubectl", "kustomize", dir) // #nosec G204 -- no shell; plugins stay disabled
	default:
		return nil, fmt.Errorf("%s is a Kustomize directory but neither `kustomize` nor `kubectl` is in PATH", dir)
	}
	out, err := run(cmd)
	if err != nil {
		return nil, err
	}
	ws, err := Parse(out, file)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", cmd.Args[0], strings.Join(cmd.Args[1:], " "), err)
	}
	return clearLines(ws), nil
}

func run(cmd *exec.Cmd) ([]byte, error) {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %v: %s", cmd.Args[0], strings.Join(cmd.Args[1:], " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func clearLines(ws []Workload) []Workload {
	for i := range ws {
		ws[i].Line = 0
		for j := range ws[i].Spec.Containers {
			ws[i].Spec.Containers[j].Line = 0
		}
		for j := range ws[i].Spec.InitContainers {
			ws[i].Spec.InitContainers[j].Line = 0
		}
	}
	return ws
}

func lookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
