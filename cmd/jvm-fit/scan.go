package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/andronaft/jvm-fit/internal/flags"
	"github.com/andronaft/jvm-fit/internal/loader"
	"github.com/andronaft/jvm-fit/internal/model"
	"github.com/andronaft/jvm-fit/internal/quantity"
	"github.com/andronaft/jvm-fit/internal/report"
	"github.com/andronaft/jvm-fit/internal/rules"
)

type scanOptions struct {
	format          string
	failOn          string
	nonHeapEstimate string
	flagsEnv        []string
	output          string
	helm            []string
	helmValues      []string
	helmSet         []string
	chartVersion    string
}

func newScanCmd() *cobra.Command {
	opts := scanOptions{}
	cmd := &cobra.Command{
		Use:   "scan [path]...",
		Short: "Scan Kubernetes manifests, Helm charts or Kustomize directories",
		Long: `Scan Kubernetes manifests: YAML files, directories (recursive) or "-" for stdin.

A directory with Chart.yaml is rendered with "helm template", a directory with
kustomization.yaml with "kustomize build" (or "kubectl kustomize"). Use --helm
for charts from a repository or an OCI registry.

Exit codes:
  0  no findings at or above --fail-on
  1  at least one finding at or above --fail-on
  2  error (bad flags, unreadable or invalid YAML, a chart that fails to render)`,
		Example: `  jvm-fit scan ./k8s/
  jvm-fit scan deploy.yaml --format json
  jvm-fit scan ./k8s/ --format sarif --output jvm-fit.sarif
  jvm-fit scan ./charts/payments -f values-prod.yaml
  jvm-fit scan --helm oci://registry-1.docker.io/bitnamicharts/tomcat
  jvm-fit scan ./k8s/overlays/prod
  helm template my-chart | jvm-fit scan -
  jvm-fit scan ./k8s/ --nonheap-estimate 500Mi --flags-env APP_JVM_ARGS`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && len(opts.helm) == 0 {
				return fmt.Errorf("give a path or --helm <chart>")
			}
			return runScan(cmd, args, opts)
		},
	}
	cmd.Flags().StringVar(&opts.format, "format", "text", "output format: text, json, sarif or github (GitHub Actions annotations)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "write the report to this file instead of stdout")
	cmd.Flags().StringVar(&opts.failOn, "fail-on", "high", "exit with code 1 when a finding has this severity or higher (critical, high, medium, low, info, none)")
	cmd.Flags().StringVar(&opts.nonHeapEstimate, "nonheap-estimate", "350Mi", "memory the JVM needs outside the heap (metaspace, threads, code cache, direct buffers)")
	cmd.Flags().StringSliceVar(&opts.flagsEnv, "flags-env", nil, "more env vars that your entrypoint passes to java (added to "+strings.Join(flags.DefaultAssumedEnv, ", ")+")")
	cmd.Flags().StringArrayVar(&opts.helm, "helm", nil, "Helm chart to render: a directory, repo/chart, an OCI reference or a URL (repeatable)")
	cmd.Flags().StringArrayVarP(&opts.helmValues, "values", "f", nil, "values file for the charts given as arguments or with --helm (repeatable)")
	cmd.Flags().StringArrayVar(&opts.helmSet, "set", nil, "set a chart value, key=value (repeatable)")
	cmd.Flags().StringVar(&opts.chartVersion, "chart-version", "", "chart version for --helm charts from a repository")
	return cmd
}

func runScan(cmd *cobra.Command, paths []string, opts scanOptions) error {
	render, ok := renderers[opts.format]
	if !ok {
		return fmt.Errorf("unknown --format %q (want text, json, sarif or github)", opts.format)
	}
	var failOn rules.Severity
	failNever := strings.EqualFold(opts.failOn, "none")
	if !failNever {
		s, err := rules.ParseSeverity(opts.failOn)
		if err != nil {
			return fmt.Errorf("--fail-on: %w", err)
		}
		failOn = s
	}
	nonHeap, err := quantity.ParseBytes(opts.nonHeapEstimate)
	if err != nil || nonHeap <= 0 {
		return fmt.Errorf("--nonheap-estimate: invalid quantity %q", opts.nonHeapEstimate)
	}
	assumed := append(append([]string{}, flags.DefaultAssumedEnv...), opts.flagsEnv...)

	lo := loader.Options{HelmValues: opts.helmValues, HelmSet: opts.helmSet, ChartVersion: opts.chartVersion}
	ws, warnings, err := loader.LoadPaths(paths, lo)
	if err != nil {
		return err
	}
	for _, chart := range opts.helm {
		hw, err := loader.RenderHelm(chart, lo)
		if err != nil {
			return err
		}
		ws = append(ws, hw...)
	}
	for _, w := range warnings {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "warning:", w)
	}
	res := report.Scan(ws, assumed, model.Options{NonHeap: nonHeap})

	if err := writeReport(cmd.OutOrStdout(), opts.output, func(w io.Writer) error { return render(w, res) }); err != nil {
		return err
	}
	if max, ok := res.Max(); ok && !failNever && max >= failOn {
		return exitError{code: 1}
	}
	return nil
}

var renderers = map[string]func(io.Writer, report.Result) error{
	"text":   report.Text,
	"json":   func(w io.Writer, r report.Result) error { return report.JSON(w, r, version) },
	"sarif":  func(w io.Writer, r report.Result) error { return report.SARIF(w, r, version) },
	"github": report.GitHub,
}

// writeReport writes to stdout, or to a file when path is set. A file is
// written only when rendering succeeds.
func writeReport(stdout io.Writer, path string, render func(io.Writer) error) error {
	if path == "" {
		return render(stdout)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := render(f); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}
