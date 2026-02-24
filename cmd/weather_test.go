package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/joaobarroca/hactl/client"
)

// --- buildWeather ---

func TestBuildWeather_Basic(t *testing.T) {
	s := &client.State{
		EntityID: "weather.forecast_home",
		State:    "sunny",
		Attributes: map[string]any{
			"temperature":      21.5,
			"humidity":         60.0,
			"wind_speed":       12.0,
			"temperature_unit": "°C",
			"wind_speed_unit":  "km/h",
		},
	}
	w := buildWeather(s)
	if w.Condition != "sunny" {
		t.Errorf("Condition = %q, want sunny", w.Condition)
	}
	if w.Temperature == nil || *w.Temperature != 21.5 {
		t.Errorf("Temperature = %v, want 21.5", w.Temperature)
	}
	if w.Humidity == nil || *w.Humidity != 60.0 {
		t.Errorf("Humidity = %v, want 60.0", w.Humidity)
	}
	if w.WindSpeed == nil || *w.WindSpeed != 12.0 {
		t.Errorf("WindSpeed = %v, want 12.0", w.WindSpeed)
	}
	if w.TempUnit != "°C" {
		t.Errorf("TempUnit = %q, want °C", w.TempUnit)
	}
	if len(w.Forecast) != 0 {
		t.Errorf("expected no forecast, got %d entries", len(w.Forecast))
	}
}

func TestBuildWeather_WithForecast(t *testing.T) {
	hi := 22.0
	lo := 14.0
	s := &client.State{
		EntityID: "weather.forecast_home",
		State:    "cloudy",
		Attributes: map[string]any{
			"forecast": []any{
				map[string]any{
					"datetime":    "2026-02-25T12:00:00+00:00",
					"condition":   "rainy",
					"temperature": hi,
					"templow":     lo,
				},
			},
		},
	}
	w := buildWeather(s)
	if len(w.Forecast) != 1 {
		t.Fatalf("expected 1 forecast entry, got %d", len(w.Forecast))
	}
	f := w.Forecast[0]
	if f.Condition != "rainy" {
		t.Errorf("forecast condition = %q, want rainy", f.Condition)
	}
	if f.Temperature == nil || *f.Temperature != 22.0 {
		t.Errorf("forecast temperature = %v, want 22.0", f.Temperature)
	}
	if f.TempLow == nil || *f.TempLow != 14.0 {
		t.Errorf("forecast templow = %v, want 14.0", f.TempLow)
	}
}

func TestBuildWeather_NoAttributes(t *testing.T) {
	s := &client.State{
		EntityID:   "weather.forecast_home",
		State:      "unknown",
		Attributes: map[string]any{},
	}
	w := buildWeather(s)
	if w.Condition != "unknown" {
		t.Errorf("Condition = %q, want unknown", w.Condition)
	}
	if w.Temperature != nil {
		t.Error("expected nil Temperature")
	}
	if w.Humidity != nil {
		t.Error("expected nil Humidity")
	}
}

// --- formatWeatherPlain ---

func TestFormatWeatherPlain_CurrentOnly(t *testing.T) {
	temp := 21.5
	hum := 60.0
	w := &WeatherConditions{
		EntityID:    "weather.home",
		Condition:   "sunny",
		Temperature: &temp,
		Humidity:    &hum,
		TempUnit:    "°C",
	}
	got := formatWeatherPlain(w)
	if !strings.Contains(got, "sunny") {
		t.Errorf("expected condition in %q", got)
	}
	if !strings.Contains(got, "21.5°C") {
		t.Errorf("expected temperature in %q", got)
	}
	if !strings.Contains(got, "humidity 60%") {
		t.Errorf("expected humidity in %q", got)
	}
	if strings.Contains(got, "forecast") {
		t.Errorf("unexpected 'forecast' in %q", got)
	}
}

func TestFormatWeatherPlain_WithForecast(t *testing.T) {
	temp := 20.0
	hi := 22.0
	lo := 13.0
	w := &WeatherConditions{
		EntityID:  "weather.home",
		Condition: "cloudy",
		Forecast: []WeatherForecast{
			{Datetime: "2026-02-25T00:00:00+00:00", Condition: "rainy", Temperature: &hi, TempLow: &lo},
			{Datetime: "2026-02-26T00:00:00+00:00", Condition: "sunny", Temperature: &temp},
		},
	}
	got := formatWeatherPlain(w)
	if !strings.Contains(got, "forecast:") {
		t.Errorf("expected 'forecast:' in %q", got)
	}
	if !strings.Contains(got, "rainy") {
		t.Errorf("expected rainy forecast in %q", got)
	}
	if !strings.Contains(got, "22/13") {
		t.Errorf("expected hi/lo in %q", got)
	}
}

func TestFormatWeatherPlain_ForecastCappedAt3(t *testing.T) {
	temp := 20.0
	forecasts := make([]WeatherForecast, 5)
	for i := range forecasts {
		t2 := time.Date(2026, 2, 25+i, 12, 0, 0, 0, time.UTC)
		forecasts[i] = WeatherForecast{
			Datetime:    t2.Format(time.RFC3339),
			Condition:   "sunny",
			Temperature: &temp,
		}
	}
	w := &WeatherConditions{
		EntityID:  "weather.home",
		Condition: "sunny",
		Forecast:  forecasts,
	}
	got := formatWeatherPlain(w)
	// Count "Sun/Mon/Tue/Wed" etc — should be at most 3
	parts := strings.Split(got, "forecast: ")
	if len(parts) < 2 {
		t.Fatalf("expected forecast section in %q", got)
	}
	forecastPart := parts[1]
	// Each forecast entry is separated by ", "
	entries := strings.Split(forecastPart, ", ")
	if len(entries) > 3 {
		t.Errorf("forecast capped at 3, got %d entries in %q", len(entries), forecastPart)
	}
}

// --- parseForecastDay ---

func TestParseForecastDay_RFC3339(t *testing.T) {
	got := parseForecastDay("2026-02-25T12:00:00Z")
	// 2026-02-25 is a Wednesday
	if got != "Wed" {
		t.Errorf("parseForecastDay = %q, want Wed", got)
	}
}

func TestParseForecastDay_DateOnly(t *testing.T) {
	got := parseForecastDay("2026-02-25")
	if got != "Wed" {
		t.Errorf("parseForecastDay = %q, want Wed", got)
	}
}

func TestParseForecastDay_Unparseable(t *testing.T) {
	got := parseForecastDay("not-a-date")
	if got != "not-a-date" {
		t.Errorf("unparseable should return as-is, got %q", got)
	}
}
