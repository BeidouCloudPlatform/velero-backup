package cmd

import (
	"github.com/spf13/cobra"
)

func Execute() {
	// build the root command
	rootCmd := BuildRoot()
	// before hook
	// execute command
	err := rootCmd.Execute()
	// after hook
	if err != nil {
		panic(err)
	}
}

// BuildRoot creates a new root command from the
func BuildRoot() *cobra.Command {
	rootCmd := NewRootCmd()
	rootCmd.AddCommand(newBackupCmd())
	return rootCmd
}

// NewRootCmd returns a new root command
func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "vb",
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "velero backup demo",
		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
