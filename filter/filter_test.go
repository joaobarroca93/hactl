package filter_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/joaobarroca/hactl/client"
	"github.com/joaobarroca/hactl/filter"
)

// setupCache writes entity and (optionally) area cache files under a temporary
// HOME directory, so tests never touch the real ~/.config/hactl/.
func setupCache(t *testing.T, ids []string, areas map[string]string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgDir := filepath.Join(home, ".config", "hactl")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	data, _ := json.Marshal(ids)
	if err := os.WriteFile(filepath.Join(cfgDir, "exposed-entities.json"), data, 0600); err != nil {
		t.Fatal(err)
	}

	if areas != nil {
		data, _ = json.Marshal(areas)
		if err := os.WriteFile(filepath.Join(cfgDir, "entity-areas.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

// --- Mode ---

func TestMode(t *testing.T) {
	for _, mode := range []string{"all", "exposed"} {
		f := filter.New(mode, true)
		if f.Mode() != mode {
			t.Errorf("Mode() = %q, want %q", f.Mode(), mode)
		}
	}
}

// --- IsAllowed ---

func TestIsAllowed_AllMode(t *testing.T) {
	f := filter.New("all", true)
	for _, id := range []string{"light.anything", "sensor.secret", "switch.hidden"} {
		if !f.IsAllowed(id) {
			t.Errorf("all mode: IsAllowed(%q) = false, want true", id)
		}
	}
}

func TestIsAllowed_ExposedMode(t *testing.T) {
	setupCache(t, []string{"light.allowed", "switch.allowed"}, nil)
	f := filter.New("exposed", false)

	if !f.IsAllowed("light.allowed") {
		t.Error("exposed entity should be allowed")
	}
	if !f.IsAllowed("switch.allowed") {
		t.Error("exposed entity should be allowed")
	}
	if f.IsAllowed("light.hidden") {
		t.Error("non-exposed entity should not be allowed")
	}
	if f.IsAllowed("sensor.secret") {
		t.Error("non-exposed entity should not be allowed")
	}
}

// --- FilterStates ---

func TestFilterStates_AllMode(t *testing.T) {
	f := filter.New("all", true)
	states := []client.State{
		{EntityID: "light.one"},
		{EntityID: "sensor.two"},
		{EntityID: "switch.three"},
	}
	got := f.FilterStates(states)
	if len(got) != len(states) {
		t.Errorf("all mode FilterStates: got %d states, want %d", len(got), len(states))
	}
}

func TestFilterStates_ExposedMode(t *testing.T) {
	setupCache(t, []string{"light.a", "switch.b"}, nil)
	f := filter.New("exposed", false)

	states := []client.State{
		{EntityID: "light.a"},
		{EntityID: "light.hidden"},
		{EntityID: "switch.b"},
		{EntityID: "sensor.secret"},
	}
	got := f.FilterStates(states)
	if len(got) != 2 {
		t.Fatalf("FilterStates: got %d states, want 2", len(got))
	}
	for _, s := range got {
		if !f.IsAllowed(s.EntityID) {
			t.Errorf("FilterStates returned non-allowed entity: %s", s.EntityID)
		}
	}
}

func TestFilterStates_Empty(t *testing.T) {
	setupCache(t, []string{"light.a"}, nil)
	f := filter.New("exposed", false)

	got := f.FilterStates(nil)
	if len(got) != 0 {
		t.Errorf("FilterStates(nil): got %d, want 0", len(got))
	}
}

// --- MatchesArea ---

func TestMatchesArea(t *testing.T) {
	setupCache(t,
		[]string{"switch.garage_door", "light.garage", "light.living_room"},
		map[string]string{
			"switch.garage_door": "garagem",
			"light.garage":       "garagem",
			"light.living_room":  "sala",
		},
	)
	f := filter.New("exposed", false)

	tests := []struct {
		entity string
		query  string
		want   bool
	}{
		{"switch.garage_door", "garagem", true},
		{"light.garage", "garagem", true},
		// case-insensitive
		{"switch.garage_door", "GARAGEM", true},
		{"switch.garage_door", "Garagem", true},
		// wrong area
		{"switch.garage_door", "sala", false},
		{"light.living_room", "garagem", false},
		// entity with no area
		{"sensor.no_area", "garagem", false},
		// empty query
		{"switch.garage_door", "", false},
	}

	for _, tt := range tests {
		got := f.MatchesArea(tt.entity, tt.query)
		if got != tt.want {
			t.Errorf("MatchesArea(%q, %q) = %v, want %v", tt.entity, tt.query, got, tt.want)
		}
	}
}

func TestMatchesArea_MissingAreasCache(t *testing.T) {
	// Areas cache not written — MatchesArea should return false, not panic.
	setupCache(t, []string{"switch.garage_door"}, nil)
	f := filter.New("exposed", false)

	if f.MatchesArea("switch.garage_door", "garagem") {
		t.Error("MatchesArea should return false when areas cache is missing")
	}
}

// --- EntityAreaID ---

func TestEntityAreaID(t *testing.T) {
	setupCache(t,
		[]string{"light.kitchen"},
		map[string]string{"light.kitchen": "kitchen_area"},
	)
	f := filter.New("exposed", false)

	if got := f.EntityAreaID("light.kitchen"); got != "kitchen_area" {
		t.Errorf("EntityAreaID = %q, want %q", got, "kitchen_area")
	}
	if got := f.EntityAreaID("light.unknown"); got != "" {
		t.Errorf("EntityAreaID for unknown entity = %q, want empty", got)
	}
}
