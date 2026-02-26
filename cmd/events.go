package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joaobarroca93/hactl/client"
	"github.com/joaobarroca93/hactl/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	eventsType   string
	eventsDomain string
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Stream Home Assistant events",
}

var eventsWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Stream events to stdout as JSON lines",
	Long: `Connect to the Home Assistant WebSocket API and stream events.

Examples:
  hactl events watch
  hactl events watch --type state_changed
  hactl events watch --domain light
  hactl events watch --type state_changed --domain motion`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token := viper.GetString("hass_token")
		if token == "" {
			return output.Err("HASS_TOKEN is required")
		}
		baseURL := viper.GetString("hass_url")
		if baseURL == "" {
			baseURL = "http://homeassistant.local:8123"
		}

		ws, err := client.NewWS(baseURL, token)
		if err != nil {
			return output.Err("websocket: %s", err)
		}
		defer ws.Close()

		// Subscribe — if --type is given use it, otherwise subscribe to all events
		if err := ws.SubscribeEvents(eventsType); err != nil {
			return output.Err("failed to subscribe: %s", err)
		}

		// Read the subscription acknowledgement
		ack, err := ws.ReadMessage()
		if err != nil {
			return output.Err("websocket: %s", err)
		}
		if ack.Success != nil && !*ack.Success {
			return output.Err("subscription failed: %v", ack.Error)
		}

		if !quiet {
			fmt.Fprintf(os.Stderr, "connected, streaming events (Ctrl-C to stop)...\n")
		}

		// Handle SIGINT/SIGTERM gracefully
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			ws.Close()
			os.Exit(0)
		}()

		enc := json.NewEncoder(os.Stdout)

		for {
			raw, err := ws.ReadRaw()
			if err != nil {
				// Connection closed
				break
			}

			var msg map[string]any
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}

			msgType, _ := msg["type"].(string)
			if msgType != "event" {
				continue
			}

			event, _ := msg["event"].(map[string]any)
			if event == nil {
				continue
			}

			// Apply domain filter
			if eventsDomain != "" {
				if !matchesDomain(event, eventsDomain) {
					continue
				}
			}

			if quiet {
				continue
			}

			if plain {
				fmt.Println(formatEventPlain(event))
			} else {
				_ = enc.Encode(event)
			}
		}
		return nil
	},
}

func init() {
	eventsWatchCmd.Flags().StringVar(&eventsType, "type", "", "filter by event type (e.g. state_changed)")
	eventsWatchCmd.Flags().StringVar(&eventsDomain, "domain", "", "filter by entity domain (e.g. light, motion)")

	eventsCmd.AddCommand(eventsWatchCmd)
}

// matchesDomain checks if a state_changed event's entity_id matches the given domain.
func matchesDomain(event map[string]any, domain string) bool {
	data, _ := event["data"].(map[string]any)
	if data == nil {
		return false
	}
	entityID, _ := data["entity_id"].(string)
	return strings.HasPrefix(entityID, domain+".")
}

// formatEventPlain returns a compact human-readable description of an event.
func formatEventPlain(event map[string]any) string {
	eventType, _ := event["event_type"].(string)
	data, _ := event["data"].(map[string]any)

	if eventType == "state_changed" && data != nil {
		entityID, _ := data["entity_id"].(string)
		newState, _ := data["new_state"].(map[string]any)
		oldState, _ := data["old_state"].(map[string]any)

		var newStateStr, oldStateStr string
		if newState != nil {
			newStateStr, _ = newState["state"].(string)
		}
		if oldState != nil {
			oldStateStr, _ = oldState["state"].(string)
		}

		if oldStateStr != "" {
			return fmt.Sprintf("%s: %s -> %s", entityID, oldStateStr, newStateStr)
		}
		return fmt.Sprintf("%s: %s", entityID, newStateStr)
	}

	return fmt.Sprintf("event: %s", eventType)
}
