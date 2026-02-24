package cmd

import (
	"testing"
)

// --- parseValue ---

func TestParseValue(t *testing.T) {
	tests := []struct {
		in   string
		want any
	}{
		{"42", int64(42)},
		{"-7", int64(-7)},
		{"0", int64(0)},
		{"3.14", float64(3.14)},
		{"-0.5", float64(-0.5)},
		{"true", true},
		{"false", false},
		{"hello", "hello"},
		{"", ""},
		{"123abc", "123abc"},
	}
	for _, tt := range tests {
		got := parseValue(tt.in)
		if got != tt.want {
			t.Errorf("parseValue(%q) = %v (%T), want %v (%T)", tt.in, got, got, tt.want, tt.want)
		}
	}
}

// --- restricted services ---

func TestRestrictedServices(t *testing.T) {
	// These services must always be in the restricted set.
	must := []string{
		"homeassistant.restart",
		"homeassistant.stop",
	}
	for _, svc := range must {
		if !restrictedServices[svc] {
			t.Errorf("service %q missing from restrictedServices", svc)
		}
	}
}

func TestRestrictedServicesDoNotBlockNormal(t *testing.T) {
	normal := []string{
		"homeassistant.check_config",
		"light.turn_on",
		"switch.toggle",
	}
	for _, svc := range normal {
		if restrictedServices[svc] {
			t.Errorf("service %q should not be restricted", svc)
		}
	}
}
