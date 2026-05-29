package agent

import "testing"

func TestWantsExercise(t *testing.T) {
	if !wantsExercise("开始练习") {
		t.Fatal("应识别开始练习")
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
