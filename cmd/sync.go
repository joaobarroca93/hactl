package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/joaobarroca/hactl/client"
	"github.com/joaobarroca/hactl/filter"
	"github.com/joaobarroca/hactl/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch assist-exposed entities and write local cache",
	Long: `Connect to the Home Assistant WebSocket API, fetch entities exposed to HA Assist,
and write the list to ~/.config/hactl/exposed-entities.json.

Also writes entity→area mappings to ~/.config/hactl/entity-areas.json, which
is used by --area filtering in state list and summary.

Run this command whenever you change which entities are exposed in HA Assist.`,
	// Override PersistentPreRunE so filter cache is not required to run sync itself.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initClient(cmd.Name())
	},
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

		registry, err := ws.FetchEntityRegistry()
		if err != nil {
			return output.Err("fetch entity registry: %s", err)
		}

		// Write exposed entity IDs.
		cachePath, err := filter.CachePath()
		if err != nil {
			return output.Err("cannot determine home directory: %s", err)
		}
		entitiesData, err := json.Marshal(registry.ExposedIDs)
		if err != nil {
			return output.Err("marshal: %s", err)
		}
		if err := os.WriteFile(cachePath, entitiesData, 0600); err != nil {
			return output.Err("write entity cache: %s", err)
		}

		// Write entity→area_id mapping.
		areasCachePath, err := filter.AreasCachePath()
		if err != nil {
			return output.Err("cannot determine home directory: %s", err)
		}
		areasData, err := json.Marshal(registry.EntityAreas)
		if err != nil {
			return output.Err("marshal areas: %s", err)
		}
		if err := os.WriteFile(areasCachePath, areasData, 0600); err != nil {
			return output.Err("write areas cache: %s", err)
		}

		if !quiet {
			fmt.Printf("Synced %d exposed entities to %s\n", len(registry.ExposedIDs), cachePath)
			fmt.Printf("Synced %d entity→area mappings to %s\n", len(registry.EntityAreas), areasCachePath)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
