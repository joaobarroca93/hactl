package cmd

import (
	"fmt"
	"strings"

	"github.com/joaobarroca/hactl/output"
	"github.com/spf13/cobra"
)

var automationCmd = &cobra.Command{
	Use:   "automation",
	Short: "List, trigger, enable, or disable automations",
}

var automationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all automations",
	RunE: func(cmd *cobra.Command, args []string) error {
		states, err := getClient().ListStates()
		if err != nil {
			return output.Err("%s", err)
		}

		// Apply entity filter before domain filtering.
		states = entityFilter.FilterStates(states)

		type AutomationInfo struct {
			EntityID     string `json:"entity_id"`
			State        string `json:"state"`
			FriendlyName string `json:"friendly_name,omitempty"`
			LastTriggered string `json:"last_triggered,omitempty"`
		}

		var automations []AutomationInfo
		for _, s := range states {
			if !strings.HasPrefix(s.EntityID, "automation.") {
				continue
			}
			info := AutomationInfo{
				EntityID: s.EntityID,
				State:    s.State,
			}
			if name, ok := s.Attributes["friendly_name"].(string); ok {
				info.FriendlyName = name
			}
			if lt, ok := s.Attributes["last_triggered"].(string); ok {
				info.LastTriggered = lt
			}
			automations = append(automations, info)
		}

		if quiet {
			return nil
		}
		if plain {
			for _, a := range automations {
				name := a.FriendlyName
				if name == "" {
					name = a.EntityID
				}
				output.PrintPlain(fmt.Sprintf("%s [%s]", name, a.State))
			}
			return nil
		}
		return output.PrintJSON(automations)
	},
}

var automationTriggerCmd = &cobra.Command{
	Use:   "trigger <automation_id>",
	Short: "Trigger an automation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityID := ensureAutomationPrefix(args[0])
		if !entityFilter.IsAllowed(entityID) {
			return output.Err("entity not found: %s", entityID)
		}
		_, err := getClient().CallService("automation", "trigger", map[string]any{
			"entity_id": entityID,
		})
		if err != nil {
			return output.Err("%s", err)
		}
		if quiet {
			return nil
		}
		if plain {
			output.PrintPlain(fmt.Sprintf("triggered %s", entityID))
			return nil
		}
		return output.PrintJSON(map[string]string{"triggered": entityID})
	},
}

var automationEnableCmd = &cobra.Command{
	Use:   "enable <automation_id>",
	Short: "Enable an automation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityID := ensureAutomationPrefix(args[0])
		if !entityFilter.IsAllowed(entityID) {
			return output.Err("entity not found: %s", entityID)
		}
		_, err := getClient().CallService("automation", "turn_on", map[string]any{
			"entity_id": entityID,
		})
		if err != nil {
			return output.Err("%s", err)
		}
		if quiet {
			return nil
		}
		if plain {
			output.PrintPlain(fmt.Sprintf("enabled %s", entityID))
			return nil
		}
		return output.PrintJSON(map[string]string{"enabled": entityID})
	},
}

var automationDisableCmd = &cobra.Command{
	Use:   "disable <automation_id>",
	Short: "Disable an automation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityID := ensureAutomationPrefix(args[0])
		if !entityFilter.IsAllowed(entityID) {
			return output.Err("entity not found: %s", entityID)
		}
		_, err := getClient().CallService("automation", "turn_off", map[string]any{
			"entity_id": entityID,
		})
		if err != nil {
			return output.Err("%s", err)
		}
		if quiet {
			return nil
		}
		if plain {
			output.PrintPlain(fmt.Sprintf("disabled %s", entityID))
			return nil
		}
		return output.PrintJSON(map[string]string{"disabled": entityID})
	},
}

func init() {
	automationCmd.AddCommand(automationListCmd)
	automationCmd.AddCommand(automationTriggerCmd)
	automationCmd.AddCommand(automationEnableCmd)
	automationCmd.AddCommand(automationDisableCmd)
}

func ensureAutomationPrefix(id string) string {
	if strings.HasPrefix(id, "automation.") {
		return id
	}
	return "automation." + id
}
