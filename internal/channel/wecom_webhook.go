package channel

import (
	"encoding/xml"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/channel/wecom/wxbizmsgcrypt"
	"github.com/regulus-academy/regulus-academy/internal/config"
)

// WeComWebhook 企业微信回调
type WeComWebhook struct {
	cfg    config.WeComConfig
	router *Router
	crypt  *wxbizmsgcrypt.WXBizMsgCrypt
	allow  map[string]struct{}
}

// NewWeComWebhook 创建企微 webhook
func NewWeComWebhook(cfg config.WeComConfig, router *Router) *WeComWebhook {
	allow := make(map[string]struct{})
	for _, id := range cfg.AllowedUsers {
		allow[id] = struct{}{}
	}
	crypt := wxbizmsgcrypt.NewWXBizMsgCrypt(cfg.Token, cfg.EncodingAESKey, cfg.CorpID, wxbizmsgcrypt.XmlType)
	return &WeComWebhook{cfg: cfg, router: router, crypt: crypt, allow: allow}
}

// Verify GET 回调 URL 验证
func (w *WeComWebhook) Verify(rw http.ResponseWriter, r *http.Request) {
	msgSign := r.URL.Query().Get("msg_signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")
	echoStr := r.URL.Query().Get("echostr")

	echo, cryptErr := w.crypt.VerifyURL(msgSign, timestamp, nonce, echoStr)
	if cryptErr != nil {
		http.Error(rw, "verify failed", http.StatusBadRequest)
		return
	}
	_, _ = rw.Write(echo)
}

// HandleMessage POST 消息回调
func (w *WeComWebhook) HandleMessage(rw http.ResponseWriter, r *http.Request) {
	msgSign := r.URL.Query().Get("msg_signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(rw, "read body", http.StatusBadRequest)
		return
	}

	plain, cryptErr := w.crypt.DecryptMsg(msgSign, timestamp, nonce, body)
	if cryptErr != nil {
		http.Error(rw, "decrypt failed", http.StatusBadRequest)
		return
	}

	var msg wecomIncomingMsg
	if err := xml.Unmarshal(plain, &msg); err != nil {
		http.Error(rw, "parse xml", http.StatusBadRequest)
		return
	}

	if msg.MsgType != "text" || strings.TrimSpace(msg.Content) == "" {
		rw.WriteHeader(http.StatusOK)
		return
	}
	if msg.AgentID != "" && w.cfg.AgentID != "" && msg.AgentID != w.cfg.AgentID {
		rw.WriteHeader(http.StatusOK)
		return
	}
	userID := msg.FromUserName
	if len(w.allow) > 0 {
		if _, ok := w.allow[userID]; !ok {
			rw.WriteHeader(http.StatusOK)
			return
		}
	}

	ev := MessageEvent{
		Platform:       PlatformWeCom,
		ChatID:         userID,
		PlatformUserID: userID,
		Text:           strings.TrimSpace(msg.Content),
	}

	result := w.router.Handle(r.Context(), ev)
	parts := append(result.InstantReplies, result.Replies...)
	if len(parts) == 0 {
		rw.WriteHeader(http.StatusOK)
		return
	}

	replyText := strings.Join(parts, "\n\n")
	_ = w.sendPassiveReply(rw, msg, replyText, timestamp, nonce)
}

func (w *WeComWebhook) sendPassiveReply(rw http.ResponseWriter, msg wecomIncomingMsg, text, timestamp, nonce string) error {
	reply := wecomOutgoingMsg{
		ToUserName:   msg.FromUserName,
		FromUserName: msg.ToUserName,
		CreateTime:   time.Now().Unix(),
		MsgType:      "text",
		Content:      text,
	}
	out, err := xml.Marshal(reply)
	if err != nil {
		return err
	}
	if timestamp == "" {
		timestamp = strconv.FormatInt(time.Now().Unix(), 10)
	}
	encrypted, cryptErr := w.crypt.EncryptMsg(string(out), timestamp, nonce)
	if cryptErr != nil {
		http.Error(rw, cryptErr.Error(), http.StatusBadRequest)
		return cryptErr
	}
	rw.Header().Set("Content-Type", "application/xml")
	_, err = rw.Write(encrypted)
	return err
}

type wecomIncomingMsg struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	AgentID      string   `xml:"AgentID"`
}

type wecomOutgoingMsg struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
}
