# Changelog

本文件记录 Regulus Academy 的版本变更。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)。

## [Unreleased]

## [0.3.0] - 2026-11-11

相对 `v0.2.2`：课程资料接地、领域分类、知识图谱星座布局。

### 新增

- **课程资料抽屉**：课程页顶栏「资料」可上传 PDF / Markdown / 文本或粘贴正文；资料持久化到 `domain_sources`，建课时按蒸馏大纲切块并标注概念，绑定到节点字段（`must_teach` / `grading_hints` 等）
- **划词助教资料召回**：优先引用当前课已挂资料，文末 `[1]` 短摘；资料变更会使术语卡缓存失效
- **助教可选联网**：配置 `SEARCH_API_KEY`（Brave Search）后，用户主动勾选才可联网；「怎么读」不联网；有资料时仍以资料为准
- **领域分类（Field）**：课程可归入预设领域或自定义领域（最多 8 个）；建课自动归类；我的课程可按领域筛选；重新生成课程时保留用户已设领域
- **星座布局**：知识图谱按课程结构渲染星座；星空主题 spiral 旋臂、宣纸主题 chart 盖天式星图；领域决定旋臂 / 扇区配色与排布

### 修复

- 星空银河光晕、尾迹与尘埃参数，改善可见度与层次

### 工程

- 迁移：`027_domain_sources`、`028_source_chunk_concepts`、`029_domain_field`
- 环境变量：`REGULUS_INGEST_ALLOW_PRIVATE`、`REGULUS_SOURCE_MAX_USER_BYTES`、`SEARCH_API_KEY`
- 领域重绑统一走 `rebindAndInvalidateTermCards`，避免术语卡脏缓存

### 文档

- 划词助教 / 教学模式 / 环境变量 / knowledge-vault：课程资料接地与联网口径
- 知识图谱：星座布局与领域排布说明
- 功能一览：资料抽屉入口

## [0.2.2] - 2026-08-14

补记：相对 `v0.1.0` 已随 cloud 迭代合入、此前记在 Unreleased 的能力。

### 新增

- **行动助手**：多轮对话产出北星 / 清障 / 今日行动；可钉北星、勾选跨会话保留；四象限为展开详情（节奏恢复，非通用待办）
- **学习捷径**：侧栏「上一节」续课 +「今日推荐」（优先行动助手计划，其次知识缺口匹配，否则按进度）
- **划词助教**：教练页划词解术语 / 读音 / 短发散；术语本与知识缺口账本；可绑独立轻量模型；不打断主线进度
- **内置课程目录**：`#/catalog` 浏览 Skill 课并一键开练
- **课程体检与优化**：规则 + 可选 LLM 质检报告；勾选建议后补全节点 YAML（保留 `node_key` 与进度）
- **练习体验**：作答格式校验、题目清洗、批改反馈与上下文管理增强
- **画像**：按课摘要与结构化字段编辑；设置页可查看 / 补充背景与目标

### 修复

- **建树 modules 一致性自动修补**：漏挂节点 / 幽灵引用 / 双归属在校验前确定性修复，避免因此整树重试 LLM
- 导航、错误处理与知识图谱命中检测

### Cloud

- 日配额默认：教练消息 **30**/天、自定义建课 **3**/天（内置 Skill 快路径不计入建课额度）
- 每 IP 每日创建学习角色上限、建课并发与管理鉴权测试补强

### 工程

- Release workflow：同步 coach embed 并创建 `dist` 目录后再打包 CLI

### 文档

- README / 文档站按第一性原理改版：决策层 README + 任务层文档站；口径见 `_copy-ssot.md`
- 规划项命名消歧：「每日推荐」→ **Agent 主动日推**（与已上线「今日推荐」区分）
- 新增文档页：行动助手、学习画像、课程体检、为什么是 Regulus、划词助教
- 截图清单补充划词助教动图槽位（`aside-selection.gif` 等，素材待录）

## [0.1.0] - 2026-06-18

首个公开试用版本，适合自托管、在线 Demo 与小范围邀请测试。

### 新增

- **学习闭环**：讲解 → 练习 → 批改 → 点亮 → 学习笔记蒸馏（Obsidian Vault 导出 MVP）
- **Web 应用**：知识树、知识图谱（宣纸/星空双主题）、教练对话、PDF/URL 导入建课、纵深扩展
- **多入口**：Docker 一键安装、Coach Skill（Linked / Agent-lite / 可选 CLI）、IM 频道（Telegram / 钉钉 / 飞书 / 企微）
- **Cloud Demo**：日配额 + BYOK、共学统计、管理员控制台
- **内置知识域**：Go 并发、Prompt 设计、Python 快速入门、Rust 快速入门（共 46 节点）
- **用户画像**：冷启动问卷、节末异步合并、按背景裁剪讲解
- **多学习角色**：进度与课程列表按角色隔离

### 工程

- Apache 2.0 许可证；`CONTRIBUTING.md`、`SECURITY.md`、`CODE_OF_CONDUCT.md`
- CI：`go test`、前端构建、文档站构建；GHCR 预构建镜像；`v*` tag 触发 CLI Release
- 在线文档：https://regulus-academy-docs.vercel.app

### 已知限制

- Obsidian 导出为 MVP，模板与蒸馏质量欢迎社区贡献
- Cloud Demo 不含 IM；敏感学习内容请自托管
- 部分知识域 `teaching_beats` 仍在对齐中（Go 并发域最成熟）

[Unreleased]: https://github.com/liuwenji007/regulus-academy/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/liuwenji007/regulus-academy/releases/tag/v0.3.0
[0.2.2]: https://github.com/liuwenji007/regulus-academy/releases/tag/v0.2.2
[0.1.0]: https://github.com/liuwenji007/regulus-academy/releases/tag/v0.1.0
