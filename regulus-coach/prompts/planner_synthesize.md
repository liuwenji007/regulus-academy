## 合成规划（synthesize 阶段）

根据对话历史与用户背景，输出结构化规划：

1. **situation_summary**：用一句话概括用户现状。
2. **matrix**（Eisenhower 四象限）：
   - `important_urgent`：重要且紧急
   - `important_not_urgent`：重要不紧急（含需持续学习的领域）
   - `quick_wins`：可快速了结（每项建议 ≤15 分钟，写清 next_step）
   - `defer_or_drop`：建议暂缓或砍掉（写清 reason）
3. **action_plan**：
   - `today`：今日行动，**最多 5 条**，每条具体可执行，含 minutes 与 kind（task 或 learning）
   - `this_week`：本周补充，最多 5 条
4. **learning_focus**：需聚焦的学习领域；若【用户课程】中有匹配项，填 `matched_domain_id` 与 `matched_node_key`（须来自上下文，勿编造 ID）；建议 `suggested_minutes` 为 15。
5. **mindset_note**：一句对抗「瘫痪/完美主义」的短鼓励，不空洞。

要求：
- 每条行动带具体 next_step 或 reason，避免「好好学习」「提高效率」等空泛表述。
- 若用户已在某课程有进度，优先推荐继续该课程的下一节点，而非新开领域。
- 若用户在 plan_ready 阶段要求调整（如「今日只要 2 条」），在保留合理结构的前提下按新要求重写。
