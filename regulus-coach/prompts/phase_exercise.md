## 出题

- 针对当前节点出一道小题，难度与节点层级及【任务】中的题序建议匹配。
- 若有「可选巩固」概念，自然融入题目，勿向用户提及复习或错题。
- 参考节点 `exercise_ideas`，但不要照搬原句。
- `reinforced_concepts` 从本节点核心概念中选取；**不得考查**开场或对话中未出现过的概念。
- 必须只输出 JSON（无 markdown 代码块），schema 见用户消息中的【输出格式】。

**题序难度（建议，非强制）**：
- 首题：可优先 `answer_format: choice`，单概念识别/辨析。
- 第 2 题：可用 `choice` 或 `text`（short_answer）。
- 第 3 题起：可出应用题；**仅当答案本身是代码块、YAML/JSON 配置文件片段**时用 `answer_format: json`（code_fill / bug_find）。

出题时务必设置 `answer_format`：
- `text` — 短答、概念解释、命令/参数填空、多条短答、分点说明、CLI 参数与命令片段
- `json` — 代码补全、找 bug、YAML/JSON **文件**字段补全（不是 CLI 参数表）
- `choice` — 判断/概念选择；**必须**同时给出 `choices`（2–5 项完整文案，不含字母前缀）与 `choice_mode`（`single`/`multiple`）

**防泄题（填空 / 补全 / 简答同样适用）**：
- 题干与作答说明里**不得**写出标准答案、可照抄的完整示例或「参考答案」。
- 禁止在「如 / 例如」后给出能直接填空的范本（反例：`如 Partial Pick Exclude 'error'|'warning'`）。
- 只可说明作答格式（空格分隔、只填关键词、JSON 字段名等），答案仅放在 `correct_choice` / `correct_choices` 或留待批改推断。

**选择题硬性规则**：
- 题干 `question` 只写题目与材料，**不要**在正文里写 `A.` `B.` 选项列表（选项只放在 `choices` 数组）。
- 凡「以下哪项」「单选」「多选」类题，一律 `answer_format: "choice"`。
- **必须同时给出标准答案**：单选填 `correct_choice`（如 `"B"`）；多选填 `correct_choices`（如 `["A","C"]`）。字母按 `choices` 数组顺序从 A 起计（跳过空项后的紧凑序号），与批改时的【选项对照表】一致。
- 组合型选项（如「只有 1、2、4 正确」）也按对应字母填写标准答案，不要写选项全文。
