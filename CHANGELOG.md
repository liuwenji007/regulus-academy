# Changelog

本文件记录 Regulus Academy 的版本变更。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)。

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

[0.1.0]: https://github.com/liuwenji007/regulus-academy/releases/tag/v0.1.0
