# 界面预览

以下为 Regulus Web 端主要界面。截图源文件位于仓库 [`docs/screenshots/`](https://github.com/liuwenji007/regulus-academy/tree/main/docs/screenshots)，构建文档站时自动同步。

## 入口与学习路径

| 开始学习 | 课程详情 | 我的课程 |
|:---:|:---:|:---:|
| <img src="/screenshots/home.png" width="280" alt="开始学习页" /> | <img src="/screenshots/tree.png" width="280" alt="课程详情" /> | <img src="/screenshots/courses.png" width="280" alt="我的课程" /> |

- **课程详情** `tree.png`：节点列表含多种状态；顶部应可见「导出 Domain 包」「导出学习笔记」
- 更新截图：`node scripts/capture-screenshots.mjs`（见仓库 `docs/screenshots/README.md`）

## 进阶与导出

| 导出按钮 | 纵深扩展（≥80% 完成度） |
|:---:|:---:|
| <img src="/screenshots/export.png" width="280" alt="导出 Domain 包与学习笔记" /> | <img src="/screenshots/tree-extend.png" width="280" alt="纵深扩展" /> |

完成度达标后出现「解锁进阶路径」；导出按钮与纵深扩展同页（亦见 `tree.png`）。

## 导出学习笔记 · Obsidian

| 课程详情 · 导出 | Obsidian 中的笔记 |
|:---:|:---:|
| <img src="/screenshots/export.png" width="280" alt="导出按钮" /> | <img src="/screenshots/obsidian.png" width="280" alt="Obsidian 学习笔记" /> |

使用说明见 [导出学习笔记](./learning-notes.md)。欢迎改进模板与导出设计，见 [参与贡献 · Obsidian 导出](./contributing.md#知识沉淀-obsidian-导出欢迎设计贡献)。

## 知识图谱

| 图谱 · 宣纸（默认） | 图谱 · 星空 | 目录 |
|:---:|:---:|:---:|
| <img src="/screenshots/graph-paper.png" width="280" alt="知识图谱·宣纸" /> | <img src="/screenshots/graph-sky.png" width="280" alt="知识图谱·星空" /> | <img src="/screenshots/graph-outline.png" width="280" alt="知识图谱·目录" /> |

顶栏可切换 **图谱 / 目录** 视图与 **宣纸 / 星空** 主题（默认宣纸；偏好存于浏览器）。拍摄要点见仓库 [`docs/screenshots/README.md`](https://github.com/liuwenji007/regulus-academy/blob/main/docs/screenshots/README.md)。

## 教练闭环与建课

| AI 教练 · 练习反馈 | PDF/URL 导入建课 |
|:---:|:---:|
| <img src="/screenshots/coach-exercise.png" width="280" alt="AI 教练" /> | <img src="/screenshots/import.png" width="280" alt="导入建课" /> |

## 在线体验版（Cloud）

<div class="docs-callout">

Cloud 演示截图需在本地设置 <code>REGULUS_DEPLOYMENT=cloud</code> 后运行 <code>SCREENSHOT_MODE=cloud node scripts/capture-screenshots.mjs</code>。

</div>

| 首页 | 角色创建 | 设置 |
|:---:|:---:|:---:|
| <img src="/screenshots/cloud-home.png" width="280" alt="Cloud 首页" /> | <img src="/screenshots/cloud-profile.png" width="280" alt="角色选择" /> | <img src="/screenshots/cloud-settings.png" width="280" alt="设置页" /> |

`cloud-profile.png` 取自在线 Demo；`cloud-home` / `cloud-settings` 在 Vite dev + 自托管 API 下截取（布局与 Cloud 一致；演示模式横幅需 `REGULUS_DEPLOYMENT=cloud` 后端）。

[了解在线体验版限制](./cloud-demo.md)
