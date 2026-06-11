# 界面预览

以下为 Regulus Web 端主要界面。截图源文件位于仓库 [`docs/screenshots/`](https://github.com/liuwenji007/regulus-academy/tree/main/docs/screenshots)，构建文档站时自动同步。

## 入口与学习路径

| 开始学习 | 课程详情 | 我的课程 |
|:---:|:---:|:---:|
| ![开始学习页](/screenshots/home.png) | ![课程详情](/screenshots/tree.png) | ![我的课程](/screenshots/courses.png) |

- **课程详情** `tree.png`：节点列表含多种状态；顶部应可见「导出 Skill 包」「导出学习笔记」
- 更新截图：`node scripts/capture-screenshots.mjs`（见仓库 `docs/screenshots/README.md`）

## 进阶与导出

| 纵深扩展（≥80% 完成度） |
|:---:|
| ![纵深扩展](/screenshots/tree-extend.png) |

完成度达标后，课程详情页出现「解锁进阶路径」；与 Skill / Vault 导出按钮同页展示（见上一节 `tree.png`）。

## 知识图谱

| 银河视图 | 目录视图 |
|:---:|:---:|
| ![知识图谱·银河](/screenshots/graph-galaxy.png) | ![知识图谱·目录](/screenshots/graph-outline.png) |

## 教练闭环与建课

| AI 教练 · 练习反馈 | PDF/URL 导入建课 |
|:---:|:---:|
| ![AI 教练](/screenshots/coach-exercise.png) | ![导入建课](/screenshots/import.png) |

## 在线体验版（Cloud）

<div class="docs-callout">

Cloud 演示截图需在本地设置 <code>REGULUS_DEPLOYMENT=cloud</code> 后运行 <code>SCREENSHOT_MODE=cloud node scripts/capture-screenshots.mjs</code>。

</div>

| 首页 | 角色创建 | 设置 |
|:---:|:---:|:---:|
| ![Cloud 首页](/screenshots/cloud-home.png) | ![角色选择](/screenshots/cloud-profile.png) | ![设置页](/screenshots/cloud-settings.png) |

`cloud-profile.png` 取自在线 Demo；`cloud-home` / `cloud-settings` 在 Vite dev + 自托管 API 下截取（布局与 Cloud 一致；演示模式横幅需 `REGULUS_DEPLOYMENT=cloud` 后端）。

[了解在线体验版限制](./cloud-demo.md)
