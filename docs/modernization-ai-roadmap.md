# Boardit 现代化与 AI 化路线图

**文档状态**：草稿 / 待确认
**需求来源**：重启项目维护与现代化、AI 化改造，使其成为 AI 时代可用于简历展示的门面项目
**需求类型**：改造优化
**文档版本**：v1.0
**创建时间**：2026-08-10
**涉及模块**：前端编辑器、笔记与发布、搜索、AI 工作流、后端服务、数据层、测试与交付

---

## 一、产品定位

Boardit 不应只演化为“博客编辑器加一个 AI 聊天框”，建议将产品定位收敛为：

> **从私人知识到可信发布的 AI 原生内容工作台。**

用户可以基于自己的笔记进行检索和问答，生成带来源引用的草稿，审阅 AI 修改，执行发布前质量检查，最终公开发布；后续还可通过 MCP 让外部 AI 客户端安全访问 Boardit 中的内容。

该定位能够用一条连续产品流程承载 RAG、流式交互、结构化输出、异步任务、人工审批、评测、可观测性、安全边界和 MCP，而不是堆叠彼此无关的 AI 功能。

## 二、当前项目基础

### 已具备能力

- React 19、TypeScript、Vite、Tailwind CSS 和 React Query 前端。
- Go、Gin、GORM 和 PostgreSQL 后端。
- OpenAPI 契约与 Orval 生成的前端类型和请求 hooks。
- MSW Mock、Docker Compose 和 GitHub Actions。
- 用户认证、笔记、Markdown 编辑、文件夹、公开与非公开发布。
- 后端自动化测试、前端 lint 和生产构建当前均可通过。

### 主要缺口

- `frontend/src/pages/Editor.tsx` 和 `backend/internal/handler/note.go` 均超过 1,000 行，职责集中。
- 当前已有安全、数据一致性、保存状态和发布行为问题，详见 `docs/bug-report.md`。
- `NoteRevision` 数据模型存在，但未形成版本历史接口与用户体验。
- description、tags 已出现在编辑器 UI 中，但数据模型和接口并不支持持久化。
- 私有笔记搜索仍是数据库 `LIKE` 匹配，尚无全文索引、语义检索和引用能力。
- 缺少前端自动化测试、端到端测试、正式数据库迁移和 AI 可观测性。
- 缺少在线演示、架构说明、演示视频、质量指标和完整 Case Study。

## 三、目标用户体验

建议围绕以下 3 分钟演示路径设计产品：

1. 用户导入或编写多篇私人笔记。
2. 用户询问“基于这些笔记，整理一篇关于 X 的文章”。
3. 系统执行关键词与向量混合检索，并在回答中引用具体笔记段落。
4. 用户将回答转为文章草稿。
5. AI 检查结构、重复内容、语气、来源完整性和发布元信息。
6. 用户查看修改 diff，逐项接受或拒绝。
7. 用户发布公开文章。
8. 外部 AI 客户端通过 Boardit MCP 搜索或读取已授权内容。

### 异常体验

| 场景 | 系统行为 |
|---|---|
| AI 请求超时或失败 | 保留原文，显示可理解的失败原因并允许重试 |
| 用户取消生成 | 立即中止生成，不写入笔记 |
| 检索不到可信来源 | 明确提示证据不足，不伪造引用 |
| AI 输出格式不合法 | 服务端拒绝应用结果并记录失败原因 |
| 保存发生版本冲突 | 展示服务端版本与本地 diff，由用户选择合并方式 |
| 高风险写操作 | 必须由用户确认后执行 |

## 四、功能方向与优先级

| 方向 | 产品价值 | 简历展示价值 | 优先级 |
|---|---:|---:|---:|
| 选中文本润色、改写、扩写和生成标题 | 中 | 中 | P0 |
| 流式生成与 diff 接受/拒绝 | 高 | 高 | P0 |
| 私人知识库问答，答案带段落引用 | 很高 | 很高 | P0 |
| 混合搜索、语义关联和相关推荐 | 高 | 很高 | P1 |
| 研究 → 大纲 → 草稿 → 审校 → 发布工作流 | 很高 | 很高 | P1 |
| AI 运行、成本、延迟和反馈看板 | 中 | 很高 | P1 |
| Boardit MCP Server | 中 | 很高 | P2 |
| 图片生成、语音输入和多智能体 | 待验证 | 中 | P3 |

