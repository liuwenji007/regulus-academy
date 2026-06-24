package main

import (
	"fmt"
	"strings"
)

type buildOpts struct {
	coachRoot string
	force     bool
	profile   string
	topic     string
}

// parseBuildArgs 解析 build 参数；flag 与主题顺序无关。
func parseBuildArgs(args []string) (buildOpts, error) {
	var opt buildOpts
	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--coach-root":
			if i+1 >= len(args) {
				return opt, fmt.Errorf("--coach-root 需要目录参数")
			}
			opt.coachRoot = args[i+1]
			i++
		case "--force":
			opt.force = true
		case "--profile":
			if i+1 >= len(args) {
				return opt, fmt.Errorf("--profile 需要参数")
			}
			opt.profile = args[i+1]
			i++
		default:
			if strings.HasPrefix(a, "-") {
				return opt, fmt.Errorf("未知选项: %s", a)
			}
			positional = append(positional, a)
		}
	}
	opt.topic = strings.TrimSpace(strings.Join(positional, " "))
	return opt, nil
}

type sessionMessageOpts struct {
	coachRoot string
	sessionID string
	text      string
}

// parseSessionMessageArgs 解析 session message；flag 与正文顺序无关。
func parseSessionMessageArgs(args []string) (sessionMessageOpts, error) {
	var opt sessionMessageOpts
	var parts []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--coach-root":
			if i+1 >= len(args) {
				return opt, fmt.Errorf("--coach-root 需要目录参数")
			}
			opt.coachRoot = args[i+1]
			i++
		case "--session":
			if i+1 >= len(args) {
				return opt, fmt.Errorf("--session 需要会话 ID")
			}
			opt.sessionID = args[i+1]
			i++
		default:
			if strings.HasPrefix(a, "-") {
				return opt, fmt.Errorf("未知选项: %s", a)
			}
			parts = append(parts, a)
		}
	}
	opt.text = strings.TrimSpace(strings.Join(parts, " "))
	return opt, nil
}
