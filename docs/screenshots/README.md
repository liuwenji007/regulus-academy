# 界面截图

README 与 [在线文档](https://regulus-academy-docs.vercel.app/guide/screenshots) 引用的 PNG 均放在本目录。文件名固定，覆盖同名文件即可更新预览。

## 清单

### 入口与学习路径

| 文件 | 页面 | 路由 | 拍摄要点 |
|------|------|------|----------|
| `home.png` | 开始学习 | `#/` | 领域输入框可见；右上角「Skill 下载」 |
| `tree.png` | 课程详情 | `#/tree/:id` | 节点列表含 pending / 进行中 / 已完成；顶部露出「导出 Domain 包」「导出学习笔记」 |
| `tree-extend.png` | 课程详情 · 纵深扩展 | `#/tree/:id` | 完成度 ≥80%，「解锁进阶路径」按钮可见 |
| `courses.png` | 我的课程 | `#/courses` | 至少 2 门课及完成比例 |
| `catalog.png` | 内置课目录 | `#/catalog` | **待补**：课程卡片列表与一键开练 |
| `assistant.png` | 行动助手 | `#/assistant` | **待补**：北星 / 今日学习 / 清障可见 |

### 知识图谱

| 文件 | 页面 | 路由 | 拍摄要点 |
|------|------|------|----------|
| `graph-paper.png` | 知识图谱 · 图谱（宣纸） | `#/graph` | **默认主题**；多领域节点；可见「图谱 \| 目录」与主题切换按钮（显示「星空」表示当前为宣纸） |
| `graph-sky.png` | 知识图谱 · 图谱（星空） | `#/graph` | 顶栏切到星空后拍摄；星云光晕、进度星光可见 |
| `graph-outline.png` | 知识图谱 · 目录 | `#/graph?view=outline` | 领域卡片 + 模块手风琴 + 节点列表（顶栏固定宣纸色） |

`graph-paper.png` 可由自动脚本生成；`graph-sky.png` 需手动切换主题后截取（脚本无法写入星空偏好）。

### 教练与建课

| 文件 | 页面 | 路由 | 拍摄要点 |
|------|------|------|----------|
| `coach-exercise.png` | AI 教练 | `#/coach/:sessionId` | 一题练习 + 批改反馈；可选「再来一道 / 下一节」 |
| `import.png` | 导入建课 | `#/import` | PDF/URL 上传区与说明文案 |

### 划词助教（截图与动图）

划词是**交互过程**，静态图不如动图直观。建议优先补 GIF，再补面板静帧。

| 文件 | 类型 | 路由 / 场景 | 拍摄要点 |
|------|------|-------------|----------|
| `aside-selection.gif` | **动图（优先）** | `#/coach/:sessionId` | 选中助手气泡中的术语 → 浮层三按钮 → 右侧面板展开术语卡；循环 3～6 秒、≤2MB |
| `aside-panel.png` | 静帧 | 同上，面板已打开 | 术语卡字段可见（定义 / 读音 / 类比）； ideally 露出「术语本」「知识缺口」Tab |
| `aside-gaps.png` | 静帧（可选） | 助教「知识缺口」Tab | 至少 1～2 条未关闭缺口 +「已懂」 |

**录制建议（macOS）：**

1. 本机打开教练页，确保助手已输出含英文术语的讲解
2. 用 QuickTime「屏幕录制」或 [Kap](https://getkap.co/) 框选对话区 + 右侧面板
3. 导出 GIF：Kap 直接出 GIF；或 `ffmpeg -i rec.mov -vf "fps=10,scale=640:-1" -loop 0 aside-selection.gif`
4. 覆盖写入本目录同名文件；文档站经 `apps/docs/public/screenshots` 引用（与本目录同步即可）

**不要用 AI 生成假界面动图**——和产品 UI 不一致会误导用户。素材未就绪前，README / 文档用文字说明入口即可，避免挂坏图。

### 导出与 Obsidian

| 文件 | 页面 | 路由 | 拍摄要点 |
|------|------|------|----------|
| `export.png` | 课程详情 · 导出 | `#/tree/:id` | 顶部「导出 Domain 包」「导出学习笔记」按钮清晰可见 |
| `obsidian.png` | Obsidian 学习笔记 | 本地 Obsidian | 导入 vault 后：笔记正文、Graph View 或文件列表；建议 1280×800 |

`export.png` 可用脚本在设 `DOMAIN_ID` 时与 `tree.png` 同页截取；`obsidian.png` 需本地导出 zip 后手动拍摄。

### Cloud 演示（`SCREENSHOT_MODE=cloud`）

| 文件 | 页面 | 路由 | 拍摄要点 |
|------|------|------|----------|
| `cloud-home.png` | 开始学习 | `#/`（`seedProfile`） | 页脚 Cloud 条、共学/额度信息 |
| `cloud-profile.png` | 角色选择 | `#/`（**无** seed） | 创建/选择学习角色弹窗 |
| `cloud-settings.png` | 设置 | `#/settings` | 「在线演示模式」横幅 + IM 频道禁用态 |

## 知识图谱主题（更新截图时）

图谱页 `#/graph` 顶栏主题按钮显示**将要切换到的主题**（宣纸时按钮为「星空」）。偏好键：`regulus:graphCanvasTheme`。

**手动更新星空图（Chrome 1280×800）：**

1. 打开 `http://localhost:5173/?seedProfile=...#/graph`
2. 点击顶栏「星空」，确认星云/星光效果
3. 覆盖保存 `graph-sky.png`

宣纸图可运行 `node scripts/capture-screenshots.mjs` 自动生成 `graph-paper.png`；目录图为 `graph-outline.png`（`#/graph?view=outline`）。

## 规格建议

- 分辨率：**1280×800**（仓库内 PNG 已统一此尺寸）
- README / 文档站展示：表格内 `<img width="280">`，与列数无关、视觉大小一致
- 浏览器：Chrome，隐藏书签栏，窗口尽量干净
- 数据：至少 1 门有学习进度的课；图谱建议 2 门以上领域；纵深扩展需完成度 ≥80%
- 隐私：API Key、邮箱、内部域名请打码

## 自动截取

1. 启动后端：`go run ./cmd/server`（或 `pnpm dev`）
2. 启动前端：`cd web && pnpm dev`（若未用 `pnpm dev`）
3. 运行：

```bash
# 基础页面（home / graph / courses / import）
node scripts/capture-screenshots.mjs

# 含课程详情与教练（需已有数据）
DOMAIN_ID=<uuid> SESSION_ID=<uuid> node scripts/capture-screenshots.mjs

# Cloud 专属三图（后端须 REGULUS_DEPLOYMENT=cloud）
SCREENSHOT_MODE=cloud node scripts/capture-screenshots.mjs
```

脚本通过 dev 专用的 `?seedProfile=` 跳过角色选择弹窗；`cloud-profile.png` 故意不使用 seed。

自动脚本输出 `graph-paper.png`（宣纸，默认主题）。`graph-sky.png` 仅在手改星空主题后手动覆盖。

### 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `SCREENSHOT_BASE` | `http://localhost:5173` | 前端地址 |
| `SCREENSHOT_MODE` | `default` | `cloud` 时追加 Cloud 三图 |
| `SCREENSHOT_ONLY` | `all` | `default` / `cloud` 仅截取子集 |
| `DOMAIN_ID` | — | 课程 UUID，生成 `tree.png` / `tree-extend.png` |
| `SESSION_ID` | — | 教练会话 UUID，生成 `coach-exercise.png` |
| `CHROME_PATH` | macOS Google Chrome | Headless 截图浏览器 |

**注意**：`seedProfile` 仅在 Vite dev（`localhost:5173`）生效；生产构建请用 `SCREENSHOT_BASE` 指向 dev 服务。`onboardedAt` 需写入 seed JSON 以跳过冷启动问卷。

### Cloud 模式 `.env` 片段

```bash
REGULUS_DEPLOYMENT=cloud
ADMIN_TOKEN=<openssl rand -hex 32>
REGULUS_CLOUD_ENCRYPTION_KEY=<openssl rand -hex 32>
```

## 迁移说明

| 旧文件 | 新文件 | 说明 |
|--------|--------|------|
| `graph.png` | `graph-paper.png` 或 `graph-sky.png` | 按主题拆分 |
| `graph-galaxy.png` | `graph-paper.png` / `graph-sky.png` | 已迁移；`graph-galaxy.png` 可删除 |
