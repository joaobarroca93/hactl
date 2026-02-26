package cmd

import (
	"testing"
)

// --- toFloat ---

func TestToFloat(t *testing.T) {
	tests := []struct {
		in   any
		want float64
		ok   bool
	}{
		{float64(128), 128, true},
		{float64(0), 0, true},
		{int(10), 10, true},
		{int64(20), 20, true},
		{"string", 0, false},
		{nil, 0, false},
		{true, 0, false},
	}
	for _, tt := range tests {
		got, ok := toFloat(tt.in)
		if ok != tt.ok {
			t.Errorf("toFloat(%v): ok=%v, want %v", tt.in, ok, tt.ok)
		}
		if ok && got != tt.want {
			t.Errorf("toFloat(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

// --- formatAttrsPlain ---

func TestFormatAttrsPlain(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]any
		want  string
	}{
		{
			name:  "empty",
			attrs: map[string]any{},
			want:  "",
		},
		{
			name:  "friendly_name only",
			attrs: map[string]any{"friendly_name": "Kitchen Light"},
			want:  "Kitchen Light",
		},
		{
			name:  "brightness only — 100%",
			attrs: map[string]any{"brightness": float64(255)},
			want:  "brightness 100%",
		},
		{
			name:  "brightness only — 50%",
			attrs: map[string]any{"brightness": float64(127)},
			want:  "brightness 49%",
		},
		{
			name:  "friendly name + brightness",
			attrs: map[string]any{"friendly_name": "Lamp", "brightness": float64(255)},
			want:  "Lamp, brightness 100%",
		},
		{
			name:  "temperature",
			attrs: map[string]any{"temperature": float64(21.5)},
			want:  "21.5°C",
		},
		{
			name:  "current_temperature",
			attrs: map[string]any{"current_temperature": float64(19.0)},
			want:  "current 19.0°C",
		},
		{
			name:  "friendly name + temperature",
			attrs: map[string]any{"friendly_name": "Bedroom AC", "temperature": float64(22.0)},
			want:  "Bedroom AC, 22.0°C",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAttrsPlain(tt.attrs)
			if got != tt.want {
				t.Errorf("formatAttrsPlain() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- serviceControlled blocklist ---

func TestServiceControlledBlocklist(t *testing.T) {
	// These domains must always be in the blocklist — state set should reject them.
	required := []string{
		"light", "switch", "climate", "cover", "fan",
		"media_player", "vacuum", "lock", "button", "scene",
		"script", "alarm_control_panel", "siren",
	}
	for _, domain := range required {
		if _, ok := serviceControlled[domain]; !ok {
			t.Errorf("domain %q missing from serviceControlled blocklist", domain)
		}
	}
}

func TestServiceControlledHints(t *testing.T) {
	// Each blocked domain must have a non-empty hint string.
	for domain, hint := range serviceControlled {
		if hint == "" {
			t.Errorf("serviceControlled[%q] has empty hint", domain)
		}
	}
}
