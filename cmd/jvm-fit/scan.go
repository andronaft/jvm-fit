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
}

func newScanCmd() *cobra.Command {
	opts := scanOptions{}
	cmd := &cobra.Command{
		Use:   "scan <path>...",
		Short: "Scan YAML files or directories with Kubernetes manifests",
		Long: `Scan YAML files or directories with Kubernetes manifests ("-" reads stdin).

Exit codes:
  0  no findings at or above --fail-on
  1  at least one finding at or above --fail-on
  2  error (bad flags, unreadable or invalid YAML)`,
		Example: `  jvm-fit scan ./k8s/
  jvm-fit scan deploy.yaml --format json
  jvm-fit scan ./k8s/ --format sarif --output jvm-fit.sarif
  helm template my-chart | jvm-fit scan -
  jvm-fit scan ./k8s/ --nonheap-estimate 500Mi --flags-env APP_JVM_ARGS`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, args, opts)
		},
	}
	cmd.Flags().StringVar(&opts.format, "format", "text", "output format: text, json or sarif")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "write the report to this file instead of stdout")
	cmd.Flags().StringVar(&opts.failOn, "fail-on", "high", "exit with code 1 when a finding has this severity or higher (critical, high, medium, low, info, none)")
	cmd.Flags().StringVar(&opts.nonHeapEstimate, "nonheap-estimate", "350Mi", "memory the JVM needs outside the heap (metaspace, threads, code cache, direct buffers)")
	cmd.Flags().StringSliceVar(&opts.flagsEnv, "flags-env", nil, "more env vars that your entrypoint passes to java (added to "+strings.Join(flags.DefaultAssumedEnv, ", ")+")")
	return cmd
}

func runScan(cmd *cobra.Command, paths []string, opts scanOptions) error {
	render, ok := renderers[opts.format]
	if !ok {
		return fmt.Errorf("unknown --format %q (want text, json or sarif)", opts.format)
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

	ws, err := loader.LoadPaths(paths)
	if err != nil {
		return err
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
	"text":  report.Text,
	"json":  func(w io.Writer, r report.Result) error { return report.JSON(w, r, version) },
	"sarif": func(w io.Writer, r report.Result) error { return report.SARIF(w, r, version) },
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
		f.Close()
		os.Remove(path)
		return err
	}
	return f.Close()
}
