package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage is a generic WebSocket message from HA.
type WSMessage struct {
	ID      int            `json:"id,omitempty"`
	Type    string         `json:"type"`
	Event   map[string]any `json:"event,omitempty"`
	Result  any            `json:"result,omitempty"`
	Success *bool          `json:"success,omitempty"`
	Error   map[string]any `json:"error,omitempty"`
}

// WSClient handles the Home Assistant WebSocket API.
type WSClient struct {
	conn    *websocket.Conn
	token   string
	counter int
}

// NewWS connects and authenticates to the HA WebSocket API.
func NewWS(baseURL, token string) (*WSClient, error) {
	wsURL := toWSURL(baseURL)
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(wsURL+"/api/websocket", nil)
	if err != nil {
		return nil, fmt.Errorf("websocket connection failed: %w", err)
	}

	ws := &WSClient{conn: conn, token: token}
	if err := ws.authenticate(); err != nil {
		conn.Close()
		return nil, err
	}
	return ws, nil
}

func (ws *WSClient) authenticate() error {
	// Step 1: receive auth_required
	var msg WSMessage
	if err := ws.conn.ReadJSON(&msg); err != nil {
		return fmt.Errorf("websocket read error: %w", err)
	}
	if msg.Type != "auth_required" {
		return fmt.Errorf("expected auth_required, got: %s", msg.Type)
	}

	// Step 2: send auth
	authMsg := map[string]string{"type": "auth", "access_token": ws.token}
	if err := ws.conn.WriteJSON(authMsg); err != nil {
		return fmt.Errorf("websocket write error: %w", err)
	}

	// Step 3: receive auth result
	if err := ws.conn.ReadJSON(&msg); err != nil {
		return fmt.Errorf("websocket read error: %w", err)
	}
	if msg.Type == "auth_invalid" {
		return fmt.Errorf("websocket authentication failed: invalid token")
	}
	if msg.Type != "auth_ok" {
		return fmt.Errorf("unexpected auth response: %s", msg.Type)
	}
	return nil
}

// SubscribeEvents subscribes to HA events, optionally filtered by eventType.
// Pass empty string to receive all events.
func (ws *WSClient) SubscribeEvents(eventType string) error {
	ws.counter++
	msg := map[string]any{
		"id":   ws.counter,
		"type": "subscribe_events",
	}
	if eventType != "" {
		msg["event_type"] = eventType
	}
	return ws.conn.WriteJSON(msg)
}

// ReadMessage reads the next message from the WebSocket.
func (ws *WSClient) ReadMessage() (*WSMessage, error) {
	var msg WSMessage
	if err := ws.conn.ReadJSON(&msg); err != nil {
		return nil, fmt.Errorf("websocket read error: %w", err)
	}
	return &msg, nil
}

// ReadRaw reads the next message as raw JSON bytes.
func (ws *WSClient) ReadRaw() ([]byte, error) {
	_, data, err := ws.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("websocket read error: %w", err)
	}
	return data, nil
}

// WriteJSON sends a JSON message.
func (ws *WSClient) WriteJSON(v any) error {
	return ws.conn.WriteJSON(v)
}

// Close closes the WebSocket connection.
func (ws *WSClient) Close() error {
	return ws.conn.Close()
}

// MarshalEvent returns a compact JSON representation of an event message.
func MarshalEvent(msg *WSMessage) ([]byte, error) {
	return json.Marshal(msg.Event)
}

// Area represents a Home Assistant area from the area registry.
type Area struct {
	AreaID  string `json:"area_id"`
	Name    string `json:"name"`
	Picture string `json:"picture,omitempty"`
}

