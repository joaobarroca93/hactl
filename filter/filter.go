package filter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaobarroca/hactl/client"
)

// Filter enforces entity visibility rules based on the configured mode.
type Filter struct {
	mode        string // "exposed" or "all"
	allowed     map[string]bool
	entityAreas map[string]string // entity_id -> area_id
}

// New creates a Filter with the given mode.
// If mode is "exposed" and skipCache is false, the exposed-entities cache is
// loaded from disk; the process exits with a clear error if the cache is missing.
// The entity-areas cache is always loaded when available (used for --area filtering).
// Callers must validate that mode is "exposed" or "all" before calling New.
func New(mode string, skipCache bool) *Filter {
	f := &Filter{mode: mode}
	if !skipCache {
		if mode == "exposed" {
			f.loadEntityCache()
		}
		f.loadAreasCache() // optional — no exit if missing
	}
	return f
}

// CachePath returns the path to the exposed-entities cache file.
func CachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "hactl", "exposed-entities.json"), nil
}

// AreasCachePath returns the path to the entity→area_id cache file.
func AreasCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "hactl", "entity-areas.json"), nil
}

func (f *Filter) loadEntityCache() {
	path, err := CachePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %s\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "error: No entity cache found. Run `hactl sync` first.")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "error: reading entity cache: %s\n", err)
		os.Exit(1)
	}

	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		fmt.Fprintf(os.Stderr, "error: parsing entity cache: %s\n", err)
		os.Exit(1)
	}

	f.allowed = make(map[string]bool, len(ids))
	for _, id := range ids {
		f.allowed[id] = true
	}
}

func (f *Filter) loadAreasCache() {
	path, err := AreasCachePath()
	if err != nil {
		return // non-fatal
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return // non-fatal: area cache is optional
	}

	var areas map[string]string
	if err := json.Unmarshal(data, &areas); err != nil {
		return // non-fatal
	}
	f.entityAreas = areas
}

// Mode returns the configured filter mode ("exposed" or "all").
func (f *Filter) Mode() string {
	return f.mode
}

// IsAllowed reports whether the given entity ID is permitted by the filter.
func (f *Filter) IsAllowed(entityID string) bool {
	if f.mode == "all" {
		return true
	}
	return f.allowed[entityID]
}

// FilterStates returns only the states whose entity IDs pass the filter.
func (f *Filter) FilterStates(states []client.State) []client.State {
	if f.mode == "all" {
		return states
	}
	out := make([]client.State, 0, len(states))
	for _, s := range states {
		if f.allowed[s.EntityID] {
			out = append(out, s)
		}
	}
	return out
}

// EntityAreaID returns the area_id assigned to entityID, or "" if unknown.
func (f *Filter) EntityAreaID(entityID string) string {
	return f.entityAreas[entityID]
}

// MatchesArea reports whether the entity belongs to the given area query.
// The query is matched case-insensitively against the entity's area_id.
func (f *Filter) MatchesArea(entityID, areaQuery string) bool {
	areaID := f.entityAreas[entityID]
	if areaID == "" {
		return false
	}
	return strings.EqualFold(areaID, areaQuery)
}
