package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// State represents a Home Assistant entity state.
type State struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes"`
	LastChanged time.Time      `json:"last_changed"`
	LastUpdated time.Time      `json:"last_updated"`
	Context     map[string]any `json:"context,omitempty"`
}

// HistoryEntry is a single point in an entity's history.
type HistoryEntry struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes"`
	LastChanged time.Time      `json:"last_changed"`
	LastUpdated time.Time      `json:"last_updated"`
}

// Client wraps resty for Home Assistant REST calls.
type Client struct {
	r       *resty.Client
	baseURL string
}

// New creates a REST client pointed at baseURL, authenticated with token.
func New(baseURL, token string) *Client {
	r := resty.New().
		SetBaseURL(strings.TrimRight(baseURL, "/")).
		SetHeader("Authorization", "Bearer "+token).
		SetHeader("Content-Type", "application/json").
		SetTimeout(10 * time.Second)

	return &Client{r: r, baseURL: strings.TrimRight(baseURL, "/")}
}

// Ping calls GET /api/ to verify connectivity.
func (c *Client) Ping() error {
	resp, err := c.r.R().Get("/api/")
	if err != nil {
		return fmt.Errorf("connection refused: %w", err)
	}
	if resp.StatusCode() == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: check your HASS_TOKEN")
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode())
	}
	return nil
}

// GetState fetches a single entity state.
func (c *Client) GetState(entityID string) (*State, error) {
	resp, err := c.r.R().Get("/api/states/" + entityID)
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("entity not found: %s", entityID)
	}
	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: check your HASS_TOKEN")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode(), resp.String())
	}
	var s State
	if err := json.Unmarshal(resp.Body(), &s); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &s, nil
}

// SetState posts a new state for an entity.
func (c *Client) SetState(entityID, state string, attributes map[string]any) (*State, error) {
	body := map[string]any{"state": state}
	if len(attributes) > 0 {
		body["attributes"] = attributes
	}
	resp, err := c.r.R().SetBody(body).Post("/api/states/" + entityID)
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}
	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: check your HASS_TOKEN")
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode(), resp.String())
	}
	var s State
	if err := json.Unmarshal(resp.Body(), &s); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &s, nil
}

// ListStates fetches all entity states.
func (c *Client) ListStates() ([]State, error) {
	resp, err := c.r.R().Get("/api/states")
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}
	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: check your HASS_TOKEN")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode(), resp.String())
	}
	var states []State
	if err := json.Unmarshal(resp.Body(), &states); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return states, nil
}

// CallService calls a HA service with the given data payload.
func (c *Client) CallService(domain, service string, data map[string]any) ([]State, error) {
	resp, err := c.r.R().SetBody(data).Post(fmt.Sprintf("/api/services/%s/%s", domain, service))
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}
	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: check your HASS_TOKEN")
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("service not found: %s.%s", domain, service)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode(), resp.String())
	}
	var states []State
	if err := json.Unmarshal(resp.Body(), &states); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return states, nil
}

// GetHistory fetches the history for an entity over a time range.
// start is an RFC3339 timestamp; duration is added to produce the end time.
func (c *Client) GetHistory(entityID string, start time.Time, duration time.Duration) ([][]HistoryEntry, error) {
	end := start.Add(duration)
	req := c.r.R().
		SetQueryParam("filter_entity_id", entityID).
		SetQueryParam("end_time", end.UTC().Format(time.RFC3339))

	resp, err := req.Get("/api/history/period/" + start.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}
	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: check your HASS_TOKEN")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode(), resp.String())
	}
	var history [][]HistoryEntry
	if err := json.Unmarshal(resp.Body(), &history); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return history, nil
}

// GetConfig fetches the HA configuration.
func (c *Client) GetConfig() (map[string]any, error) {
	resp, err := c.r.R().Get("/api/config")
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode(), resp.String())
	}
	var cfg map[string]any
	if err := json.Unmarshal(resp.Body(), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return cfg, nil
}

// BaseURL returns the configured base URL (useful for WebSocket client).
func (c *Client) BaseURL() string {
	return c.baseURL
}

// TodoItem represents a single item in a Home Assistant todo list.
type TodoItem struct {
	UID         string `json:"uid"`
	Summary     string `json:"summary"`
	Status      string `json:"status"` // "needs_action" or "completed"
	Description string `json:"description,omitempty"`
	Due         string `json:"due,omitempty"`
}

// GetTodoItems fetches all items from a todo list entity.
// It uses the todo.get_items service with return_response=true (requires HA 2023.11+).
func (c *Client) GetTodoItems(entityID string) ([]TodoItem, error) {
	body := map[string]any{
		"entity_id": entityID,
	}
	resp, err := c.r.R().SetBody(body).SetQueryParam("return_response", "true").Post("/api/services/todo/get_items")
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}
	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: check your HASS_TOKEN")
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("todo service not found (requires HA 2023.11+)")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode(), resp.String())
	}

	// HA 2024.x+ wraps the service result under a "response" key:
	//   {"response": {"entity_id": {"items": [...]}}, "changed_states": [...]}
	// Older versions return the entity map directly:
	//   {"entity_id": {"items": [...]}}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(resp.Body(), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse todo response: %w", err)
	}
	// HA wraps the result under "service_response"; fall back to direct map for
	// any future format changes.
	entityMap := raw
	for _, key := range []string{"service_response", "response"} {
		if data, ok := raw[key]; ok {
			var inner map[string]json.RawMessage
			if err := json.Unmarshal(data, &inner); err == nil {
				entityMap = inner
				break
			}
		}
	}
	entityData, ok := entityMap[entityID]
	if !ok {
		return nil, nil
	}
	var result struct {
		Items []TodoItem `json:"items"`
	}
	if err := json.Unmarshal(entityData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse todo items: %w", err)
	}
	return result.Items, nil
}
