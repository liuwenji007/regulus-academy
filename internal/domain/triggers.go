package domain

import (
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const triggerCategoryExercise = "exercise"
const triggerCategoryBackToExplain = "back_to_explain"
const triggerCategoryNewExercise = "new_exercise"
const triggerCategoryRealWorld = "real_world"
const triggerCategorySkipMastery = "skip_mastery"
const triggerCategoryStartNext = "start_next"

type triggerRule struct {
	Exact           []string `yaml:"exact"`
	Contains        []string `yaml:"contains"`
	ExcludeContains []string `yaml:"exclude_contains"`
}

type triggerConfig struct {
	Exercise       triggerRule `yaml:"exercise"`
	BackToExplain  triggerRule `yaml:"back_to_explain"`
	NewExercise    triggerRule `yaml:"new_exercise"`
	RealWorld      triggerRule `yaml:"real_world"`
	SkipMastery    triggerRule `yaml:"skip_mastery"`
	StartNext      triggerRule `yaml:"start_next"`
}

var (
	triggersOnce sync.Once
	triggers     triggerConfig
	triggersErr  error
)

func loadTriggers() (triggerConfig, error) {
	triggersOnce.Do(func() {
		b, err := ReadCoachFile("triggers.yaml")
		if err != nil {
			triggersErr = err
			return
		}
		triggersErr = yaml.Unmarshal(b, &triggers)
	})
	return triggers, triggersErr
}

// MatchTrigger 按类别匹配用户消息触发词
func MatchTrigger(category, msg string) bool {
	cfg, err := loadTriggers()
	if err != nil {
		return false
	}
	var rule triggerRule
	switch category {
	case triggerCategoryExercise:
		rule = cfg.Exercise
	case triggerCategoryBackToExplain:
		rule = cfg.BackToExplain
	case triggerCategoryNewExercise:
		rule = cfg.NewExercise
	case triggerCategoryRealWorld:
		rule = cfg.RealWorld
	case triggerCategorySkipMastery:
		rule = cfg.SkipMastery
	case triggerCategoryStartNext:
		rule = cfg.StartNext
	default:
		return false
	}
	return matchTriggerRule(rule, msg)
}

func matchTriggerRule(rule triggerRule, msg string) bool {
	m := strings.TrimSpace(msg)
	if m == "" {
		return false
	}
	low := strings.ToLower(m)
	for _, ex := range rule.ExcludeContains {
		if ex != "" && strings.Contains(low, strings.ToLower(ex)) {
			return false
		}
	}
	for _, t := range rule.Exact {
		if t != "" && low == strings.ToLower(strings.TrimSpace(t)) {
			return true
		}
	}
	for _, t := range rule.Contains {
		if t != "" && strings.Contains(low, strings.ToLower(t)) {
			return true
		}
	}
	return false
}
