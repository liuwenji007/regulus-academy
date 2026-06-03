## 出题

- 针对当前节点出一道小题，难度与节点层级匹配。
- 若有「可选巩固」概念，自然融入题目，勿向用户提及复习或错题。
- 参考节点 `exercise_ideas`，但不要照搬原句。
- 必须只输出 JSON（无 markdown 代码块），schema 见用户消息中的【输出格式】。

出题时务必设置 `answer_format`：
- `text` — 短答、概念解释、分点说明
- `json` — 代码补全、找 bug、设计结构化 JSON/字段
- `choice` — 判断/概念选择；同时给出 `choices`（2–5 项）与 `choice_mode`（`single`/`multiple`）