## 五、建议技术边界

### 前端

- 编辑器按笔记树、正文编辑、元信息、保存协调、AI 助手和发布流程拆分功能模块。
- 所有 AI 生成都采用流式反馈，支持取消、重试和明确的加载状态。
- AI 修改以 diff 或候选版本呈现，未经确认不得覆盖用户正文。
- 服务端状态交给 React Query；编辑草稿和界面状态保持在功能模块内部。

### 后端

- 按 notes、search、ai、workflow、jobs 和 platform 拆分领域职责。
- HTTP handler 只承担协议转换、鉴权入口和响应映射，业务规则进入可测试的 service/use-case。
- 数据访问进入 repository 或领域专用 store，避免业务流程直接依赖全局 `database.DB`。
- AI 提供商封装在适配器边界中；业务层不直接依赖具体模型 SDK。
- 长耗时索引和生成任务进入可恢复的后台任务机制，不直接依赖请求生命周期。

### 建议数据模型

- `note_revisions`：可恢复的笔记历史版本。
- `note_chunks`：检索分块、位置和 embedding。
- `sources`：外部或内部来源及其快照信息。
- `ai_runs`：模型、prompt 版本、输入来源、token、延迟、状态和错误。
- `ai_feedback`：接受、拒绝和人工评分。
- `prompt_versions`：可追踪的 prompt 版本。
- `jobs`：异步任务状态、重试次数和幂等键。

### 搜索与模型接入

- 继续以 PostgreSQL 为核心数据存储，引入全文搜索和 pgvector。
- 使用全文检索与向量检索的混合排序，而不是只依赖向量相似度。
- 若首发采用 OpenAI，使用当前 Responses API、流式响应和结构化输出能力；领域层仍保留提供商适配边界。

## 六、AI 工程原则

- 所有 AI 写操作先生成候选修改，由用户确认后保存或发布。
- 外部网页、文件和检索结果统一视为不可信输入。
- 模型只获得完成当前任务所需的最小工具和最小数据范围。
- 结构化结果使用 schema 校验，不能只靠 prompt 约定格式。
- 每次 AI 运行记录模型、prompt 版本、引用、token、延迟、成本、状态和用户反馈。
- 遥测默认不采集完整私人笔记；敏感内容记录需要脱敏和显式配置。
- AI 接口必须具有超时、取消、有限重试、幂等、配额和按用户限流。
- RAG 查询必须在数据库查询层强制限定用户作用域，并包含跨用户泄漏测试。
- 对高风险工具调用保留人工审批，不将发布和外部写操作完全交给模型。
- 每项 AI 功能必须同时交付代表性 eval、失败体验和可观测指标。

## 七、分阶段路线

### Phase 0：可信工程基线（建议 1–2 周）

- 修复现有 bug 报告中的高、中风险问题。
- 拆分 Editor 和 note handler 的职责。
- 引入正式、可回滚的数据库迁移。
- 完善 refresh token、注销和凭据存储策略。
- 增加前端组件测试、端到端测试和 PR 级 CI。
- 增加健康检查、结构化日志、request ID 和基础指标。
- 清理死代码、未持久化的假字段、未使用依赖和重复 lockfile。
- 更新 README、部署说明和架构说明。

### Phase 1：AI 编辑器 MVP（建议 1–2 周）

- 选中文本 AI 操作。
- 流式输出、取消与失败重试。
- 修改前后 diff 与人工确认。
- 自动生成标题、摘要、标签和发布元信息。
- AI run、反馈和成本记录。
- 建立第一批代表性 eval 数据集。

