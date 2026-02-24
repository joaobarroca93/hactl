package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/joaobarroca/hactl/client"
	"github.com/joaobarroca/hactl/output"
	"github.com/spf13/cobra"
)

// restrictedServices are blocked in filter.mode: exposed.
// Only permitted when the user explicitly sets filter.mode: all.
var restrictedServices = map[string]bool{
	"homeassistant.restart": true,
	"homeassistant.stop":    true,
}

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

		// Block system-wide destructive services in exposed mode.
		if restrictedServices[domain+"."+svc] && entityFilter.Mode() == "exposed" {
			return output.Err(
				"service %s.%s is not permitted in exposed mode\n  to enable it, set filter.mode: all in your config file",
				domain, svc,
			)
		}

		// Build data payload from flags
		data := map[string]any{}

		entity, _ := cmd.Flags().GetString("entity")
		if entity != "" {
			if !entityFilter.IsAllowed(entity) {
				return output.Err("entity not found: %s", entity)
			}
			// Warn when the entity's domain doesn't match the service's domain.
			// homeassistant.* services (turn_on, turn_off, toggle) are cross-domain by design.
			if domain != "homeassistant" {
				entityDomain, _, _ := strings.Cut(entity, ".")
				if entityDomain != domain {
					return output.Err(
						"domain mismatch: service %s.%s cannot target a %s entity\n  did you mean: hactl service call %s.%s --entity %s",
						domain, svc, entityDomain, entityDomain, svc, entity,
					)
				}
			}
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

		// Snapshot state before the call so we can detect when it settles.
		var stateBefore *client.State
		if entity != "" {
			stateBefore, _ = getClient().GetState(entity)
		}

		_, err := getClient().CallService(domain, svc, data)
		if err != nil {
			if entity == "" {
				return output.Err("%s\n  hint: this service may require --entity <entity_id>", err)
			}
			return output.Err("%s", err)
		}

		// Poll until the entity state changes or the timeout elapses.
		// Many integrations return an empty/stale response immediately after
		// a service call — the device hasn't acted yet.
		var states []client.State
		if entity != "" {
			s := pollStateChange(entity, stateBefore, 3*time.Second, 250*time.Millisecond)
			if s != nil {
				states = []client.State{*s}
			}
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

// pollStateChange polls entityID until its state differs from before, or timeout elapses.
// Returns the latest state in either case (nil only if all fetches fail).
func pollStateChange(entityID string, before *client.State, timeout, interval time.Duration) *client.State {
	deadline := time.Now().Add(timeout)
	var last *client.State
	for time.Now().Before(deadline) {
		time.Sleep(interval)
		s, err := getClient().GetState(entityID)
		if err != nil {
			continue
		}
		last = s
		if before == nil || s.State != before.State {
			return s
		}
	}
	return last
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
