package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/joaobarroca93/hactl/client"
	"github.com/joaobarroca93/hactl/output"
	"github.com/spf13/cobra"
)

// restrictedServices are blocked in filter.mode: exposed.
// Only permitted when the user explicitly sets filter.mode: all.
var restrictedServices = map[string]bool{
	"homeassistant.restart": true,
	"homeassistant.stop":    true,
}

// entityRequiredServiceDomains lists domains whose services always require an
// entity_id. Calling them without --entity is an error we can catch early.
// Domains not listed here (notify, homeassistant, tts, …) are passed through
// as-is and let Home Assistant decide.
var entityRequiredServiceDomains = map[string]bool{
	"light":                true,
	"switch":               true,
	"climate":              true,
	"cover":                true,
	"fan":                  true,
	"media_player":         true,
	"vacuum":               true,
	"lock":                 true,
	"button":               true,
	"scene":                true,
	"script":               true,
	"alarm_control_panel":  true,
	"siren":                true,
	"input_boolean":        true,
	"input_text":           true,
	"input_number":         true,
	"input_select":         true,
	"input_datetime":       true,
	"automation":           true,
	"todo":                 true,
	"person":               true,
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

		// Fail early for domains that always require an entity.
		if entity == "" && entityRequiredServiceDomains[domain] {
			return output.Err(
				"service %s.%s requires --entity\n  use: hactl service call %s.%s --entity %s.<entity_id>",
				domain, svc, domain, svc, domain,
			)
		}
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

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available Home Assistant services",
	Long: `List all services available in Home Assistant.

Examples:
  hactl service list
  hactl service list --domain notify
  hactl service list --domain notify --plain`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		domain, _ := cmd.Flags().GetString("domain")
		domains, err := getClient().GetServices(domain)
		if err != nil {
			return output.Err("%s", err)
		}
		if quiet {
			return nil
		}
		if plain {
			for _, d := range domains {
				for _, svc := range d.Services {
					output.PrintPlain(d.Domain + "." + svc)
				}
			}
			return nil
		}
		return output.PrintJSON(domains)
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

	serviceListCmd.Flags().String("domain", "", "filter by domain (e.g. notify, light)")

	serviceCmd.AddCommand(serviceCallCmd)
	serviceCmd.AddCommand(serviceListCmd)
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
