# 界面预览

以下为 Regulus Web 端主要界面。截图源文件位于仓库 [`docs/screenshots/`](https://github.com/liuwenji007/regulus-academy/tree/main/docs/screenshots)，构建文档站时自动同步。

## 入口与学习路径

| 开始学习 | 课程详情 | 我的课程 |
|:---:|:---:|:---:|
| <img src="/screenshots/home.png" width="280" alt="开始学习页" /> | <img src="/screenshots/tree.png" width="280" alt="课程详情" /> | <img src="/screenshots/courses.png" width="280" alt="我的课程" /> |

- **课程详情** `tree.png`：节点列表含多种状态；顶部应可见「导出 Skill 包」「导出学习笔记」
- 更新截图：`node scripts/capture-screenshots.mjs`（见仓库 `docs/screenshots/README.md`）

## 进阶与导出

| | 纵深扩展（≥80% 完成度） | |
|:---:|:---:|:---:|
| | <img src="/screenshots/tree-extend.png" width="280" alt="纵深扩展" /> | |

完成度达标后，课程详情页出现「解锁进阶路径」；与 Skill / Vault 导出按钮同页展示（见上一节 `tree.png`）。

## 知识图谱

| 银河视图 | 目录视图 |
|:---:|:---:|
| <img src="/screenshots/graph-galaxy.png" width="280" alt="知识图谱·银河" /> | <img src="/screenshots/graph-outline.png" width="280" alt="知识图谱·目录" /> |

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
