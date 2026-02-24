package cmd

import (
	"fmt"

	"github.com/joaobarroca/hactl/output"
	"github.com/spf13/cobra"
)

var exposeCmd = &cobra.Command{
	Use:   "expose <entity_id>",
	Short: "Mark an entity as exposed to HA Assist",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAllMode()
		entityID := args[0]

		msg, err := wsCommand(map[string]any{
			"type":      "config/entity_registry/update",
			"entity_id": entityID,
			"options": map[string]any{
				"conversation": map[string]any{
					"should_expose": true,
				},
			},
		})
		if err != nil {
			return output.Err("%s", err)
		}

		if msg.Success != nil && !*msg.Success {
			return output.Err("%s", wsErrMsg(msg.Error))
		}

		if !quiet {
			fmt.Printf("Entity '%s' is now exposed to Assist.\n", entityID)
			fmt.Println("Run 'hactl sync' to update the local cache.")
		}
		return nil
	},
}

var unexposeCmd = &cobra.Command{
	Use:   "unexpose <entity_id>",
	Short: "Hide an entity from HA Assist",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAllMode()
		entityID := args[0]

		msg, err := wsCommand(map[string]any{
			"type":      "config/entity_registry/update",
			"entity_id": entityID,
			"options": map[string]any{
				"conversation": map[string]any{
					"should_expose": false,
				},
			},
		})
		if err != nil {
			return output.Err("%s", err)
		}

		if msg.Success != nil && !*msg.Success {
			return output.Err("%s", wsErrMsg(msg.Error))
		}

		if !quiet {
			fmt.Printf("Entity '%s' is now hidden from Assist.\n", entityID)
			fmt.Println("Run 'hactl sync' to update the local cache.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exposeCmd)
	rootCmd.AddCommand(unexposeCmd)
}
