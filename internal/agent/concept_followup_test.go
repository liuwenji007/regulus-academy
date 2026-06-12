package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestMatchConceptInMessage_longestWins(t *testing.T) {
	core := []string{
		"goroutine 是 Go 的轻量级并发执行单元",
		"与操作系统线程的区别：更小的栈、由 Go runtime 调度",
	}
	got := MatchConceptInMessage("操作系统线程的栈和调度是怎么回事", core)
	if !strings.Contains(got, "线程") {
		t.Fatalf("want thread concept, got %q", got)
	}
	if MatchConceptInMessage("随便聊聊", core) != "" {
		t.Fatal("无关消息不应匹配")
	}
}

func TestFollowUpDeepenOnSecondAsk(t *testing.T) {
	coach, store, sess := setupCoach(t, "这是普通答疑", "这是递进深讲内容")

	msg := "请讲讲 goroutine 轻量级并发执行单元"
	result, err := coach.HandleMessage(context.Background(), sess, msg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Content, "展开讲一下") {
		t.Fatal("首次追问不应深讲")
	}

	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err = coach.HandleMessage(context.Background(), reloaded, "goroutine 轻量级并发执行单元我还是没懂")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Content, "展开讲一下") {
		t.Fatalf("第二次追问应深讲: %q", result.Content)
	}
	final, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	finalCtx := storage.ParseSessionContext(final)
	if len(finalCtx.DeepenedConcepts) == 0 {
		t.Fatal("应记录已深讲概念")
	}
}

func TestDeepenConceptReceivesUserMsg(t *testing.T) {
	coach, store, sess, rec := setupCoachRecording(t, "这是普通答疑", "这是递进深讲内容")

	_, err := coach.HandleMessage(context.Background(), sess, "请讲讲 goroutine 轻量级并发执行单元")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	followUp := "goroutine 轻量级并发执行单元我还是没懂"
	_, err = coach.HandleMessage(context.Background(), reloaded, followUp)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, msg := range rec.lastMessages {
		if strings.Contains(msg.Content, followUp) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("深讲应携带用户追问，messages=%+v", rec.lastMessages)
	}
}