// FetchAreas queries the area registry and returns all defined areas.
func (ws *WSClient) FetchAreas() ([]Area, error) {
	ws.counter++
	if err := ws.conn.WriteJSON(map[string]any{
		"id":   ws.counter,
		"type": "config/area_registry/list",
	}); err != nil {
		return nil, fmt.Errorf("websocket write error: %w", err)
	}

	raw, err := ws.ReadRaw()
	if err != nil {
		return nil, err
	}

	var msg struct {
		Success bool           `json:"success"`
		Result  []Area         `json:"result"`
		Error   map[string]any `json:"error,omitempty"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse area registry response: %w", err)
	}
	if !msg.Success {
		if msg.Error != nil {
			return nil, fmt.Errorf("area registry request failed: %v", msg.Error)
		}
		return nil, fmt.Errorf("area registry request failed")
	}
	return msg.Result, nil
}

// entityRegistryEntry is one record from config/entity_registry/list.
type entityRegistryEntry struct {
	EntityID string `json:"entity_id"`
	DeviceID string `json:"device_id"`
	AreaID   string `json:"area_id"` // set only when area is assigned directly to entity
	Options  struct {
		Conversation struct {
			ShouldExpose bool `json:"should_expose"`
		} `json:"conversation"`
	} `json:"options"`
}

// deviceRegistryEntry is one record from config/device_registry/list.
type deviceRegistryEntry struct {
	ID     string `json:"id"`
	AreaID string `json:"area_id"`
}

// EntityRegistryData holds the combined results of entity + device registry calls.
type EntityRegistryData struct {
	// ExposedIDs are entity IDs exposed to HA Assist.
	ExposedIDs []string
	// EntityAreas maps entity_id to area_id, resolving device-level area inheritance.
	EntityAreas map[string]string
}

// FetchEntityRegistry fetches the entity and device registries and returns
// exposed entity IDs plus the effective entity→area_id mapping.
// Area resolution: entity.area_id takes priority; falls back to the area of
// the entity's parent device (the common case in HA).
func (ws *WSClient) FetchEntityRegistry() (*EntityRegistryData, error) {
	// --- entity registry ---
	ws.counter++
	if err := ws.conn.WriteJSON(map[string]any{
		"id":   ws.counter,
		"type": "config/entity_registry/list",
	}); err != nil {
		return nil, fmt.Errorf("websocket write error: %w", err)
	}
	raw, err := ws.ReadRaw()
	if err != nil {
		return nil, err
	}
	var entMsg struct {
		Success bool                  `json:"success"`
		Result  []entityRegistryEntry `json:"result"`
		Error   map[string]any        `json:"error,omitempty"`
	}
	if err := json.Unmarshal(raw, &entMsg); err != nil {
		return nil, fmt.Errorf("failed to parse entity registry response: %w", err)
	}
	if !entMsg.Success {
		if entMsg.Error != nil {
			return nil, fmt.Errorf("entity registry request failed: %v", entMsg.Error)
		}
		return nil, fmt.Errorf("entity registry request failed")
	}

	// --- device registry ---
	ws.counter++
	if err := ws.conn.WriteJSON(map[string]any{
		"id":   ws.counter,
		"type": "config/device_registry/list",
	}); err != nil {
		return nil, fmt.Errorf("websocket write error: %w", err)
	}
	raw, err = ws.ReadRaw()
	if err != nil {
		return nil, err
	}
	var devMsg struct {
		Success bool                  `json:"success"`
		Result  []deviceRegistryEntry `json:"result"`
		Error   map[string]any        `json:"error,omitempty"`
	}
	if err := json.Unmarshal(raw, &devMsg); err != nil {
		return nil, fmt.Errorf("failed to parse device registry response: %w", err)
	}
	// device registry failure is non-fatal — area fallback simply won't work
	deviceArea := make(map[string]string, len(devMsg.Result))
	if devMsg.Success {
		for _, d := range devMsg.Result {
			if d.AreaID != "" {
				deviceArea[d.ID] = d.AreaID
			}
		}
	}

	// --- resolve effective area per entity (exposed entities only) ---
	data := &EntityRegistryData{
		EntityAreas: make(map[string]string),
	}
	for _, e := range entMsg.Result {
		if !e.Options.Conversation.ShouldExpose {
			continue
		}
		data.ExposedIDs = append(data.ExposedIDs, e.EntityID)
		areaID := e.AreaID
		if areaID == "" {
			areaID = deviceArea[e.DeviceID]
		}
		if areaID != "" {
			data.EntityAreas[e.EntityID] = areaID
		}
	}
	return data, nil
}

// CallCommand sends a generic command and reads the result.
// The payload must include a "type" key. The message ID is set automatically.
func (ws *WSClient) CallCommand(payload map[string]any) (*WSMessage, error) {
	ws.counter++
	payload["id"] = ws.counter
	if err := ws.conn.WriteJSON(payload); err != nil {
		return nil, fmt.Errorf("websocket write error: %w", err)
	}
	raw, err := ws.ReadRaw()
	if err != nil {
		return nil, err
	}
	var msg WSMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &msg, nil
}

// toWSURL converts http(s):// to ws(s)://.
func toWSURL(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return strings.Replace(baseURL, "http", "ws", 1)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	return u.String()
}
