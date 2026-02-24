package cmd

import (
	"fmt"

	"github.com/joaobarroca/hactl/client"
	"github.com/joaobarroca/hactl/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var areaCmd = &cobra.Command{
	Use:   "area",
	Short: "List Home Assistant areas",
}

var areaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all areas defined in Home Assistant",
	RunE: func(cmd *cobra.Command, args []string) error {
		token := viper.GetString("hass_token")
		baseURL := viper.GetString("hass_url")
		if baseURL == "" {
			baseURL = "http://homeassistant.local:8123"
		}

		ws, err := client.NewWS(baseURL, token)
		if err != nil {
			return output.Err("websocket: %s", err)
		}
		defer ws.Close()

		areas, err := ws.FetchAreas()
		if err != nil {
			return output.Err("%s", err)
		}

		if quiet {
			return nil
		}
		if plain {
			for _, a := range areas {
				fmt.Printf("%s (id=%s)\n", a.Name, a.AreaID)
			}
			return nil
		}
		return output.PrintJSON(areas)
	},
}

func init() {
	areaCmd.AddCommand(areaListCmd)
	rootCmd.AddCommand(areaCmd)
}
