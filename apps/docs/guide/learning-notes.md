# 导出学习笔记

路由：课程详情 `#/tree/:id` · 顶部 **导出学习笔记**

学习闭环的最后一环：讲解 → 练习 → 反馈 → 点亮 → **沉淀**。把 SQLite 里的进度与对话蒸馏成 Markdown，打包为 Obsidian 兼容的 zip，导入本地后就是你的个人知识库。

## 怎么导出

1. 打开某门课的 **课程详情页**（`#/tree/:id`）
2. 点击顶部 **导出学习笔记**
3. 浏览器下载 `{课程名}-vault.zip`
4. 解压到任意目录，用 Obsidian **打开文件夹作为库**即可

与「导出 Skill 包」并列，在同一操作栏。在线 Demo 与自托管均可用。

![课程详情 · 导出按钮](/screenshots/tree.png)

## 笔记什么时候生成

节点**点亮**后，系统会**异步**调用 LLM，把本节教练对话蒸馏成 300～500 字的学习笔记，写入数据库 `node_notes`。

| 时机 | 说明 |
|------|------|
| 触发 | 节点状态变为 `completed` 时自动开始 |
| 耗时 | 通常数十秒；失败不影响点亮 |
| 内容 | 第一人称笔记：核心理解、关键概念、踩过的坑（若有错题） |

刚点亮的节点可能尚未蒸馏完成。此时导出 zip 里该节点会显示**占位内容**（关键概念列表、错题摘要等）；稍后再次导出即可拿到完整笔记。

## zip 里有什么

解压后是一个以**课程名**命名的文件夹，例如 `Go 并发/`：

```
Go 并发/
├── _MOC.md              # 学习地图索引（按掌握深度分层）
├── goroutine_basics.md  # 每个节点一篇笔记
├── channel.md
└── ...
```

### 单篇笔记结构

每篇 `.md` 含 YAML frontmatter，便于 Obsidian 检索与 Dataview：

```yaml
---
domain: "Go 并发"
layer: "熟悉"
node: "channel"
mastery: 0.85
status: "completed"
updated: "2026-06-17"
---
```

正文优先使用蒸馏笔记；若无笔记但已学过，则生成关键概念与「踩过的坑」占位；未开始的节点标记为 `_尚未学习_`。

节点间通过 `[[wikilink]]` 链接（来自知识树的 `requires` 前置关系）。`_MOC.md` 汇总全部节点及掌握度图例（✅ / 🔄 / ⬜）。

### 在 Obsidian 里怎么用

- **Graph View**：wikilink 自动连成局部图谱，类似 Regulus 知识图谱的本地镜像
- **搜索 / 标签**：可按 frontmatter 的 `domain`、`layer`、`status` 过滤
- **复习**：按目录或 MOC 回顾已学节点

## 与「导出 Skill 包」的区别

| | 导出学习笔记 | 导出 Skill 包 |
|---|-------------|---------------|
| 产物 | Markdown zip（Obsidian） | `{slug}-skill.zip` |
| 内容 | **你的**进度、对话蒸馏、错题 | 课程结构与节点 YAML（可给 Agent 练习） |
| 用途 | 个人复习、知识沉淀 | 离线练习、贡献社区 `domains/` |

## 限制（当前版本）

- **只读导出**：Obsidian 里的修改不会回写 Regulus
- **手动触发**：无自动增量同步，学完一批节点后可再次导出覆盖
- **按课程导出**：一次导出一门课；多门课需分别导出
- **蒸馏依赖 LLM**：未配置 API Key 时无法生成新笔记（已有 `node_notes` 仍会导出）

更完整的设计说明见仓库 [docs/knowledge-vault.md](https://github.com/liuwenji007/regulus-academy/blob/main/docs/knowledge-vault.md)。

## 相关

- [功能一览](./features.md)
- [快速上手](./quick-start.md)
- [AI 模型配置](./model-config.md) — 蒸馏与教练共用模型 Key
