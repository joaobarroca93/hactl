package cmd

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/joaobarroca93/hactl/client"
	"github.com/joaobarroca93/hactl/output"
	"github.com/spf13/cobra"
)

var summaryArea string

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show a digest of current Home Assistant state",
	Long: `Aggregates current state across domains into a single digest.

Examples:
  hactl summary
  hactl summary --area "living room"
  hactl summary --plain`,
	RunE: func(cmd *cobra.Command, args []string) error {
		states, err := getClient().ListStates()
		if err != nil {
			return output.Err("%s", err)
		}

		// Apply entity filter before any domain/area processing.
		states = entityFilter.FilterStates(states)

		if summaryArea != "" {
			filtered := states[:0]
			for _, s := range states {
				if entityFilter.MatchesArea(s.EntityID, summaryArea) {
					filtered = append(filtered, s)
				}
			}
			states = filtered
		}

		summary := buildSummary(states)

		if quiet {
			return nil
		}
		if plain {
			output.PrintPlain(buildSummaryPlain(summary))
			return nil
		}
		return output.PrintJSON(summary)
	},
}

func init() {
	summaryCmd.Flags().StringVar(&summaryArea, "area", "", "filter to a specific area")
}

// DomainSummary holds a digest for one domain.
type DomainSummary struct {
	Domain    string         `json:"domain"`
	Total     int            `json:"total"`
	Active    int            `json:"active,omitempty"`
	Inactive  int            `json:"inactive,omitempty"`
	Alerts    []string       `json:"alerts,omitempty"`
	Entities  []EntityDigest `json:"entities"`
}

// EntityDigest is a compact view of one entity.
type EntityDigest struct {
	EntityID     string `json:"entity_id"`
	FriendlyName string `json:"friendly_name,omitempty"`
	State        string `json:"state"`
	Note         string `json:"note,omitempty"`
}

// Summary is the full digest.
type Summary struct {
	GeneratedAt time.Time        `json:"generated_at"`
	Domains     []*DomainSummary `json:"domains"`
	Alerts      []string         `json:"alerts,omitempty"`
}

func buildSummary(states []client.State) *Summary {
	now := time.Now()
	hour := now.Hour()
	isDaytime := hour >= 7 && hour < 21

	byDomain := map[string]*DomainSummary{}
	var globalAlerts []string

	for _, s := range states {
		parts := strings.SplitN(s.EntityID, ".", 2)
		if len(parts) != 2 {
			continue
		}
		domain := parts[0]

		// Only include interesting domains
		switch domain {
		case "light", "switch", "climate", "binary_sensor", "sensor",
			"lock", "cover", "media_player", "vacuum", "fan", "automation":
		default:
			continue
		}

		ds := byDomain[domain]
		if ds == nil {
			ds = &DomainSummary{Domain: domain}
			byDomain[domain] = ds
		}
		ds.Total++

		name := friendlyName(s)
		digest := EntityDigest{
			EntityID:     s.EntityID,
			FriendlyName: name,
			State:        s.State,
		}

		active := isActive(s)
		if active {
			ds.Active++
		} else {
			ds.Inactive++
		}

		// Domain-specific notes & alerts
		switch domain {
		case "light":
			if active {
				if b, ok := s.Attributes["brightness"]; ok {
					if bf, ok := toFloat(b); ok {
						pct := int(math.Round(bf / 255 * 100))
						digest.Note = fmt.Sprintf("%d%%", pct)
					}
				}
				if isDaytime {
					alert := fmt.Sprintf("light on during day: %s", name)
					ds.Alerts = append(ds.Alerts, alert)
					globalAlerts = append(globalAlerts, alert)
				}
			}

		case "climate":
			if temp, ok := s.Attributes["temperature"]; ok {
				if tf, ok := toFloat(temp); ok {
					digest.Note = fmt.Sprintf("setpoint %.1f°C", tf)
					if tf < 16 || tf > 26 {
						alert := fmt.Sprintf("unusual temperature setpoint %.1f°C on %s", tf, name)
						ds.Alerts = append(ds.Alerts, alert)
						globalAlerts = append(globalAlerts, alert)
					}
				}
			}
			if ct, ok := s.Attributes["current_temperature"]; ok {
				if ctf, ok := toFloat(ct); ok {
					digest.Note += fmt.Sprintf(" (actual %.1f°C)", ctf)
				}
			}

		case "lock":
			if s.State == "unlocked" {
				alert := fmt.Sprintf("lock open: %s", name)
				ds.Alerts = append(ds.Alerts, alert)
				globalAlerts = append(globalAlerts, alert)
				digest.Note = "UNLOCKED"
			}

		case "cover":
			if s.State == "open" {
				digest.Note = "open"
			}

		case "binary_sensor":
			if active {
				age := time.Since(s.LastChanged)
				if age < 10*time.Minute {
					digest.Note = fmt.Sprintf("triggered %s ago", formatAge(age))
				}
			}

		case "sensor":
			if unit, ok := s.Attributes["unit_of_measurement"].(string); ok && unit != "" {
				digest.Note = fmt.Sprintf("%s %s", s.State, unit)
			}

		case "media_player":
			if active {
				if title, ok := s.Attributes["media_title"].(string); ok && title != "" {
					digest.Note = title
				}
			}
		}

		ds.Entities = append(ds.Entities, digest)
	}

	// Build ordered list of domains
	domainOrder := []string{"light", "climate", "switch", "lock", "cover", "binary_sensor", "sensor", "media_player", "fan", "vacuum", "automation"}
	var domains []*DomainSummary
	seen := map[string]bool{}
	for _, d := range domainOrder {
		if ds, ok := byDomain[d]; ok {
			domains = append(domains, ds)
			seen[d] = true
		}
	}
	for d, ds := range byDomain {
		if !seen[d] {
			domains = append(domains, ds)
		}
	}

	return &Summary{
		GeneratedAt: now,
		Domains:     domains,
		Alerts:      globalAlerts,
	}
}

