package cmd

import (
	"fmt"
	"strings"

	"github.com/joaobarroca93/hactl/output"
	"github.com/spf13/cobra"
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Get, set, or list entity states",
}

var stateGetCmd = &cobra.Command{
	Use:   "get <entity_id>",
	Short: "Get the current state of an entity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityID := args[0]
		if !entityFilter.IsAllowed(entityID) {
			return output.Err("entity not found: %s", entityID)
		}
		s, err := getClient().GetState(entityID)
		if err != nil {
			return output.Err("%s", err)
		}
		if quiet {
			return nil
		}
		if plain {
			attrs := formatAttrsPlain(s.Attributes)
			if attrs != "" {
				output.PrintPlain(fmt.Sprintf("%s: %s (%s)", s.EntityID, s.State, attrs))
			} else {
				output.PrintPlain(fmt.Sprintf("%s: %s", s.EntityID, s.State))
			}
			return nil
		}
		return output.PrintJSON(s)
	},
}

// serviceControlled maps domains that must be controlled via services.
// Values are example service hints shown in the error message.
var serviceControlled = map[string]string{
	"light":               "light.turn_on / light.turn_off",
	"switch":              "switch.turn_on / switch.turn_off / switch.toggle",
	"climate":             "climate.set_temperature / climate.set_hvac_mode",
	"cover":               "cover.open_cover / cover.close_cover",
	"fan":                 "fan.turn_on / fan.turn_off",
	"media_player":        "media_player.turn_on / media_player.turn_off",
	"vacuum":              "vacuum.start / vacuum.stop",
	"lock":                "lock.lock / lock.unlock",
	"button":              "button.press",
	"scene":               "scene.turn_on",
	"script":              "script.turn_on",
	"alarm_control_panel": "alarm_control_panel.alarm_arm_away / alarm_control_panel.alarm_disarm",
	"siren":               "siren.turn_on / siren.turn_off",
}

var stateSetCmd = &cobra.Command{
	Use:   "set <entity_id> <state>",
	Short: "Set the state of an entity",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityID := args[0]
		newState := args[1]

		domain, _, _ := strings.Cut(entityID, ".")
		if hint, blocked := serviceControlled[domain]; blocked {
			return output.Err(
				"%s entities are controlled via services\n  use: hactl service call %s.turn_on/off --entity %s\n  services: %s",
				domain, domain, entityID, hint,
			)
		}

		s, err := getClient().SetState(entityID, newState, nil)
		if err != nil {
			return output.Err("%s", err)
		}
		if quiet {
			return nil
		}
		if plain {
			output.PrintPlain(fmt.Sprintf("%s set to %s", s.EntityID, s.State))
			return nil
		}
		return output.PrintJSON(s)
	},
}

var (
	stateListDomain string
	stateListArea   string
)

var stateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List entity states, optionally filtered by domain or area",
	RunE: func(cmd *cobra.Command, args []string) error {
		states, err := getClient().ListStates()
		if err != nil {
			return output.Err("%s", err)
		}

		// Apply entity filter
		states = entityFilter.FilterStates(states)

		// Filter by domain
		if stateListDomain != "" {
			filtered := states[:0]
			for _, s := range states {
				if strings.HasPrefix(s.EntityID, stateListDomain+".") {
					filtered = append(filtered, s)
				}
			}
			states = filtered
		}

		// Filter by area using the entity registry cache written by hactl sync.
		if stateListArea != "" {
			filtered := states[:0]
			for _, s := range states {
				if entityFilter.MatchesArea(s.EntityID, stateListArea) {
					filtered = append(filtered, s)
				}
			}
			states = filtered
		}

		if quiet {
			return nil
		}
		if plain {
			for _, s := range states {
				attrs := formatAttrsPlain(s.Attributes)
				if attrs != "" {
					fmt.Printf("%s: %s (%s)\n", s.EntityID, s.State, attrs)
				} else {
					fmt.Printf("%s: %s\n", s.EntityID, s.State)
				}
			}
			return nil
		}
		return output.PrintJSON(states)
	},
}

func init() {
	stateListCmd.Flags().StringVar(&stateListDomain, "domain", "", "filter by domain (e.g. light, climate, sensor, switch, binary_sensor)")
	stateListCmd.Flags().StringVar(&stateListArea, "area", "", "filter by area name")

	stateCmd.AddCommand(stateGetCmd)
	stateCmd.AddCommand(stateSetCmd)
	stateCmd.AddCommand(stateListCmd)
}

// formatAttrsPlain returns a brief human-readable summary of useful attributes.
func formatAttrsPlain(attrs map[string]any) string {
	parts := []string{}

	if brightness, ok := attrs["brightness"]; ok {
		if b, ok := toFloat(brightness); ok {
			pct := int(b / 255 * 100)
			parts = append(parts, fmt.Sprintf("brightness %d%%", pct))
		}
	}
	if temp, ok := attrs["temperature"]; ok {
		if t, ok := toFloat(temp); ok {
			parts = append(parts, fmt.Sprintf("%.1f°C", t))
		}
	}
	if currentTemp, ok := attrs["current_temperature"]; ok {
		if t, ok := toFloat(currentTemp); ok {
			parts = append(parts, fmt.Sprintf("current %.1f°C", t))
		}
	}
	if name, ok := attrs["friendly_name"].(string); ok && name != "" {
		// prepend friendly name
		parts = append([]string{name}, parts...)
	}
	return strings.Join(parts, ", ")
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
