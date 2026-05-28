# Agent 审核系统设计方案

## 概述

将现有的五维并发审核替换为基于 ReAct（Reasoning + Acting）模式的 Agent 审核架构，覆盖发布审核和事后复审两个场景。两个 Agent 共享工具实现但使用不同的 System Prompt。

## 架构总览

```
┌─────────────────────────────────────────────────────────────────┐
│                        Agent 审核系统                            │
├─────────────────────────────┬───────────────────────────────────┤
│     Publishing Agent        │      Post-Review Agent            │
│     (发布审核)               │      (事后复审)                    │
├─────────────────────────────┼───────────────────────────────────┤
│  Phase 0: 硬性关卡          │  触发: 规则驱动                     │
│  Phase 1: 强制基础审核       │  Triage Agent → 快速分流           │
│  Phase 2: Agent ReAct 决策   │  Review Agent → ReAct 深度复审     │
│  Phase 3: 综合裁决           │  裁决: safe / risky / violation    │
├─────────────────────────────┴───────────────────────────────────┤
│                    共享工具箱 (Toolbox)                          │
│  extract_more_frames | transcribe_audio | ocr_frames             │
│  enlarge_frame | review_with_better_model | full_review          │
│  mark_checked | done | escalate                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 一、Publishing Agent（发布审核 Agent）

### Phase 0 — 硬性关卡

不通过直接拒绝，不进入后续阶段。

| 检查项 | 说明 | 结果 |
|--------|------|------|
| 格式校验 | 视频格式、分辨率、时长是否符合平台要求 | pass / reject |
| ffprobe 元数据 | 检查视频流、音频流完整性，是否有损坏 | pass / reject |
| 敏感词 DFA 过滤 | 对文本和封面 OCR 文字进行敏感词匹配 | pass / reject |

敏感词库覆盖 7 大类别（涉政、色情、暴恐、辱骂、广告、赌博、诈骗），共 120+ 词条，基于 DFA + Trie 实现高效匹配。

### Phase 1 — 强制基础审核

Phase 0 全部通过后进入。三项全部完成后 Agent 才获得决策权。

| 审核项 | 内容 | 模型 |
|--------|------|------|
| 文本审核 | 封面文案 + 视频描述 | 多模态 LLM |
| 封面图片审核 | 封面图内容 | 多模态 LLM |
| 抽帧画面审核 | 默认 5 帧，均匀采样 | 多模态 LLM |

Phase 1 三项全部通过 → 进入 Phase 2 Agent 决策。

Phase 1 任一项不通过 → 标记风险，进入 Phase 3 汇总裁决。

### Phase 2 — Agent ReAct 决策

Agent 获得 Phase 0/1 的完整结果后，自主决定后续审核策略。最大轮次 5 轮。

**ReAct 循环：**
```
Thought → Action → Observation → Thought → Action → ... → done
```

Agent 每轮可以调用一个工具，观察结果后决定下一步。常见决策路径示例：

- "封面和文本通过，但 5 帧中有 1 帧可疑" → `extract_more_frames(region=可疑帧前后)` → 确认无问题 → `done`
- "视频有音频轨道，文本审核发现暗示性语言" → `transcribe_audio` → ASR 文字无问题 → `done`
- "画面中出现了文字内容（路牌、字幕等）" → `ocr_frames` → 审核 OCR 文字 → `done`
- "多帧画面模糊看似低俗但不确定" → `review_with_better_model` → 高精度模型确认 → `done`
- "整体质量存疑，多个维度都踩线" → `full_review` → 全面结果 → `done`

### Phase 3 — 综合裁决

汇总 Phase 0/1/2 结果，输出最终意见：

| 裁决 | 条件 | 处理 |
|------|------|------|
| approved | 所有阶段无风险 | 发布成功 |
| rejected | Phase 0 硬性关卡不通过 | 拒绝发布，通知用户 |
| risky | 有风险点但未达拒绝阈值 | 标记为 risky，发布但进入事后复审池 |

---

## 二、Post-Review Agent（事后复审 Agent）

### 触发条件

| 触发来源 | 条件 | 说明 |
|----------|------|------|
| 高风险内容 | 发布时标记为 `risky` | 定时巡检，复审池中的内容 |
| 用户举报 | 举报数 ≥ 阈值（默认 5） | 达到阈值自动触发复审 |
| 定时抽检 | 每小时随机 5% 已发布内容 | 防止漏网，抽查 |

### Triage Agent（分流 Agent）

轻量级模型，快速判断是否需要深入复审。

**输入：** 原审核记录摘要 + 触发原因 + 视频基础信息

**输出：**
- `skip` — 风险低，放行。原 `risky` 标记降级为 `safe`
- `review` — 需要深入复审，进入 Review Agent

限制 1 轮决策，控制成本。

### Review Agent（深度复审 Agent）

与 Publishing Agent 共享工具箱，追加 `escalate` 工具。

**ReAct 循环**（最大 5 轮），决策逻辑参考 Publishing Agent。

输入上下文包含：
- 原始发布审核的全部结果
- Triage Agent 的判定理由
- 举报内容（如有）

### 裁决处理

| 裁决 | 处理 |
|------|------|
| safe | 恢复正常状态，同步更新审核记录 |
| risky | 限流（降低推荐权重），保留标记 |
| violation | 下架内容，通知用户，记录违规 |

---

## 三、共享工具箱

两个 Agent 共享同一套工具实现，在 Prompt 中可见的工具集可以按场景裁剪。

| 工具 | 功能 | 成本 |
|------|------|------|
| extract_more_frames(n, region) | 在指定区间追加抽帧 | 低 |
| transcribe_audio | ASR 语音转文字 | 中 |
| ocr_frames | 对抽帧图片做 OCR 提取文字 | 中 |
| enlarge_frame(frame_id) | 单帧放大，提高审核精度 | 低 |
| review_with_better_model | 切换高精度模型重新审核 | 高 |
| full_review | 启用全部审核维度完整走一遍 | 高 |
| mark_checked(frame_id) | 标记某帧已确认无问题 | 零 |
| done | 结束 ReAct 循环，输出最终判断 | 零 |
| escalate | 升级人工审核（仅 Post-Review） | 零 |

---

## 四、数据模型

### 审核记录扩展

```sql
-- review_records 表新增字段
agent_trace   JSON    -- Agent 的 ReAct 完整轨迹 (thought/action/observation)
agent_rounds  INT     -- 实际执行轮数
agent_verdict VARCHAR -- Agent 最终判断
phase0_result JSON    -- Phase 0 硬性关卡结果
phase1_result JSON    -- Phase 1 强制基础审核结果
```

### Agent Trace 结构

```json
{
  "rounds": [
    {
      "round": 1,
      "thought": "封面和文本通过，但第3帧画面模糊，需要追抽该区域帧",
      "action": "extract_more_frames",
      "params": { "n": 3, "region": "frame_3_nearby" },
      "observation": "追抽3帧，画面清晰无违规内容"
    },
    {
      "round": 2,
      "thought": "所有帧均无问题，结束审核",
      "action": "done",
      "params": null,
      "observation": null
    }
  ],
  "final_verdict": "approved",
  "final_reason": "所有审核维度通过，无风险"
}
```

---

## 五、Prompt 设计要点

### Publishing Agent System Prompt 核心指令

1. 你是内容审核 Agent，目标是在效率和准确率之间取得平衡
2. 文本审核和封面审核通过不代表视频安全，必须结合抽帧结果综合判断
3. 每次决策前先分析当前证据，再决定是否需要更多信息
4. 不要过度审核 — 当证据足够做出判断时，调用 done
5. 不确定时宁可标记 risky，不要直接 approved

### Post-Review Agent System Prompt 核心指令

1. 你是事后复审 Agent，聚焦于可能被初次审核遗漏的风险
2. 复审是二次把关，宁可严一些
3. 仔细阅读举报内容和原始审核记录
4. 有疑问时使用 escalate 升级人工，不要强行判断
5. 复审应该是低成本操作 — 先判断必要性再行动

---

## 六、实现计划

### 阶段一：基础设施（后端）

1. Tool 接口定义和注册机制
2. 9 个 Tool 的实现（extract_more_frames, transcribe_audio, ocr_frames, enlarge_frame, review_with_better_model, full_review, mark_checked, done, escalate）
3. Agent ReAct 引擎（Thought → Action → Observation 循环）
4. Agent Trace 记录和持久化
5. review_records 表结构扩展（migration）

### 阶段二：Publishing Agent

6. Phase 0 实现（格式校验 + ffprobe + 敏感词 DFA）
7. Phase 1 实现（文本 + 封面 + 5 帧抽帧审核）
8. Phase 2 ReAct 循环集成
9. Phase 3 综合裁决逻辑
10. Publishing Agent Prompt 编写和调优

### 阶段三：Post-Review Agent

11. 触发机制实现（risky 巡检 + 举报阈值 + 定时抽检）
12. Triage Agent 实现（轻量 Prompt + 单轮决策）
13. Review Agent ReAct 循环集成
14. 裁决处理（safe/risky/violation 对应的内容状态变更）
15. Post-Review Agent Prompt 编写和调优

### 阶段四：前端适配

16. 审核管理页面展示 Agent Trace（ReAct 过程可视化）
17. 审核结果详情展示各阶段结果
18. 人工审核入口（escalate 升级后的人工处理界面）

---

## 七、成本控制

| 策略 | 说明 |
|------|------|
| Triage 免费/低成本模型 | 大部分内容走 Triage → skip，不用进 ReAct |
| 最大轮次限制 | ReAct 限制 5 轮，避免无限循环 |
| 工具分级定价 | Agent Prompt 中明确每个工具的成本，引导优先低成本工具 |
| 定时抽检比例可调 | 默认 5%，高峰期可降至 1-2% |

---

## 状态: 待审批

下一步：方案确认后进入 writing-plans 生成详细实现计划。
