package cmd

import (
	"strings"
	"testing"
	"time"
)

// --- isActiveState ---

func TestIsActiveState(t *testing.T) {
	active := []string{"on", "open", "unlocked", "playing", "home", "detected", "active", "cleaning"}
	for _, s := range active {
		if !isActiveState(s) {
			t.Errorf("isActiveState(%q) = false, want true", s)
		}
		// case-insensitive
		if !isActiveState(strings.ToUpper(s)) {
			t.Errorf("isActiveState(%q) = false, want true (upper case)", strings.ToUpper(s))
		}
	}

	inactive := []string{"off", "closed", "locked", "idle", "unavailable", "unknown", ""}
	for _, s := range inactive {
		if isActiveState(s) {
			t.Errorf("isActiveState(%q) = true, want false", s)
		}
	}
}

// --- formatAge ---

func TestFormatAge(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{59 * time.Second, "59s"},
		{60 * time.Second, "1m"},
		{90 * time.Second, "1m"},
		{5 * time.Minute, "5m"},
	}
	for _, tt := range tests {
		got := formatAge(tt.d)
		if got != tt.want {
			t.Errorf("formatAge(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

// --- plural ---

func TestPlural(t *testing.T) {
	if plural(1) != "" {
		t.Error("plural(1) should be empty")
	}
	if plural(0) != "s" {
		t.Error("plural(0) should be 's'")
	}
	if plural(2) != "s" {
		t.Error("plural(2) should be 's'")
	}
}

// --- buildSummaryPlain ---

func TestBuildSummaryPlain_Empty(t *testing.T) {
	s := &Summary{}
	got := buildSummaryPlain(s)
	if got != "everything looks normal" {
		t.Errorf("empty summary = %q, want %q", got, "everything looks normal")
	}
}

func TestBuildSummaryPlain_LightsOnAndOff(t *testing.T) {
	s := &Summary{
		Domains: []*DomainSummary{
			{
				Domain: "light",
				Total:  2,
				Active: 1,
				Entities: []EntityDigest{
					{EntityID: "light.kitchen", FriendlyName: "Kitchen", State: "on", Note: "80%"},
					{EntityID: "light.bedroom", FriendlyName: "Bedroom", State: "off"},
				},
			},
		},
	}
	got := buildSummaryPlain(s)
	if !strings.Contains(got, "1 on") {
		t.Errorf("expected '1 on' in %q", got)
	}
	if !strings.Contains(got, "Kitchen 80%") {
		t.Errorf("expected 'Kitchen 80%%' in %q", got)
	}
	if !strings.Contains(got, "1 off") {
		t.Errorf("expected '1 off' in %q", got)
	}
	if !strings.Contains(got, "Bedroom") {
		t.Errorf("expected 'Bedroom' in %q", got)
	}
}

func TestBuildSummaryPlain_AllLightsOff(t *testing.T) {
	// Previously this returned "everything looks normal" — now it should list lights.
	s := &Summary{
		Domains: []*DomainSummary{
			{
				Domain: "light",
				Total:  1,
				Entities: []EntityDigest{
					{EntityID: "light.outdoor", FriendlyName: "Outdoor", State: "off"},
				},
			},
		},
	}
	got := buildSummaryPlain(s)
	if got == "everything looks normal" {
		t.Error("off lights should still appear in plain summary")
	}
	if !strings.Contains(got, "Outdoor") {
		t.Errorf("expected entity name in %q", got)
	}
}

func TestBuildSummaryPlain_Switches(t *testing.T) {
	s := &Summary{
		Domains: []*DomainSummary{
			{
				Domain: "switch",
				Total:  2,
				Active: 1,
				Entities: []EntityDigest{
					{EntityID: "switch.office", FriendlyName: "Office Light Switch", State: "on"},
					{EntityID: "switch.fan", FriendlyName: "Fan", State: "off"},
				},
			},
		},
	}
	got := buildSummaryPlain(s)
	if !strings.Contains(got, "switches:") {
		t.Errorf("expected 'switches:' in %q", got)
	}
	if !strings.Contains(got, "Office Light Switch") {
		t.Errorf("expected switch name in %q", got)
	}
}

func TestBuildSummaryPlain_SensorsWithUnits(t *testing.T) {
	s := &Summary{
		Domains: []*DomainSummary{
			{
				Domain: "sensor",
				Total:  2,
				Entities: []EntityDigest{
					{EntityID: "sensor.temp", FriendlyName: "Temperature", State: "21.5", Note: "21.5 °C"},
					{EntityID: "sensor.timestamp", FriendlyName: "Sun Dawn", State: "2026-02-25T06:00:00Z", Note: ""},
				},
			},
		},
	}
	got := buildSummaryPlain(s)
	if !strings.Contains(got, "Temperature: 21.5 °C") {
		t.Errorf("expected sensor reading in %q", got)
	}
	// Timestamp sensor with no unit should be skipped
	if strings.Contains(got, "Sun Dawn") {
		t.Errorf("sensor without unit should be omitted, got %q", got)
	}
}

func TestBuildSummaryPlain_UnlockedDoorAlert(t *testing.T) {
	s := &Summary{
		Domains: []*DomainSummary{
			{
				Domain: "lock",
				Total:  1,
				Entities: []EntityDigest{
					{EntityID: "lock.front_door", FriendlyName: "Front Door", State: "unlocked", Note: "UNLOCKED"},
				},
			},
		},
		Alerts: []string{"lock open: Front Door"},
	}
	got := buildSummaryPlain(s)
	if !strings.Contains(got, "Front Door") {
		t.Errorf("expected lock in %q", got)
	}
	if !strings.Contains(got, "ALERTS") {
		t.Errorf("expected ALERTS in %q", got)
	}
}

func TestBuildSummaryPlain_ActiveBinarySensor(t *testing.T) {
	s := &Summary{
		Domains: []*DomainSummary{
			{
				Domain: "binary_sensor",
				Total:  1,
				Active: 1,
				Entities: []EntityDigest{
					{EntityID: "binary_sensor.hall", FriendlyName: "Hallway", State: "detected", Note: "triggered 2m ago"},
				},
			},
		},
	}
	got := buildSummaryPlain(s)
	if !strings.Contains(got, "Hallway") {
		t.Errorf("expected binary sensor name in %q", got)
	}
}

func TestBuildSummaryPlain_InactiveBinarySensorHidden(t *testing.T) {
	s := &Summary{
		Domains: []*DomainSummary{
			{
				Domain: "binary_sensor",
				Total:  1,
				Entities: []EntityDigest{
					{EntityID: "binary_sensor.hall", FriendlyName: "Hallway", State: "clear"},
				},
			},
		},
	}
	got := buildSummaryPlain(s)
	if got != "everything looks normal" {
		t.Errorf("inactive binary sensor should not appear, got %q", got)
	}
}
