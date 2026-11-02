// Package loader reads Kubernetes manifests from YAML files and directories
// and returns the pod templates inside them.
package loader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Workload is one object with a pod spec: a Pod, Deployment, StatefulSet,
// DaemonSet, ReplicaSet, Job or CronJob.
type Workload struct {
	Kind      string
	Name      string
	Namespace string
	File      string
	Line      int
	Spec      PodSpec
}

// ID is a short name for reports: "Deployment/payments-api".
func (w Workload) ID() string {
	if w.Namespace != "" {
		return w.Namespace + "/" + w.Kind + "/" + w.Name
	}
	return w.Kind + "/" + w.Name
}

type PodSpec struct {
	Containers     []Container `yaml:"containers"`
	InitContainers []Container `yaml:"initContainers"`
}

type Container struct {
	Name           string    `yaml:"name"`
	Image          string    `yaml:"image"`
	Command        []string  `yaml:"command"`
	Args           []string  `yaml:"args"`
	Env            []EnvVar  `yaml:"env"`
	Resources      Resources `yaml:"resources"`
	LivenessProbe  *Probe    `yaml:"livenessProbe"`
	ReadinessProbe *Probe    `yaml:"readinessProbe"`
	StartupProbe   *Probe    `yaml:"startupProbe"`
	Line           int       `yaml:"-"`
}

type EnvVar struct {
	Name      string     `yaml:"name"`
	Value     string     `yaml:"value"`
	ValueFrom *yaml.Node `yaml:"valueFrom"`
}

type Resources struct {
	Limits   map[string]string `yaml:"limits"`
	Requests map[string]string `yaml:"requests"`
}

type Probe struct {
	InitialDelaySeconds int `yaml:"initialDelaySeconds"`
	PeriodSeconds       int `yaml:"periodSeconds"`
	FailureThreshold    int `yaml:"failureThreshold"`
	TimeoutSeconds      int `yaml:"timeoutSeconds"`
}

// LoadPaths reads every .yaml and .yml file in the given files and directories.
// Directories are walked recursively. "-" reads from stdin.
func LoadPaths(paths []string) ([]Workload, error) {
	var out []Workload
	for _, p := range paths {
		if p == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil, err
			}
			ws, err := Parse(data, "<stdin>")
			if err != nil {
				return nil, err
			}
			out = append(out, ws...)
			continue
		}
		files, err := yamlFiles(p)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				return nil, err
			}
			ws, err := Parse(data, f)
			if err != nil {
				return nil, err
			}
			out = append(out, ws...)
		}
	}
	return out, nil
}

func yamlFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{root}, nil
	}
	var files []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && p != root && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		ext := strings.ToLower(filepath.Ext(p))
		if !d.IsDir() && (ext == ".yaml" || ext == ".yml") {
			files = append(files, p)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

// Parse reads a multi-document YAML stream. Documents without a pod spec are
// skipped. Helm template files with {{ }} are not valid YAML and give an error.
func Parse(data []byte, file string) ([]Workload, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var out []Workload
	for {
		var doc yaml.Node
		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if len(doc.Content) == 0 {
			continue
		}
		ws, err := fromNode(doc.Content[0], file)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", file, doc.Content[0].Line, err)
		}
		out = append(out, ws...)
	}
	return out, nil
}

type object struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Items []yaml.Node `yaml:"items"`
	Spec  yaml.Node   `yaml:"spec"`
}

// podSpecPath is where each kind keeps its pod spec, under .spec.
var podSpecPath = map[string][]string{
	"Pod":                   {},
	"Deployment":            {"template", "spec"},
	"StatefulSet":           {"template", "spec"},
	"DaemonSet":             {"template", "spec"},
	"ReplicaSet":            {"template", "spec"},
	"ReplicationController": {"template", "spec"},
	"Job":                   {"template", "spec"},
	"CronJob":               {"jobTemplate", "spec", "template", "spec"},
	"PodTemplate":           nil, // has .template, not .spec; handled below
}

func fromNode(n *yaml.Node, file string) ([]Workload, error) {
	if n.Kind != yaml.MappingNode {
		return nil, nil
	}
	var obj object
	if err := n.Decode(&obj); err != nil {
		return nil, err
	}
	if strings.HasSuffix(obj.Kind, "List") {
		var out []Workload
		for i := range obj.Items {
			ws, err := fromNode(&obj.Items[i], file)
			if err != nil {
				return nil, err
			}
			out = append(out, ws...)
		}
		return out, nil
	}
	path, ok := podSpecPath[obj.Kind]
	if !ok || path == nil {
		return nil, nil
	}
	specNode := find(&obj.Spec, path)
	if specNode == nil {
		return nil, nil
	}
	var spec PodSpec
	if err := specNode.Decode(&spec); err != nil {
		return nil, err
	}
	setContainerLines(specNode, &spec)
	return []Workload{{
		Kind:      obj.Kind,
		Name:      obj.Metadata.Name,
		Namespace: obj.Metadata.Namespace,
		File:      file,
		Line:      n.Line,
		Spec:      spec,
	}}, nil
}

func find(n *yaml.Node, path []string) *yaml.Node {
	cur := n
	for _, key := range path {
		cur = child(cur, key)
		if cur == nil {
			return nil
		}
	}
	if cur.Kind != yaml.MappingNode {
		return nil
	}
	return cur
}

func child(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

func setContainerLines(spec *yaml.Node, ps *PodSpec) {
	for key, list := range map[string][]Container{"containers": ps.Containers, "initContainers": ps.InitContainers} {
		seq := child(spec, key)
		if seq == nil || seq.Kind != yaml.SequenceNode {
			continue
		}
		for i := range list {
			if i < len(seq.Content) {
				list[i].Line = seq.Content[i].Line
			}
		}
	}
}
