# 界面截图

README 与 [在线文档](https://regulus-academy-docs.vercel.app/guide/screenshots) 引用的 PNG 均放在本目录。文件名固定，覆盖同名文件即可更新预览。

## 清单

### 入口与学习路径

| 文件 | 页面 | 路由 | 拍摄要点 |
|------|------|------|----------|
| `home.png` | 开始学习 | `#/` | 领域输入框可见；顶部「模型已连接」 |
| `tree.png` | 课程详情 | `#/tree/:id` | 节点列表含 pending / 进行中 / 已完成；顶部露出「导出 Skill 包」「导出学习笔记」 |
| `tree-extend.png` | 课程详情 · 纵深扩展 | `#/tree/:id` | 完成度 ≥80%，「解锁进阶路径」按钮可见 |
| `courses.png` | 我的课程 | `#/courses` | 至少 2 门课及完成比例 |

### 知识图谱

| 文件 | 页面 | 路由 | 拍摄要点 |
|------|------|------|----------|
| `graph-paper.png` | 知识图谱 · 图谱（宣纸） | `#/graph` | **默认主题**；多领域节点；可见「图谱 \| 目录」与主题切换按钮（显示「星空」表示当前为宣纸） |
| `graph-sky.png` | 知识图谱 · 图谱（星空） | `#/graph` | 点击顶栏切到星空后拍摄；星云光晕、进度星光可见 |
| `graph-outline.png` | 知识图谱 · 目录 | `#/graph?view=outline` | 领域卡片 + 模块手风琴 + 节点列表（顶栏固定宣纸色） |

> **需手动拍摄**：`graph-sky.png` 需在浏览器中切换到星空主题后截取（见下方「知识图谱主题」）。`graph-paper.png` 可由自动脚本生成（默认宣纸），也可手动重拍以更清晰展示主题按钮。

### 教练与建课

| 文件 | 页面 | 路由 | 拍摄要点 |
|------|------|------|----------|
| `coach-exercise.png` | AI 教练 | `#/coach/:sessionId` | 一题练习 + 批改反馈；可选「再来一道 / 下一节」 |
| `import.png` | 导入建课 | `#/import` | PDF/URL 上传区与说明文案 |

### Cloud 演示（`SCREENSHOT_MODE=cloud`）

| 文件 | 页面 | 路由 | 拍摄要点 |
|------|------|------|----------|
| `cloud-home.png` | 开始学习 | `#/`（`seedProfile`） | 页脚 Cloud 条、共学/额度信息 |
| `cloud-profile.png` | 角色选择 | `#/`（**无** seed） | 创建/选择学习角色弹窗 |
| `cloud-settings.png` | 设置 | `#/settings` | 「在线演示模式」横幅 + IM 频道禁用态 |

## 知识图谱主题（手动拍摄）

图谱页 `#/graph` 顶栏有主题快捷按钮：当前为宣纸时按钮文案为「星空」，点击后切到星空（反之亦然）。偏好写入 `localStorage` 键 `regulus:graphCanvasTheme`。

**推荐步骤（Chrome 1280×800）：**

1. 打开 `http://localhost:5173/?seedProfile=...#/graph`（或清空该 localStorage 键，默认即为宣纸）
2. 确认画布为宣纸风格 → 保存为 **`graph-paper.png`**
3. 点击顶栏「星空」→ 确认星云/星光效果 → 保存为 **`graph-sky.png`**
4. 切换到「目录」标签或访问 `#/graph?view=outline` → 保存为 **`graph-outline.png`**（若需更新）

有父子课程时，图谱视图可额外拍到父 → 子有向边（可选，非必须）。

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

自动脚本默认输出 `graph-paper.png`（宣纸主题）。`graph-sky.png` 需按下方步骤手动拍摄。

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
| `graph-galaxy.png` | `graph-paper.png`（默认）/ `graph-sky.png`（星空） | 产品改称「知识图谱」后文件名与主题对齐 |
| — | 删除 `graph-galaxy.png` | 新截图就位后可删 |
