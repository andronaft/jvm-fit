package main

import (
	"fmt"
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
}

func newScanCmd() *cobra.Command {
	opts := scanOptions{}
	cmd := &cobra.Command{
		Use:   "scan <path>...",
		Short: "Scan YAML files or directories with Kubernetes manifests",
		Long: `Scan YAML files or directories with Kubernetes manifests ("-" reads stdin).

Exit codes: 0 no findings at or above --fail-on, 1 findings at or above
--fail-on, 2 error.`,
		Example: `  jvm-fit scan ./k8s/
  jvm-fit scan deploy.yaml --format json
  helm template my-chart | jvm-fit scan -
  jvm-fit scan ./k8s/ --nonheap-estimate 500Mi --flags-env APP_JVM_ARGS`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, args, opts)
		},
	}
	cmd.Flags().StringVar(&opts.format, "format", "text", "output format: text or json")
	cmd.Flags().StringVar(&opts.failOn, "fail-on", "high", "exit with code 1 when a finding has this severity or higher (critical, high, medium, low, info, none)")
	cmd.Flags().StringVar(&opts.nonHeapEstimate, "nonheap-estimate", "350Mi", "memory the JVM needs outside the heap (metaspace, threads, code cache, direct buffers)")
	cmd.Flags().StringSliceVar(&opts.flagsEnv, "flags-env", nil, "more env vars that your entrypoint passes to java (added to "+strings.Join(flags.DefaultAssumedEnv, ", ")+")")
	return cmd
}

func runScan(cmd *cobra.Command, paths []string, opts scanOptions) error {
	if opts.format != "text" && opts.format != "json" {
		return fmt.Errorf("unknown --format %q (want text or json)", opts.format)
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

	out := cmd.OutOrStdout()
	if opts.format == "json" {
		err = report.JSON(out, res, version)
	} else {
		err = report.Text(out, res)
	}
	if err != nil {
		return err
	}
	if max, ok := res.Max(); ok && !failNever && max >= failOn {
		return exitError{code: 1}
	}
	return nil
}
