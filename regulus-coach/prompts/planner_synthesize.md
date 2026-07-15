## 合成规划（synthesize 阶段）

目标：把过载压成「可回看的聚焦」——北星一句、少量清障、今日一条学习。

根据对话历史与【上下文】中的画像/课程，输出结构化规划：

1. **situation_summary**：用一句话概括用户现状（过载点 + 学习卡点）。
2. **focus**（必填）：
   - `north_star`：当下应钉住的专注目标（一句话，用户稍后确认是否钉住）
   - `why`：为何此刻优先这个（短）
   - `week_wedge`：本周只押的一条能力线或一门已有课
   - `today_learning`：今日 **唯一** 学习动作；`minutes` 建议 15；若【用户课程】有匹配，填 `matched_domain_id` / `matched_node_key`（须来自上下文，**勿编造 ID**）
3. **clear_first**：先清障、可快速了结的事（最多 3 条，每项 ≤15 分钟，写清 `next_step`）——过载时这是恢复动力的关键。
4. **matrix**（Eisenhower，作补充详情）：
   - `important_urgent` / `important_not_urgent` / `quick_wins` / `defer_or_drop`
   - `defer_or_drop` 必须给出「暂缓/砍掉」的许可理由
5. **action_plan**：
   - `today`：最多 **3** 条；其中 `kind=learning` **最多 1 条**（与 `focus.today_learning` 一致）；其余为清障或必要事务
   - `this_week`：最多 5 条
6. **learning_focus**：可与 `today_learning` 对齐；匹配字段规则同上。不要塞多条互相打架的学习线。
7. **mindset_note**：一句对抗「瘫痪/完美主义」的短鼓励，不空洞。

要求：
- 信息预算：清晰 > 完备。宁少勿滥。
- 每条行动带具体 next_step 或 reason，避免空泛表述。
- 若用户已在某课程有进度，**优先**推荐继续该课下一节点，而非新开领域。
- 不要主动推销「再建一门新课」。
- 不要输出 `ui_state`（钉住与勾选由客户端写入）。
- 若用户在 plan_ready 阶段要求调整（如「今日只要 2 条」），在保留合理结构的前提下按新要求重写。
