package cmd

import (
	"fmt"

	"github.com/joaobarroca93/hactl/output"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <entity_id> <friendly_name>",
	Short: "Set the friendly name of an entity in the entity registry",
	Long: `Set the friendly name of an entity in the Home Assistant entity registry.

This sets the display name only — entity IDs cannot be changed via hactl.
To change an entity ID, use the Home Assistant UI (Settings → Devices & Services → Entities).`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAllMode()
		entityID := args[0]
		friendlyName := args[1]

		msg, err := wsCommand(map[string]any{
			"type":      "config/entity_registry/update",
			"entity_id": entityID,
			"name":      friendlyName,
		})
		if err != nil {
			return output.Err("%s", err)
		}

		if msg.Success != nil && !*msg.Success {
			return output.Err("%s", wsErrMsg(msg.Error))
		}

		if !quiet {
			fmt.Printf("Entity '%s' renamed to '%s'.\n", entityID, friendlyName)
			fmt.Println("Run 'hactl sync' to update the local cache.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
