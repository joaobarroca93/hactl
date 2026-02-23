package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/joaobarroca/hactl/client"
	"github.com/joaobarroca/hactl/output"
	"github.com/spf13/cobra"
)

var historyLast string

var historyCmd = &cobra.Command{
	Use:   "history <entity_id>",
	Short: "Show state history for an entity",
	Long: `Show state history for an entity over a time window.

Examples:
  hactl history sensor.temperature --last 1h
  hactl history light.living_room --last 24h --plain`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityID := args[0]

		duration, err := time.ParseDuration(historyLast)
		if err != nil {
			return output.Err("invalid --last value %q: use values like 1h, 30m, 24h", historyLast)
		}

		start := time.Now().Add(-duration)
		history, err := getClient().GetHistory(entityID, start, duration)
		if err != nil {
			return output.Err("%s", err)
		}

		if len(history) == 0 || len(history[0]) == 0 {
			if !quiet {
				if plain {
					output.PrintPlain(fmt.Sprintf("no history for %s in the last %s", entityID, historyLast))
				} else {
					_ = output.PrintJSON([]any{})
				}
			}
			return nil
		}

		entries := history[0]

		if quiet {
			return nil
		}
		if plain {
			output.PrintPlain(buildHistoryPlain(entries, historyLast))
			return nil
		}
		return output.PrintJSON(entries)
	},
}

func init() {
	historyCmd.Flags().StringVar(&historyLast, "last", "1h", "time window (e.g. 1h, 2h, 24h)")
}

// buildHistoryPlain returns compact prose describing state transitions.
// Example: "on at 08:32, off at 09:15, on at 14:20 (still on)"
func buildHistoryPlain(entries []client.HistoryEntry, window string) string {
	if len(entries) == 0 {
		return "no history"
	}

	type transition struct {
		state string
		t     time.Time
	}

	var transitions []transition
	var lastState string

	for _, e := range entries {
		if e.State != lastState {
			transitions = append(transitions, transition{state: e.State, t: e.LastChanged})
			lastState = e.State
		}
	}

	if len(transitions) == 0 {
		return "no state changes"
	}

	last := transitions[len(transitions)-1]
	parts := make([]string, 0, len(transitions))

	for i, tr := range transitions {
		timeStr := tr.t.Local().Format("15:04")
		if i == len(transitions)-1 {
			// Check if still in this state (last entry is recent)
			age := time.Since(last.t)
			if age < time.Duration(len(transitions))*time.Hour || i == 0 {
				parts = append(parts, fmt.Sprintf("%s at %s (still %s)", tr.state, timeStr, tr.state))
			} else {
				parts = append(parts, fmt.Sprintf("%s at %s", tr.state, timeStr))
			}
		} else {
			parts = append(parts, fmt.Sprintf("%s at %s", tr.state, timeStr))
		}
	}

	return strings.Join(parts, ", ")
}

