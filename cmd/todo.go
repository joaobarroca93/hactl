package cmd

import (
	"fmt"
	"strings"

	"github.com/joaobarroca93/hactl/client"
	"github.com/joaobarroca93/hactl/output"
	"github.com/spf13/cobra"
)

var todoCmd = &cobra.Command{
	Use:   "todo",
	Short: "Manage Home Assistant todo lists",
}

var todoListCmd = &cobra.Command{
	Use:   "list [entity_id]",
	Short: "List items in a todo list (or all todo lists if no entity given)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var entityIDs []string
		if len(args) == 1 {
			eid := ensureTodoPrefix(args[0])
			if !entityFilter.IsAllowed(eid) {
				return output.Err("entity not found: %s", eid)
			}
			entityIDs = append(entityIDs, eid)
		} else {
			states, err := getClient().ListStates()
			if err != nil {
				return output.Err("%s", err)
			}
			states = entityFilter.FilterStates(states)
			for _, s := range states {
				if strings.HasPrefix(s.EntityID, "todo.") {
					entityIDs = append(entityIDs, s.EntityID)
				}
			}
			if len(entityIDs) == 0 {
				if !quiet {
					output.PrintPlain("no todo lists found")
				}
				return nil
			}
		}

		type listResult struct {
			EntityID string            `json:"entity_id"`
			Items    []client.TodoItem `json:"items"`
		}

		var results []listResult
		for _, eid := range entityIDs {
			items, err := getClient().GetTodoItems(eid)
			if err != nil {
				return output.Err("%s: %s", eid, err)
			}
			results = append(results, listResult{EntityID: eid, Items: items})
		}

		if quiet {
			return nil
		}
		if plain {
			multi := len(results) > 1
			for _, r := range results {
				if multi {
					fmt.Printf("%s:\n", r.EntityID)
				}
				for _, item := range r.Items {
					status := "[ ]"
					if item.Status == "completed" {
						status = "[x]"
					}
					fmt.Printf("  %s %s\n", status, item.Summary)
				}
			}
			return nil
		}
		if len(results) == 1 {
			return output.PrintJSON(results[0].Items)
		}
		return output.PrintJSON(results)
	},
}

var todoAddCmd = &cobra.Command{
	Use:   "add <entity_id> <item>",
	Short: "Add an item to a todo list",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityID := ensureTodoPrefix(args[0])
		item := args[1]
		if !entityFilter.IsAllowed(entityID) {
			return output.Err("entity not found: %s", entityID)
		}
		_, err := getClient().CallService("todo", "add_item", map[string]any{
			"entity_id": entityID,
			"item":      item,
		})
		if err != nil {
			return output.Err("%s", err)
		}
		if quiet {
			return nil
		}
		if plain {
			output.PrintPlain(fmt.Sprintf("added %q to %s", item, entityID))
			return nil
		}
		return output.PrintJSON(map[string]string{"added": item, "list": entityID})
	},
}

var todoDoneCmd = &cobra.Command{
	Use:   "done <entity_id> <item>",
	Short: "Mark a todo item as completed",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityID := ensureTodoPrefix(args[0])
		item := args[1]
		if !entityFilter.IsAllowed(entityID) {
			return output.Err("entity not found: %s", entityID)
		}
		_, err := getClient().CallService("todo", "update_item", map[string]any{
			"entity_id": entityID,
			"item":      item,
			"status":    "completed",
		})
		if err != nil {
			return output.Err("%s", err)
		}
		if quiet {
			return nil
		}
		if plain {
			output.PrintPlain(fmt.Sprintf("marked %q as done in %s", item, entityID))
			return nil
		}
		return output.PrintJSON(map[string]string{"completed": item, "list": entityID})
	},
}

var todoRemoveCmd = &cobra.Command{
	Use:   "remove <entity_id> <item>",
	Short: "Remove an item from a todo list",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityID := ensureTodoPrefix(args[0])
		item := args[1]
		if !entityFilter.IsAllowed(entityID) {
			return output.Err("entity not found: %s", entityID)
		}
		_, err := getClient().CallService("todo", "remove_item", map[string]any{
			"entity_id": entityID,
			"item":      item,
		})
		if err != nil {
			return output.Err("%s", err)
		}
		if quiet {
			return nil
		}
		if plain {
			output.PrintPlain(fmt.Sprintf("removed %q from %s", item, entityID))
			return nil
		}
		return output.PrintJSON(map[string]string{"removed": item, "list": entityID})
	},
}

func init() {
	todoCmd.AddCommand(todoListCmd)
	todoCmd.AddCommand(todoAddCmd)
	todoCmd.AddCommand(todoDoneCmd)
	todoCmd.AddCommand(todoRemoveCmd)
	rootCmd.AddCommand(todoCmd)
}

// ensureTodoPrefix adds the "todo." prefix if not already present.
func ensureTodoPrefix(id string) string {
	if strings.HasPrefix(id, "todo.") {
		return id
	}
	return "todo." + id
}
