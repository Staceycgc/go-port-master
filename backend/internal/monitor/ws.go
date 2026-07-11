package monitor

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsReadTimeout  = 2 * time.Minute
	wsWriteTimeout = 5 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin: checkSameOrigin,
}

func checkSameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	originURL, err := url.Parse(origin)
	if err != nil || originURL.Scheme == "" || originURL.Host == "" {
		return false
	}
	if !isAllowedOriginScheme(originURL.Scheme) {
		return false
	}
	requestOrigin := requestCanonicalOrigin(r)
	if requestOrigin == "" {
		return false
	}
	return canonicalOrigin(originURL.Scheme, originURL.Host) == requestOrigin
}

func isAllowedOriginScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}

func canonicalOrigin(scheme, host string) string {
	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		hostname = host
		port = defaultPortForScheme(scheme)
	}
	return strings.ToLower(scheme) + "://" + strings.ToLower(hostname) + ":" + port
}

func defaultPortForScheme(scheme string) string {
	if strings.EqualFold(scheme, "https") {
		return "443"
	}
	return "80"
}

func requestCanonicalOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if r.Host == "" {
		return ""
	}
	return canonicalOrigin(scheme, r.Host)
}

type wsSession struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *wsSession) sendJSON(payload interface{}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return false
	}
	_ = s.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
	if err := s.conn.WriteJSON(payload); err != nil {
		return false
	}
	return true
}

func (s *wsSession) sendText(text string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return false
	}
	_ = s.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
	if err := s.conn.WriteMessage(websocket.TextMessage, []byte(text)); err != nil {
		return false
	}
	return true
}

func (s *wsSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
}

func (s *wsSession) extendReadDeadline() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		_ = s.conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	}
}

type WSHandler struct {
	hub *Hub
}

func NewWSHandler(hub *Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	session := &wsSession{conn: conn}
	h.hub.Add(session)
	defer func() {
		h.hub.Remove(session)
		session.close()
	}()

	session.extendReadDeadline()
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if messageType == websocket.TextMessage {
			text := strings.TrimSpace(string(payload))
			if text == "ping" {
				session.extendReadDeadline()
				if !session.sendText("pong") {
					return
				}
				continue
			}
			var envelope struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(payload, &envelope) == nil && envelope.Type == "ping" {
				session.extendReadDeadline()
				if !session.sendText("pong") {
					return
				}
			}
		}
	}
}
