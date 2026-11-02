package main

import (
	"errors"

	"github.com/spf13/cobra"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

// exitError ends the program with a code but prints nothing more.
type exitError struct{ code int }

func (e exitError) Error() string { return "exit" }

func asExit(err error, target *exitError) bool { return errors.As(err, target) }

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "jvm-fit",
		Short: "Check if the JVM fits in its Kubernetes container",
		Long: `jvm-fit reads Kubernetes manifests together with JVM flags
(JAVA_TOOL_OPTIONS, JDK_JAVA_OPTIONS, JAVA_OPTS, command and args)
and tells you, before deploy, if the JVM will fit in the container.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newScanCmd(), newExplainCmd())
	return root
}