func buildSummaryPlain(s *Summary) string {
	parts := []string{}

	for _, ds := range s.Domains {
		if ds.Total == 0 {
			continue
		}
		switch ds.Domain {
		case "light":
			var on, off []string
			for _, e := range ds.Entities {
				n := e.FriendlyName
				if n == "" {
					n = e.EntityID
				}
				if isActiveState(e.State) {
					if e.Note != "" {
						on = append(on, fmt.Sprintf("%s %s", n, e.Note))
					} else {
						on = append(on, n)
					}
				} else {
					off = append(off, n)
				}
			}
			var sub []string
			if len(on) > 0 {
				sub = append(sub, fmt.Sprintf("%d on (%s)", len(on), strings.Join(on, ", ")))
			}
			if len(off) > 0 {
				sub = append(sub, fmt.Sprintf("%d off (%s)", len(off), strings.Join(off, ", ")))
			}
			if len(sub) > 0 {
				parts = append(parts, "lights: "+strings.Join(sub, ", "))
			}

		case "switch":
			var on, off []string
			for _, e := range ds.Entities {
				n := e.FriendlyName
				if n == "" {
					n = e.EntityID
				}
				if isActiveState(e.State) {
					on = append(on, n)
				} else {
					off = append(off, n)
				}
			}
			var sub []string
			if len(on) > 0 {
				sub = append(sub, fmt.Sprintf("%d on (%s)", len(on), strings.Join(on, ", ")))
			}
			if len(off) > 0 {
				sub = append(sub, fmt.Sprintf("%d off (%s)", len(off), strings.Join(off, ", ")))
			}
			if len(sub) > 0 {
				parts = append(parts, "switches: "+strings.Join(sub, ", "))
			}

		case "sensor":
			var readings []string
			for _, e := range ds.Entities {
				if e.Note == "" {
					continue // skip sensors without a unit
				}
				n := e.FriendlyName
				if n == "" {
					n = e.EntityID
				}
				readings = append(readings, fmt.Sprintf("%s: %s", n, e.Note))
			}
			if len(readings) > 0 {
				parts = append(parts, strings.Join(readings, ", "))
			}

		case "climate":
			for _, e := range ds.Entities {
				n := e.FriendlyName
				if n == "" {
					n = e.EntityID
				}
				parts = append(parts, fmt.Sprintf("%s %s %s", n, e.State, e.Note))
			}

		case "lock":
			for _, e := range ds.Entities {
				n := e.FriendlyName
				if n == "" {
					n = e.EntityID
				}
				parts = append(parts, fmt.Sprintf("%s %s", n, e.State))
			}

		case "binary_sensor":
			for _, e := range ds.Entities {
				if isActiveState(e.State) {
					n := e.FriendlyName
					if n == "" {
						n = e.EntityID
					}
					if e.Note != "" {
						parts = append(parts, fmt.Sprintf("motion in %s %s", n, e.Note))
					} else {
						parts = append(parts, fmt.Sprintf("%s active", n))
					}
				}
			}
		}
	}

	if len(parts) == 0 {
		return "everything looks normal"
	}
	result := strings.Join(parts, ", ")
	if len(s.Alerts) > 0 {
		result += " [ALERTS: " + strings.Join(s.Alerts, "; ") + "]"
	}
	return result
}

func isActive(s client.State) bool {
	return isActiveState(s.State)
}

func isActiveState(state string) bool {
	switch strings.ToLower(state) {
	case "on", "open", "unlocked", "playing", "home", "detected", "active", "cleaning":
		return true
	}
	return false
}

func friendlyName(s client.State) string {
	if name, ok := s.Attributes["friendly_name"].(string); ok && name != "" {
		return name
	}
	return s.EntityID
}

func formatAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
