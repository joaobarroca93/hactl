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
