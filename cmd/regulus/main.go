package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/regulus-academy/regulus-academy/internal/cliruntime"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	var code int
	switch cmd {
	case "build":
		code = runBuild(args)
	case "session":
		code = runSession(args)
	case "progress":
		code = runProgress(args)
	case "doctor":
		code = runDoctor(args)
	case "link":
		code = runLink(args)
	case "sync":
		code = runSync(args)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n\n", cmd)
		printUsage()
		code = 1
	}
	os.Exit(code)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Regulus Academy CLI（便携 Skill 运行时）

用法:
  regulus <command> [options]

命令:
  build     建课写入 domains/
  session   教练会话（start | message）
  progress  查看本地进度
  doctor    环境自检
  link      关联已部署的 Regulus Web
  sync      pull | push 与远程同步进度

全局选项（build / session / progress / doctor / sync）:
  --coach-root <dir>   regulus-coach 目录

示例:
  regulus doctor
  regulus build --coach-root ./regulus-coach "想学 TypeScript"
  regulus session start --slug go-concurrency
  regulus session message --session <id> "开始练习"
  regulus link --url http://localhost:8080 --user-id default
  regulus sync pull

环境:
  <coach-root>/.env 或 data/.env 中的 LLM_API_KEY
  本地进度: <coach-root>/data/regulus.db
`)
}

func coachRootFlag(args []string) (string, []string) {
	fs := flag.NewFlagSet("global", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("coach-root", "", "")
	_ = fs.Parse(args)
	return *root, fs.Args()
}

func openRT(args []string) (*cliruntime.Runtime, []string, error) {
	root, rest := coachRootFlag(args)
	rt, err := cliruntime.Open(root)
	if err != nil {
		return nil, rest, err
	}
	return rt, rest, nil
}

func runBuild(args []string) int {
	opt, err := parseBuildArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if opt.topic == "" {
		fmt.Fprintln(os.Stderr, "错误: 请提供学习主题")
		return 1
	}
	rt, err := cliruntime.Open(opt.coachRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer rt.Close()
	if err := rt.BuildDomain(context.Background(), opt.topic, opt.force, opt.profile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runSession(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: regulus session start|message ...")
		return 1
	}
	switch args[0] {
	case "start":
		return sessionStart(args[1:])
	case "message":
		return sessionMessage(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "未知 session 子命令:", args[0])
		return 1
	}
}

func sessionStart(args []string) int {
	fs := flag.NewFlagSet("session start", flag.ExitOnError)
	coachRoot := fs.String("coach-root", "", "")
	slug := fs.String("slug", "", "")
	node := fs.String("node", "", "")
	layer := fs.String("layer", "", "")
	_ = fs.Parse(args)
	if *slug == "" {
		fmt.Fprintln(os.Stderr, "需要 --slug")
		return 1
	}
	rt, err := cliruntime.Open(*coachRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer rt.Close()
	out, err := rt.SessionStart(context.Background(), *slug, *node, *layer)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return printErr(cliruntime.PrintJSON(out))
}

func sessionMessage(args []string) int {
	opt, err := parseSessionMessageArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if opt.sessionID == "" || opt.text == "" {
		fmt.Fprintln(os.Stderr, "需要 --session 和消息正文")
		return 1
	}
	rt, err := cliruntime.Open(opt.coachRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer rt.Close()
	out, err := rt.SessionMessage(context.Background(), opt.sessionID, opt.text)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return printErr(cliruntime.PrintJSON(out))
}

func runProgress(args []string) int {
	fs := flag.NewFlagSet("progress", flag.ExitOnError)
	coachRoot := fs.String("coach-root", "", "")
	slug := fs.String("slug", "", "")
	_ = fs.Parse(args)
	rt, err := cliruntime.Open(*coachRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer rt.Close()
	rows, err := rt.ListProgress(*slug)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return printErr(cliruntime.PrintJSON(map[string]any{"progress": rows}))
}

func runDoctor(args []string) int {
	rt, rest, err := openRT(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer rt.Close()
	_ = rest
	return printErr(cliruntime.PrintJSON(rt.Doctor(context.Background())))
}

func runLink(args []string) int {
	fs := flag.NewFlagSet("link", flag.ExitOnError)
	coachRoot := fs.String("coach-root", "", "")
	url := fs.String("url", "", "")
	userID := fs.String("user-id", "", "")
	_ = fs.Parse(args)
	if *url == "" {
		fmt.Fprintln(os.Stderr, "需要 --url，例如 http://localhost:8080")
		return 1
	}
	rt, err := cliruntime.Open(*coachRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer rt.Close()
	if err := rt.Link(*url, *userID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("已关联远程 Regulus:", *url)
	return 0
}

func runSync(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: regulus sync pull|push")
		return 1
	}
	sub := args[0]
	rt, _, err := openRT(args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer rt.Close()
	ctx := context.Background()
	switch sub {
	case "pull":
		n, err := rt.SyncPull(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("已合并 %d 条远程进度到本地\n", n)
		return 0
	case "push":
		n, err := rt.SyncPush(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("已推送 %d 条进度到远程\n", n)
		return 0
	default:
		fmt.Fprintln(os.Stderr, "未知 sync 子命令:", sub)
		return 1
	}
}

func printErr(err error) int {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
