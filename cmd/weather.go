package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/joaobarroca/hactl/client"
	"github.com/joaobarroca/hactl/output"
	"github.com/spf13/cobra"
)

var weatherCmd = &cobra.Command{
	Use:   "weather [entity_id]",
	Short: "Show current weather conditions and forecast",
	Long: `Shows current conditions (temperature, humidity, wind) for a weather entity.
Includes forecast if available from the integration's state attributes.

Examples:
  hactl weather
  hactl weather weather.forecast_home
  hactl weather --plain`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var entityID string
		if len(args) == 1 {
			entityID = args[0]
			if !strings.HasPrefix(entityID, "weather.") {
				entityID = "weather." + entityID
			}
			if !entityFilter.IsAllowed(entityID) {
				return output.Err("entity not found: %s", entityID)
			}
		} else {
			// Find the first exposed weather entity
			states, err := getClient().ListStates()
			if err != nil {
				return output.Err("%s", err)
			}
			states = entityFilter.FilterStates(states)
			for _, s := range states {
				if strings.HasPrefix(s.EntityID, "weather.") {
					entityID = s.EntityID
					break
				}
			}
			if entityID == "" {
				return output.Err("no weather entity found")
			}
		}

		s, err := getClient().GetState(entityID)
		if err != nil {
			return output.Err("%s", err)
		}

		w := buildWeather(s)

		if quiet {
			return nil
		}
		if plain {
			output.PrintPlain(formatWeatherPlain(w))
			return nil
		}
		return output.PrintJSON(w)
	},
}

// WeatherConditions holds parsed weather data from a weather entity.
type WeatherConditions struct {
	EntityID    string           `json:"entity_id"`
	Condition   string           `json:"condition"`
	Temperature *float64         `json:"temperature,omitempty"`
	Humidity    *float64         `json:"humidity,omitempty"`
	WindSpeed   *float64         `json:"wind_speed,omitempty"`
	TempUnit    string           `json:"temperature_unit,omitempty"`
	WindUnit    string           `json:"wind_speed_unit,omitempty"`
	Forecast    []WeatherForecast `json:"forecast,omitempty"`
}

// WeatherForecast is a single forecast entry.
type WeatherForecast struct {
	Datetime      string   `json:"datetime"`
	Condition     string   `json:"condition,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
	TempLow       *float64 `json:"templow,omitempty"`
	Precipitation *float64 `json:"precipitation,omitempty"`
}

func buildWeather(s *client.State) *WeatherConditions {
	w := &WeatherConditions{
		EntityID:  s.EntityID,
		Condition: s.State,
	}

	if v, ok := s.Attributes["temperature"]; ok {
		if f, ok := toFloat(v); ok {
			w.Temperature = &f
		}
	}
	if v, ok := s.Attributes["humidity"]; ok {
		if f, ok := toFloat(v); ok {
			w.Humidity = &f
		}
	}
	if v, ok := s.Attributes["wind_speed"]; ok {
		if f, ok := toFloat(v); ok {
			w.WindSpeed = &f
		}
	}
	if u, ok := s.Attributes["temperature_unit"].(string); ok && u != "" {
		w.TempUnit = u
	}
	if u, ok := s.Attributes["wind_speed_unit"].(string); ok && u != "" {
		w.WindUnit = u
	}

	// Forecast may be present in attributes for many HA integrations.
	if raw, ok := s.Attributes["forecast"]; ok {
		if list, ok := raw.([]any); ok {
			for _, entry := range list {
				fm, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				fc := WeatherForecast{}
				if dt, ok := fm["datetime"].(string); ok {
					fc.Datetime = dt
				}
				if c, ok := fm["condition"].(string); ok {
					fc.Condition = c
				}
				if v, ok := fm["temperature"]; ok {
					if f, ok := toFloat(v); ok {
						fc.Temperature = &f
					}
				}
				if v, ok := fm["templow"]; ok {
					if f, ok := toFloat(v); ok {
						fc.TempLow = &f
					}
				}
				if v, ok := fm["precipitation"]; ok {
					if f, ok := toFloat(v); ok {
						fc.Precipitation = &f
					}
				}
				w.Forecast = append(w.Forecast, fc)
			}
		}
	}
	return w
}

func formatWeatherPlain(w *WeatherConditions) string {
	var parts []string

	parts = append(parts, w.Condition)

	if w.Temperature != nil {
		unit := w.TempUnit
		if unit == "" {
			unit = "°C"
		}
		parts = append(parts, fmt.Sprintf("%.1f%s", *w.Temperature, unit))
	}
	if w.Humidity != nil {
		parts = append(parts, fmt.Sprintf("humidity %.0f%%", *w.Humidity))
	}
	if w.WindSpeed != nil {
		unit := w.WindUnit
		if unit == "" {
			unit = "km/h"
		}
		parts = append(parts, fmt.Sprintf("wind %.1f %s", *w.WindSpeed, unit))
	}

	current := strings.Join(parts, ", ")

	if len(w.Forecast) == 0 {
		return current
	}

	limit := len(w.Forecast)
	if limit > 3 {
		limit = 3
	}
	var fparts []string
	for _, f := range w.Forecast[:limit] {
		day := parseForecastDay(f.Datetime)
		cond := f.Condition
		if f.Temperature != nil && f.TempLow != nil {
			fparts = append(fparts, fmt.Sprintf("%s %s %.0f/%.0f", day, cond, *f.Temperature, *f.TempLow))
		} else if f.Temperature != nil {
			fparts = append(fparts, fmt.Sprintf("%s %s %.0f", day, cond, *f.Temperature))
		} else {
			fparts = append(fparts, fmt.Sprintf("%s %s", day, cond))
		}
	}
	return current + "; forecast: " + strings.Join(fparts, ", ")
}

// parseForecastDay returns the abbreviated weekday name for a datetime string.
func parseForecastDay(dt string) string {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05-07:00", "2006-01-02"} {
		if t, err := time.Parse(layout, dt); err == nil {
			return t.Format("Mon")
		}
	}
	return dt
}

func init() {
	rootCmd.AddCommand(weatherCmd)
}
