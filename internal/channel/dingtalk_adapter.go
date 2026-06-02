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
	dingtalkBotCallback    = "/v1.0/im/bot/messages/get"
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
	SetPlatformConnected(PlatformDingTalk, false)
	for {
		if ctx.Err() != nil {
			SetPlatformConnected(PlatformDingTalk, false)
			return ctx.Err()
		}
		if err := s.runOnce(ctx, onMessage); err != nil {
			SetPlatformConnected(PlatformDingTalk, false)
			RecordPlatformError(PlatformDingTalk, err.Error())
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
	endpoint, ticket, err := s.openConnection(ctx)
	if err != nil {
		return err
	}

	wsURL := endpoint
	if ticket != "" {
		sep := "?"
		if strings.Contains(wsURL, "?") {
			sep = "&"
		}
		wsURL += sep + "ticket=" + ticket
	}

	log.Printf("[dingtalk] 建立 WebSocket: %s", maskTicketURL(wsURL))
	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("WebSocket 握手失败: %w", err)
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	SetPlatformConnected(PlatformDingTalk, true)
	log.Println("[dingtalk] Stream 已连接，可在开放平台点击「验证 Stream 模式通道」")

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
		if disconnect, err := s.handleFrame(conn, data, onMessage); err != nil {
			return err
		} else if disconnect {
			return fmt.Errorf("服务端请求断开连接")
		}
	}
}

func (s *DingTalkAdapter) openConnection(ctx context.Context) (endpoint, ticket string, err error) {
	// 协议要求：EVENT(*)+CALLBACK(机器人消息)；缺 CALLBACK 会导致开放平台 Stream 验证失败
	reqBody, _ := json.Marshal(map[string]any{
		"clientId":     s.cfg.ClientID,
		"clientSecret": s.cfg.ClientSecret,
		"ua":           "regulus-academy/dingtalk-stream/1.0",
		"subscriptions": []map[string]string{
			{"type": "EVENT", "topic": "*"},
			{"type": "CALLBACK", "topic": dingtalkBotCallback},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dingtalkStreamURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return "", "", err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("注册 Stream 凭证失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var openResp struct {
		Endpoint string `json:"endpoint"`
		Ticket   string `json:"ticket"`
		Code     string `json:"code"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(body, &openResp); err != nil {
		return "", "", err
	}
	if openResp.Endpoint == "" {
		return "", "", fmt.Errorf("注册 Stream 凭证失败: %s", string(body))
	}
	return openResp.Endpoint, openResp.Ticket, nil
}

type streamFrame struct {
	SpecVersion string            `json:"specVersion"`
	Type        string            `json:"type"`
	Headers     map[string]string `json:"headers"`
	Data        json.RawMessage   `json:"data"`
}

func (s *DingTalkAdapter) handleFrame(conn *websocket.Conn, raw []byte, onMessage func(MessageEvent)) (disconnect bool, err error) {
	var frame streamFrame
	if err := json.Unmarshal(raw, &frame); err != nil {
		return false, nil
	}
	topic := frame.Headers["topic"]
	msgType := strings.ToUpper(frame.Type)

	switch msgType {
	case "SYSTEM":
		switch topic {
		case "ping", "KEEP_ALIVE":
			s.replyFrame(conn, frame.Headers, 200, "OK", pingAckData(frame.Data))
		case "disconnect":
			log.Printf("[dingtalk] 收到 disconnect: %s", string(frame.Data))
			return true, nil
		default:
			s.replyFrame(conn, frame.Headers, 200, "OK", `{"response":null}`)
		}
	case "CALLBACK":
		s.replyFrame(conn, frame.Headers, 200, "OK", `{"response":null}`)
		if topic == dingtalkBotCallback {
			s.parseBotCallback(frame.Data, onMessage)
		}
	case "EVENT":
		s.replyFrame(conn, frame.Headers, 200, "OK", `{"status":"SUCCESS","message":"success"}`)
	default:
		// 兼容旧格式
		switch topic {
		case "SYSTEM", "ping", "KEEP_ALIVE":
			s.replyFrame(conn, frame.Headers, 200, "OK", pingAckData(frame.Data))
		case dingtalkBotCallback, "bot_message_callback":
			s.replyFrame(conn, frame.Headers, 200, "OK", `{"response":null}`)
			s.parseBotCallback(frame.Data, onMessage)
		case "EVENT", "*":
			s.replyFrame(conn, frame.Headers, 200, "OK", `{"status":"SUCCESS","message":"success"}`)
		}
	}
	return false, nil
}

func pingAckData(data json.RawMessage) string {
	if len(data) == 0 {
		return `{"opaque":""}`
	}
	var encoded string
	if err := json.Unmarshal(data, &encoded); err == nil && encoded != "" && encoded[0] == '{' {
		data = json.RawMessage(encoded)
	}
	var ping struct {
		Opaque string `json:"opaque"`
	}
	_ = json.Unmarshal(data, &ping)
	b, _ := json.Marshal(ping)
	return string(b)
}

func (s *DingTalkAdapter) replyFrame(conn *websocket.Conn, headers map[string]string, code int, message, data string) {
	msgID := headers["messageId"]
	if msgID == "" {
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"code":    code,
		"message": message,
		"headers": map[string]string{
			"messageId":   msgID,
			"contentType": "application/json",
		},
		"data": data,
	})
	if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		log.Printf("[dingtalk] 回复 ACK 失败: %v", err)
	}
}

func (s *DingTalkAdapter) parseBotCallback(data json.RawMessage, onMessage func(MessageEvent)) {
	if len(data) == 0 {
		return
	}
	// data 可能是 JSON 字符串（双重编码）
	var payloadStr string
	if err := json.Unmarshal(data, &payloadStr); err == nil && payloadStr != "" {
		data = json.RawMessage(payloadStr)
	}

	var payload struct {
		Text struct {
			Content string `json:"content"`
		} `json:"text"`
		ConversationType string `json:"conversationType"`
		ConversationID   string `json:"conversationId"`
		SenderStaffID    string `json:"senderStaffId"`
		SenderID         string `json:"senderId"`
		MsgType          string `json:"msgtype"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		log.Printf("[dingtalk] 解析机器人消息失败: %v raw=%s", err, truncate(string(data), 200))
		return
	}
	if payload.MsgType != "" && payload.MsgType != "text" {
		return
	}
	text := strings.TrimSpace(payload.Text.Content)
	if text == "" {
		return
	}
	// conversationType: 1=单聊 2=群聊
	if payload.ConversationType != "" && payload.ConversationType != "1" {
		log.Printf("[dingtalk] 忽略群聊消息（请在单聊窗口与机器人对话）")
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
	RecordPlatformEvent(PlatformDingTalk)
	log.Printf("[dingtalk] 收到单聊消息 user=%s text=%s", userID, truncate(text, 80))
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
		return "", fmt.Errorf("获取 accessToken 失败: %s", string(b))
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

func maskTicketURL(u string) string {
	if i := strings.Index(u, "ticket="); i >= 0 {
		return u[:i+7] + "***"
	}
	return u
}
