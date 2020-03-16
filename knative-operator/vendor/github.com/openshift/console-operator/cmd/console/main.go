package main

import (
	// standard lib
	goflag "flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	// 3rd party
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	// kube / openshift
	utilflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"

	// us
	"github.com/openshift/console-operator/pkg/cmd/operator"
	"github.com/openshift/console-operator/pkg/cmd/version"
)

func main() {
	// random seed, set it & forget it
	rand.Seed(time.Now().UTC().UnixNano())
	// normalize flags, if _ use -
	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	// add the default flag set for go
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	logs.InitLogs()
	defer logs.FlushLogs()

	// build a new cobra command
	command := NewOperatorCommand()
	// die on errors
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// create the root "console" command
// we will add subcommands to this
func NewOperatorCommand() *cobra.Command {
	// "console" just prints help, then exists.  It doesn't start
	// the operator.
	cmd := &cobra.Command{
		Use:   "console",
		Short: "Top level command",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
	}

	cmd.AddCommand(operator.NewOperator())
	cmd.AddCommand(version.NewVersion())

	return cmd
}
