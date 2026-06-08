# 界面截图

README「界面预览」引用的 PNG 均放在本目录。文件名固定，覆盖同名文件即可更新预览。

## 清单

| 文件 | 页面 | 路由 | 拍摄要点 |
|------|------|------|----------|
| `home.png` | 开始学习 | `#/` | 领域输入框可见；顶部「模型已连接」 |
| `tree.png` | 课程详情 | `#/tree/:id` | 节点列表含 pending / 进行中 / 已完成；最好有「继续」标签 |
| `courses.png` | 我的课程 | `#/courses` | 至少 2 门课及完成比例 |
| `graph-galaxy.png` | 知识图谱 · 银河 | `#/graph` | 星空主题、多领域节点；可见「银河 \| 目录」切换 |
| `graph-outline.png` | 知识图谱 · 目录 | `#/graph?view=outline` | 领域卡片 + 模块手风琴 + 节点列表 |
| `coach-exercise.png` | AI 教练 | `#/coach/:sessionId` | 一题练习 + 批改反馈；可选「再来一道 / 下一节」 |
| `import.png` | 导入建课 | `#/import` | PDF/URL 上传区与说明文案 |

## 规格建议

- 分辨率：**1280×800** 或 **1440×900**
- 浏览器：Chrome，隐藏书签栏，窗口尽量干净
- 数据：至少 1 门有学习进度的课；图谱建议 2 门以上领域
- 隐私：API Key、邮箱、内部域名请打码

## 自动截取（部分页面）

1. 启动后端：`go run ./cmd/server`
2. 启动前端：`cd web && pnpm dev`
3. 运行：`node scripts/capture-screenshots.mjs`

脚本可生成：`home`、`graph-galaxy`、`graph-outline`、`courses`、`import`。  
`tree`、`coach-exercise` 依赖具体课程/会话 ID，请手动截取。

可选环境变量：

- `SCREENSHOT_BASE` — 默认 `http://localhost:5173`
- `CHROME_PATH` — 默认 macOS Google Chrome 路径

脚本通过 dev 专用的 `?seedProfile=` 跳过角色选择弹窗。

## 迁移说明

旧文件 `graph.png` 已更名为 **`graph-galaxy.png`**。若本地仍有 `graph.png`，可删除或复制为 `graph-galaxy.png`。
