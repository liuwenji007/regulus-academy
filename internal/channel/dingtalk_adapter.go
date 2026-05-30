package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/regulus-academy/regulus-academy/internal/config"
)

const (
	dingtalkStreamURL      = "https://api.dingtalk.com/v1.0/gateway/connections/open"
	dingtalkTokenURL       = "https://api.dingtalk.com/v1.0/oauth2/accessToken"
	dingtalkSendMessageURL = "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend"
)

// DingTalkAdapter 钉钉 Stream 模式
type DingTalkAdapter struct {
	cfg      config.DingTalkConfig
	token    string
	tokenExp time.Time
	tokenMu  sync.Mutex
	http     *http.Client
}

// NewDingTalkAdapter 创建钉钉适配器
func NewDingTalkAdapter(cfg config.DingTalkConfig) *DingTalkAdapter {
	return &DingTalkAdapter{
		cfg:  cfg,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *DingTalkAdapter) Name() string { return PlatformDingTalk }

func (s *DingTalkAdapter) Start(ctx context.Context, onMessage func(MessageEvent)) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := s.runOnce(ctx, onMessage); err != nil {
			log.Printf("[dingtalk] 连接断开: %v，5s 后重连", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
			}
		}
	}
}

func (s *DingTalkAdapter) runOnce(ctx context.Context, onMessage func(MessageEvent)) error {
	token, err := s.getToken(ctx)
	if err != nil {
		return err
	}

	reqBody, _ := json.Marshal(map[string]any{
		"clientId":      s.cfg.ClientID,
		"clientSecret":  s.cfg.ClientSecret,
		"subscriptions": []map[string]string{{"type": "EVENT", "topic": "*"}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dingtalkStreamURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-acs-dingtalk-access-token", token)

	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("open connection: %s", string(body))
	}

	var openResp struct {
		Endpoint string `json:"endpoint"`
		Ticket   string `json:"ticket"`
	}
	if err := json.Unmarshal(body, &openResp); err != nil {
		return err
	}
	if openResp.Endpoint == "" {
		return fmt.Errorf("empty stream endpoint")
	}

	wsURL := openResp.Endpoint
	if openResp.Ticket != "" {
		if strings.Contains(wsURL, "?") {
			wsURL += "&ticket=" + openResp.Ticket
		} else {
			wsURL += "?ticket=" + openResp.Ticket
		}
	}

	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Println("[dingtalk] Stream 已连接")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		s.handleFrame(conn, data, onMessage)
	}
}

func (s *DingTalkAdapter) handleFrame(conn *websocket.Conn, data []byte, onMessage func(MessageEvent)) {
	var frame struct {
		Headers map[string]string `json:"headers"`
		Data    json.RawMessage   `json:"data"`
	}
	if err := json.Unmarshal(data, &frame); err != nil {
		return
	}
	topic := frame.Headers["topic"]
	if topic == "" {
		topic = frame.Headers["eventType"]
	}

	switch topic {
	case "SYSTEM", "ping", "KEEP_ALIVE":
		s.ack(conn, frame.Headers)
		return
	case "EVENT", "bot_message_callback":
		s.ack(conn, frame.Headers)
		s.parseEvent(frame.Data, onMessage)
	default:
		if strings.Contains(string(data), "message") {
			s.ack(conn, frame.Headers)
			s.parseEvent(frame.Data, onMessage)
		}
	}
}

func (s *DingTalkAdapter) ack(conn *websocket.Conn, headers map[string]string) {
	msgID := headers["messageId"]
	if msgID == "" {
		return
	}
	ack, _ := json.Marshal(map[string]any{
		"code":    200,
		"headers": map[string]string{"messageId": msgID, "contentType": "application/json"},
	})
	_ = conn.WriteMessage(websocket.TextMessage, ack)
}

func (s *DingTalkAdapter) parseEvent(data json.RawMessage, onMessage func(MessageEvent)) {
	var payload struct {
		Text struct {
			Content string `json:"content"`
		} `json:"text"`
		ConversationType string `json:"conversationType"`
		ConversationID   string `json:"conversationId"`
		SenderStaffID    string `json:"senderStaffId"`
		SenderID         string `json:"senderId"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		var wrap struct {
			Event json.RawMessage `json:"event"`
		}
		if json.Unmarshal(data, &wrap) == nil && len(wrap.Event) > 0 {
			_ = json.Unmarshal(wrap.Event, &payload)
		}
	}
	text := strings.TrimSpace(payload.Text.Content)
	if text == "" {
		return
	}
	if payload.ConversationType != "" && payload.ConversationType != "1" {
		return
	}
	userID := payload.SenderStaffID
	if userID == "" {
		userID = payload.SenderID
	}
	if userID == "" {
		return
	}
	chatID := payload.ConversationID
	if chatID == "" {
		chatID = userID
	}
	onMessage(MessageEvent{
		Platform:       PlatformDingTalk,
		ChatID:         chatID,
		PlatformUserID: userID,
		Text:           text,
	})
}

func (s *DingTalkAdapter) getToken(ctx context.Context) (string, error) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	if s.token != "" && time.Now().Before(s.tokenExp) {
		return s.token, nil
	}
	body, _ := json.Marshal(map[string]string{
		"appKey":    s.cfg.ClientID,
		"appSecret": s.cfg.ClientSecret,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dingtalkTokenURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var out struct {
		AccessToken string `json:"accessToken"`
		ExpireIn    int    `json:"expireIn"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("dingtalk token failed: %s", string(b))
	}
	s.token = out.AccessToken
	s.tokenExp = time.Now().Add(time.Duration(out.ExpireIn-60) * time.Second)
	return s.token, nil
}

func (s *DingTalkAdapter) SendText(ctx context.Context, target ReplyTarget, text string) error {
	token, err := s.getToken(ctx)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{
		"robotCode": s.cfg.ClientID,
		"userIds":   []string{target.PlatformUserID},
		"msgKey":    "sampleText",
		"msgParam":  fmt.Sprintf(`{"content":%q}`, text),
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dingtalkSendMessageURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-acs-dingtalk-access-token", token)
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dingtalk send: %s", string(b))
	}
	return nil
}
