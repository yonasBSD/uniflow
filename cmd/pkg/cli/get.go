package cli

import (
	"github.com/siyul-park/uniflow/cmd/pkg/printer"
	"github.com/siyul-park/uniflow/pkg/scheme"
	"github.com/siyul-park/uniflow/pkg/spec"
	"github.com/siyul-park/uniflow/pkg/store"
	"github.com/spf13/cobra"
)

// GetConfig represents the configuration for the get command.
type GetConfig struct {
	Scheme *scheme.Scheme
	Store  *store.Store
}

// NewGetCommand creates a new cobra.Command for the get command.
func NewGetCommand(config GetConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get node specifications from the specified namespace",
		RunE:  runGetCommand(config),
	}

	cmd.PersistentFlags().StringP(flagNamespace, toShorthand(flagNamespace), spec.DefaultNamespace, "Set the resource's namespace. If not set, use all namespace")

	return cmd
}

func runGetCommand(config GetConfig) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		namespace, err := cmd.Flags().GetString(flagNamespace)
		if err != nil {
			return err
		}

		var filter *store.Filter
		if namespace != "" {
			filter = store.Where[string](spec.KeyNamespace).Equal(namespace)
		}

		specs, err := config.Store.FindMany(ctx, filter)
		if err != nil {
			return err
		}

		return printer.PrintTable(cmd.OutOrStdout(), specs, printer.SpecTableColumnDefinitions)
	}
}
