package agent

import "testing"

func TestWantsExercise(t *testing.T) {
	if !wantsExercise("开始练习") {
		t.Fatal("应识别开始练习")
	}
	if !wantsExercise("再来一道") {
		t.Fatal("应识别再来一道")
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
	if !wantsRealWorldCase("想看看生产环境怎么写") {
		t.Fatal("应识别生产场景")
	}
	if wantsRealWorldCase("开始练习") {
		t.Fatal("不应与开始练习冲突")
	}
}
