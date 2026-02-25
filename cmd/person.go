package cmd

import (
	"fmt"
	"strings"

	"github.com/joaobarroca93/hactl/output"
	"github.com/spf13/cobra"
)

var personCmd = &cobra.Command{
	Use:   "person",
	Short: "Show person locations",
}

var personListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all persons with their home/away status",
	RunE: func(cmd *cobra.Command, args []string) error {
		states, err := getClient().ListStates()
		if err != nil {
			return output.Err("%s", err)
		}
		states = entityFilter.FilterStates(states)

		type PersonInfo struct {
			EntityID     string `json:"entity_id"`
			FriendlyName string `json:"friendly_name,omitempty"`
			State        string `json:"state"`
		}

		var persons []PersonInfo
		for _, s := range states {
			if !strings.HasPrefix(s.EntityID, "person.") {
				continue
			}
			p := PersonInfo{
				EntityID: s.EntityID,
				State:    s.State,
			}
			if name, ok := s.Attributes["friendly_name"].(string); ok {
				p.FriendlyName = name
			}
			persons = append(persons, p)
		}

		if quiet {
			return nil
		}
		if plain {
			for _, p := range persons {
				name := p.FriendlyName
				if name == "" {
					name = p.EntityID
				}
				fmt.Printf("%s: %s\n", name, p.State)
			}
			return nil
		}
		return output.PrintJSON(persons)
	},
}

func init() {
	personCmd.AddCommand(personListCmd)
	rootCmd.AddCommand(personCmd)
}
