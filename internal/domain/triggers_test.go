package domain

import (
	"os"
	"path/filepath"
	"testing"
)

func chdirCoachRoot(t *testing.T) {
	t.Helper()
	wd, _ := os.Getwd()
	for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "regulus-coach")); err == nil {
			_ = os.Chdir(d)
			t.Cleanup(func() { _ = os.Chdir(wd) })
			return
		}
	}
	t.Fatal("找不到 regulus-coach 目录")
}

func TestMatchTriggerExercise(t *testing.T) {
	chdirCoachRoot(t)
	if !MatchTrigger(triggerCategoryExercise, "开始练习") {
		t.Fatal("应匹配开始练习")
	}
	if MatchTrigger(triggerCategoryExercise, "开始讲 WaitGroup") {
		t.Fatal("不应匹配开始讲")
	}
	if MatchTrigger(triggerCategoryExercise, "我今天想继续学习别的") {
		t.Fatal("长句不应因继续学习 exact 误触")
	}
	if !MatchTrigger(triggerCategoryExercise, "再来一道") {
		t.Fatal("应匹配再来一道")
	}
}

func TestMatchTriggerSkipMastery(t *testing.T) {
	chdirCoachRoot(t)
	if !MatchTrigger(triggerCategorySkipMastery, "我已经掌握了，下一节") {
		t.Fatal("应匹配已掌握")
	}
	if MatchTrigger(triggerCategorySkipMastery, "还没完全掌握，能再讲讲吗") {
		t.Fatal("否定掌握不应匹配")
	}
}

func TestMatchTriggerStartNext(t *testing.T) {
	chdirCoachRoot(t)
	if !MatchTrigger(triggerCategoryStartNext, "下一节") {
		t.Fatal("应匹配下一节")
	}
}
