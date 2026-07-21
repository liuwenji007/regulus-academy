# 参与贡献

感谢你愿意花时间改进 Regulus。无论你是开发者还是领域专家，都有适合的贡献方式。

## 你可以做什么

| 方向 | 适合谁 | 入口 |
|------|--------|------|
| **知识节点** | 领域专家、不写代码的协作者 | 编写 `regulus-coach/domains/` 下的节点 YAML |
| **教练与前端** | Go / TypeScript 开发者 | `internal/agent/`、`web/src/` |
| **Prompt** | 熟悉 LLM 教学编排的人 | `regulus-coach/prompts/` |
| **Obsidian / 笔记设计** | 知识库用户、信息架构 | 见下文「知识沉淀 · Obsidian 导出」 |
| **体验反馈** | 所有用户 | GitHub `[体验]` Issue |
| **文档** | 愿意改说明的人 | 本仓库 `apps/docs/` 或 README |

当前特别欢迎：知识图谱体验优化、节点教考对齐、新知识域（Python / Agent / Docker 等）、**Obsidian 学习笔记导出设计**（见下文）。

## 知识沉淀 · Obsidian 导出（欢迎设计贡献）

[导出学习笔记](./learning-notes.md) 的 MVP 已上线：节点点亮 → 异步蒸馏 → 导出 zip → Obsidian 打开。当前实现**刻意从简**——基础 frontmatter、`_MOC.md`、wikilink、蒸馏正文，能跑通闭环，但离「愿意每天打开复习」还有距离。

![Obsidian 学习笔记预览](/screenshots/obsidian.png)

如果你常用 Obsidian、Logseq 或 Markdown 知识库，这里可能是**高性价比**的贡献方向：不一定要写 Go，可以先交设计稿或样例 vault。

### 现状（MVP）

| 已有 | 尚未做 |
|------|--------|
| 节点点亮后 LLM 蒸馏笔记（`phase_note_distill.md`） | 闪卡 / 间隔复习格式 |
| 按课程导出 zip，中文文件名 + wikilink | 多课程合并为一个 vault |
| `_MOC.md` 按掌握深度列表 | Dataview 查询模板、标签体系 |
| frontmatter：`domain` / `layer` / `mastery` / `status` | 跨域笔记自动建链 |
| 无笔记时的占位（关键概念、踩坑） | Agent 维护笔记、RAG 反哺教练 |

代码入口：`internal/domain/export_vault.go`、`internal/agent/note_distill.go`。可以先在 Issue 中提交样例 vault 或设计稿，再决定是否修改代码。

### 欢迎贡献什么

- **笔记模板**：frontmatter 字段、正文结构（复习友好、可扫读）
- **MOC / 索引页**：按模块分组、掌握度视图、待复习清单
- **蒸馏 Prompt**：`regulus-coach/prompts/phase_note_distill.md` 怎样写更像「自己的笔记」
- **导出组装**：`export_vault.go` 的链接策略、文件名、附件结构
- **Obsidian 生态**：兼容 Dataview、Spaced Repetition、Templater 等的约定与示例

### 怎么参与

1. 先 [导出一份](./learning-notes.md) 自己的 vault，在 Obsidian 里试用
2. 开 GitHub Issue，标签建议 `[讨论]`，附上：痛点、理想样例（可贴 `.md` 片段或截图）
3. 若已有具体改动，PR 欢迎；纯设计讨论也欢迎，先对齐再写代码

用户向说明：[导出学习笔记](./learning-notes.md) · 界面截图：[界面预览](./screenshots.md#导出与-obsidian)

## 协作流程

1. Fork 仓库，从 `main` 拉分支
2. 本地按 [本地开发](./development.md) 跑通
3. 提交 PR，说明改动动机；关联 Issue 请写 `Fixes #123`
4. 尊重不同经验背景，围绕问题讨论，不公开他人的敏感信息

项目结构见 [本地开发](./development.md)；节点写法与教练改动要求见 [教学质量](./contributing-teaching.md)。

## 教学质量相关

若你关心「练太多 / 一点就亮」「节点边界模糊」等教学体验，请先读 [贡献 · 教学质量](./contributing-teaching.md)，再开 `[讨论]` 或 `[体验]` Issue 对齐预期。

## 相关

- [本地开发](./development.md)
- [教学模式](./teaching-model.md) — 理解产品教学机制
- [GitHub Issues](https://github.com/liuwenji007/regulus-academy/issues)