### Phase 2：知识库与 RAG（建议约 2 周）

- 笔记分块和异步索引。
- PostgreSQL 全文搜索与 pgvector 混合检索。
- 基于私人笔记的问答。
- 可点击的笔记与段落引用。
- 语义相关推荐和反向链接。
- 跨用户隔离与 prompt injection 测试。

### Phase 3：可信发布工作流（建议约 2 周）

- 研究、大纲、草稿、审校和发布状态机。
- 来源快照、引用完整性和发布前检查。
- 后台长任务、失败恢复和有限重试。
- MCP 只读接入。
- AI 质量、延迟、成本和用户反馈看板。

## 八、本阶段不包含

- 不在 Phase 0 同时重写前后端技术栈。
- 不在没有 eval 证明收益前引入多智能体编排。
- 不在检索和反馈数据不足时进行模型微调。
- 不默认允许 AI 自动发布或执行外部写操作。
- 不为展示概念而提前引入 Kubernetes、微服务或复杂消息基础设施。

## 九、验收标准

### 产品闭环

- [ ] 可从私人笔记检索来源、生成草稿、审阅修改并发布文章。
- [ ] 所有来源引用可定位到具体笔记和段落。
- [ ] 所有 AI 写操作都能在不修改原文的情况下取消或拒绝。

### 工程质量

- [x] 关键业务规则不再依赖 HTTP handler 才能测试。
- [ ] 前端保存、冲突、AI 生成和发布主流程具有自动化测试。
- [x] 数据库结构变更由版本化迁移管理。
- [ ] 每个 AI run 可追踪模型、prompt、来源、token、延迟和结果状态。
- [x] CI 能阻止类型、lint、测试和契约回归进入主分支；eval 门禁随首个 AI 功能加入。

### 安全与隔离

- [ ] 不存在跨用户读取、检索、embedding 或引用泄漏。
- [ ] 高风险写操作需要用户确认。
- [ ] 外部不可信内容不能直接改变系统工具权限。
- [ ] 模型密钥仅存在于服务端配置中。

### 简历展示

- [ ] README 包含产品定位、架构图、演示路径和关键设计决策。
- [ ] 提供在线 Demo、演示视频和可复现的本地启动方式。
- [ ] 展示真实 eval、引用准确性、P95 延迟和单次运行成本。

## 十、待确认问题

- [ ] **[产品]** 是否确认“私人知识到可信发布”作为唯一主产品主线？
- [ ] **[产品]** 第一阶段是否优先服务个人创作者，而不是团队协作？
- [ ] **[产品+开发]** 是否接受 OpenAI 首发、领域层保持模型可替换的策略？
- [ ] **[开发]** 是否先完成 Phase 0 的模块边界与测试保护，再开始 AI 功能？
- [ ] **[开发]** MCP 是否仅在核心 RAG 和权限模型稳定后进入 P2？

## 十一、外部参考

- OpenAI Model Guidance：https://developers.openai.com/api/docs/guides/latest-model
- OpenAI Streaming Responses：https://developers.openai.com/api/docs/guides/streaming-responses
- OpenAI Structured Outputs：https://developers.openai.com/api/docs/guides/structured-outputs
- OpenAI Evaluation Best Practices：https://developers.openai.com/api/docs/guides/evaluation-best-practices
- pgvector Hybrid Search：https://github.com/pgvector/pgvector#hybrid-search
- OWASP LLM01 Prompt Injection：https://genai.owasp.org/llmrisk/llm01-prompt-injection/
- MCP Authorization：https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization
- OpenTelemetry GenAI Observability：https://opentelemetry.io/blog/2026/genai-observability/

## 十二、变更记录

| 版本 | 日期 | 修改内容 | 修改人 |
|---|---|---|---|
| v1.0 | 2026-08-10 | 根据代码现状与 AI 时代简历展示目标创建初版 | AI |

---

> 下一步：产品确认第三章目标体验，开发确认第五章技术边界；确认后先执行 Phase 0，再进入 AI 功能建设。
