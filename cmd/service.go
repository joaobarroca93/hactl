package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/joaobarroca/hactl/output"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Call Home Assistant services",
}

var serviceCallCmd = &cobra.Command{
	Use:   "call <domain.service>",
	Short: "Call a Home Assistant service",
	Long: `Call a Home Assistant service.

Examples:
  hactl service call light.turn_on --entity light.living_room --brightness 80
  hactl service call climate.set_temperature --entity climate.bedroom --temperature 21.0
  hactl service call switch.toggle --entity switch.fan
  hactl service call homeassistant.restart`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		parts := strings.SplitN(args[0], ".", 2)
		if len(parts) != 2 {
			return output.Err("service must be in format domain.service (e.g. light.turn_on)")
		}
		domain, svc := parts[0], parts[1]

		// Build data payload from flags
		data := map[string]any{}

		entity, _ := cmd.Flags().GetString("entity")
		if entity != "" {
			data["entity_id"] = entity
		}

		// Generic --data key=value flags
		dataFlags, _ := cmd.Flags().GetStringArray("data")
		for _, kv := range dataFlags {
			k, v, found := strings.Cut(kv, "=")
			if !found {
				return output.Err("--data must be in key=value format, got: %s", kv)
			}
			data[k] = parseValue(v)
		}

		// Convenience flags
		if brightness, err := cmd.Flags().GetInt("brightness"); err == nil && cmd.Flags().Changed("brightness") {
			data["brightness"] = int(float64(brightness) / 100 * 255)
		}
		if temperature, err := cmd.Flags().GetFloat64("temperature"); err == nil && cmd.Flags().Changed("temperature") {
			data["temperature"] = temperature
		}
		if colorTemp, err := cmd.Flags().GetInt("color-temp"); err == nil && cmd.Flags().Changed("color-temp") {
			data["color_temp"] = colorTemp
		}
		if hvacMode, err := cmd.Flags().GetString("hvac-mode"); err == nil && cmd.Flags().Changed("hvac-mode") {
			data["hvac_mode"] = hvacMode
		}
		if rgb, err := cmd.Flags().GetString("rgb"); err == nil && cmd.Flags().Changed("rgb") {
			rgbParts := strings.Split(rgb, ",")
			if len(rgbParts) == 3 {
				r, _ := strconv.Atoi(strings.TrimSpace(rgbParts[0]))
				g, _ := strconv.Atoi(strings.TrimSpace(rgbParts[1]))
				b, _ := strconv.Atoi(strings.TrimSpace(rgbParts[2]))
				data["rgb_color"] = []int{r, g, b}
			}
		}

		states, err := getClient().CallService(domain, svc, data)
		if err != nil {
			return output.Err("%s", err)
		}
		if quiet {
			return nil
		}
		if plain {
			if len(states) == 0 {
				output.PrintPlain(fmt.Sprintf("called %s.%s", domain, svc))
			} else {
				for _, s := range states {
					output.PrintPlain(fmt.Sprintf("%s: %s", s.EntityID, s.State))
				}
			}
			return nil
		}
		return output.PrintJSON(states)
	},
}

func init() {
	serviceCallCmd.Flags().String("entity", "", "entity_id to target")
	serviceCallCmd.Flags().StringArray("data", nil, "additional key=value pairs (repeatable)")
	serviceCallCmd.Flags().Int("brightness", 0, "brightness percentage (0-100), for light services")
	serviceCallCmd.Flags().Float64("temperature", 0, "target temperature, for climate services")
	serviceCallCmd.Flags().Int("color-temp", 0, "color temperature in mireds, for light services")
	serviceCallCmd.Flags().String("hvac-mode", "", "HVAC mode (heat, cool, auto, off), for climate services")
	serviceCallCmd.Flags().String("rgb", "", "RGB color as R,G,B (e.g. 255,128,0)")

	serviceCmd.AddCommand(serviceCallCmd)
}

// parseValue attempts to parse a string as a number, bool, or falls back to string.
func parseValue(s string) any {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return s
}
