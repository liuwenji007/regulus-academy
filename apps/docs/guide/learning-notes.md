# 导出学习笔记

入口：课程详情页顶部 **导出学习笔记**

学习闭环的最后一环：讲解 → 练习 → 反馈 → 点亮 → **沉淀**。把学习进度与对话蒸馏成 Markdown，打包为 Obsidian 兼容的 zip，导入本地后就是你的个人知识库。

## 怎么导出

1. 打开某门课的 **课程详情页**
2. 点击顶部 **导出学习笔记**
3. 浏览器下载 `{课程名}-vault.zip`
4. 解压到任意目录，用 Obsidian **打开文件夹作为库**即可

与「导出 Domain 包」并列，在同一操作栏。Coach Skill 基础包在主页下载。在线 Demo 与自托管均可用。

![课程详情 · 导出按钮](/screenshots/export.png)

![Obsidian · 学习笔记](/screenshots/obsidian.png)

<div class="docs-callout">

<strong>MVP 说明</strong>：当前导出偏基础（frontmatter + MOC + 蒸馏正文）。若你有 Obsidian 笔记结构、Dataview 或复习工作流方面的想法，欢迎到 <a href="./contributing.md#知识沉淀-obsidian-导出欢迎设计贡献">参与贡献 · Obsidian 导出</a> 一起讨论或提 PR。

</div>

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
├── _MOC.md                    # 学习地图（按模块 + 掌握进度）
├── goroutine 是什么.md        # 中文文件名，wikilink 可直达
├── channel 通信.md
└── ...
```

笔记文件以**节点中文标题**命名（冲突时加 `node_key` 后缀）；frontmatter 含 `module`、`layer`、`mastery` 等字段。

### 单篇笔记结构

每篇 `.md` 含 YAML frontmatter，便于 Obsidian 检索与 Dataview：

```yaml
---
domain: "Go 并发"
module: "Channel 与通信"
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

## 与「导出 Domain 包」的区别

| | 导出学习笔记 | 导出 Domain 包 |
|---|-------------|---------------|
| 产物 | Markdown zip（Obsidian） | `{slug}-domain.zip` |
| 内容 | **你的**进度、对话蒸馏、错题 | 课程结构与节点 YAML（给 Agent 练习） |
| 用途 | 个人复习、知识沉淀 | 叠加到 Coach Skill 的 `domains/`，或贡献社区 |

## 限制（当前版本）

- **只读导出**：Obsidian 里的修改不会回写 Regulus
- **手动触发**：无自动增量同步，学完一批节点后可再次导出覆盖
- **按课程导出**：一次导出一门课；多门课需分别导出
- **蒸馏依赖 LLM**：未配置 API Key 时无法生成新笔记（已有 `node_notes` 仍会导出）

想改进模板、链接方式或导出体验，见 [参与贡献 · Obsidian 导出](./contributing.md#知识沉淀-obsidian-导出欢迎设计贡献)。

## 相关

- [功能一览](./features.md)
- [快速上手](./quick-start.md)
- [参与贡献](./contributing.md) — 含 Obsidian 导出设计呼吁
- [AI 模型配置](./model-config.md) — 蒸馏与教练共用模型 Key
