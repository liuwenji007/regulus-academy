# 界面截图

README 引用的预览图存放在此目录。

## 重新生成

1. 启动后端：`go run ./cmd/server`
2. 启动前端 dev：`cd web && pnpm dev`
3. 运行：`node scripts/capture-screenshots.mjs`

脚本通过 dev 环境专用的 `?seedProfile=` 参数跳过角色选择弹窗（不会进入生产构建）。

可选环境变量：

- `SCREENSHOT_BASE` — 默认 `http://localhost:5173`
- `CHROME_PATH` — 默认 macOS Google Chrome 路径
