package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/andronaft/jvm-fit/internal/report"
	"github.com/andronaft/jvm-fit/internal/rules"
)

func newExplainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "explain [rule-id]",
		Short: "Explain a rule: why it is a problem, how to fix it, where to read more",
		Long:  "Explain a rule. Without a rule ID, list all rules.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if len(args) == 0 {
				return report.RuleList(out)
			}
			r, ok := rules.Get(args[0])
			if !ok {
				return fmt.Errorf("unknown rule %q (run \"jvm-fit explain\" for the list)", args[0])
			}
			if _, err := fmt.Fprintf(out, "%s  %s  [%s]\n\n%s\n\nSee also:\n", r.ID, r.Title, r.Severity, r.Explain); err != nil {
				return err
			}
			for _, ref := range r.Refs {
				if _, err := fmt.Fprintf(out, "  %s\n", ref); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
