package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/joaobarroca93/hactl/client"
	"github.com/spf13/viper"
)

// requireAllMode exits with a clear error if filter.mode is not "all".
// Call this at the top of any admin command's RunE.
func requireAllMode() {
	mode := viper.GetString("filter.mode")
	if mode != "all" {
		fmt.Fprintln(os.Stderr, "error: this command requires filter.mode: all in ~/.config/hactl/config.yaml")
		fmt.Fprintln(os.Stderr, "       these are admin operations — set filter.mode: all to proceed")
		os.Exit(1)
	}
}

// wsErrMsg extracts the human-readable message from a failed WS response.
func wsErrMsg(errMap map[string]any) string {
	if m, ok := errMap["message"].(string); ok && m != "" {
		return m
	}
	return "unknown error"
}

// wsCommand dials a WebSocket connection, sends the given payload, reads the
// response, and closes the connection. The payload must include a "type" key.
func wsCommand(payload map[string]any) (*client.WSMessage, error) {
	token := viper.GetString("hass_token")
	if token == "" {
		return nil, fmt.Errorf("HASS_TOKEN is required")
	}
	baseURL := viper.GetString("hass_url")
	if baseURL == "" {
		baseURL = "http://homeassistant.local:8123"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	ws, err := client.NewWS(baseURL, token)
	if err != nil {
		return nil, fmt.Errorf("websocket: %w", err)
	}
	defer ws.Close()

	return ws.CallCommand(payload)
}
