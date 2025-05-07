package cli

import (
	"fmt"

	"github.com/acuvity/mcp-server-microsoft-graph/api/sites"
	"github.com/acuvity/mcp-server-microsoft-graph/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Run(cmd *cobra.Command, args []string) error {

	cl, err := client.GetClient(
		viper.GetString("tenant-id"),     // Tenant ID
		viper.GetString("client-id"),     // Client ID
		viper.GetString("client-secret"), // Client Secret
	)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	u, err := sites.Get(cmd.Context(), cl, nil)
	if err != nil {
		return fmt.Errorf("error getting sites: %v", err)
	}

	fmt.Println(string(u))
	return nil
}
