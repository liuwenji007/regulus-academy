package agent

import "testing"

func TestWantsExercise(t *testing.T) {
	if !wantsExercise("开始练习") {
		t.Fatal("应识别开始练习")
	}
	if !wantsExercise("再来一道") {
		t.Fatal("应识别再来一道")
	}
	if !wantsExercise("继续学习") {
		t.Fatal("应识别继续学习")
	}
	if wantsExercise("开始讲 WaitGroup") {
		t.Fatal("不应因「开始」误触发")
	}
	if wantsExercise("什么是 channel") {
		t.Fatal("普通提问不应触发练习")
	}
}

func TestWantsBackToExplain(t *testing.T) {
	if !wantsBackToExplain("不懂") {
		t.Fatal("应识别不懂")
	}
}

func TestWantsNewExercise(t *testing.T) {
	if !wantsNewExercise("换一题") {
		t.Fatal("应识别换题")
	}
}

func TestWantsRealWorldCase(t *testing.T) {
	if !wantsRealWorldCase("实际案例") {
		t.Fatal("应识别实际案例")
	}
}

func TestWantsSkipMastery(t *testing.T) {
	for _, msg := range []string{"我已经掌握了，下一节", "已经掌握", "我已经掌握了"} {
		if !wantsSkipMastery(msg) {
			t.Fatalf("应识别已掌握申请: %q", msg)
		}
	}
	if wantsSkipMastery("怎么掌握 channel") {
		t.Fatal("普通提问不应触发")
	}
	if wantsSkipMastery("还没完全掌握，能再讲讲吗") {
		t.Fatal("否定掌握不应触发跳过")
	}
}

func TestWantsStartNext(t *testing.T) {
	if !wantsStartNext("下一节") {
		t.Fatal("应识别下一节")
	}
	if wantsStartNext("已经掌握") {
		t.Fatal("仅掌握不应触发下一节")
	}
}
