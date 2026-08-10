# Boardit AI 化前重构就绪度评估

**文档状态**：草稿 / 待确认
**需求来源**：在继续现代化与 AI 化之前，评估项目是否需要重构、拆分大模块并降低不合理耦合
**需求类型**：改造优化
**文档版本**：v1.0
**创建时间**：2026-08-10
**涉及模块**：前端编辑器、API 与认证、后端笔记领域、数据层、测试、CI/CD

---

## 一、结论

**项目需要先进行一轮有边界的重构，但不需要重写，也不建议拆成微服务。**

推荐策略是：

1. 保留 React、Go、Gin、GORM、PostgreSQL、OpenAPI 和 Orval。
2. 将现有系统整理为“模块化单体”，优先恢复清晰的领域边界和依赖方向。
3. 先修复已知数据一致性与权限问题，再进行以行为不变为目标的结构拆分。
4. 为笔记、文件夹、认证和保存流程建立测试保护后，再加入 AI 写入、RAG、异步索引和 MCP。
5. 追求“高内聚、低耦合”，而不是机械地增加目录、接口和抽象层。

如果原问题中的“增加部分不合理代码的耦合程度”是指处理耦合问题，本评估建议是：**降低不合理的跨层耦合，同时提高同一业务模块内部的内聚程度。**

## 二、为什么 AI 化前必须处理

普通 CRUD 中的边界问题通常只影响一次请求；加入 AI 后，同一问题会扩散到更多数据副本和长流程：

- 权限规则遗漏会进一步造成跨用户检索、embedding 或引用泄漏。
- 保存和版本冲突会导致 AI 结果覆盖用户正文。
- 缺少事务会使批量重排、版本记录、索引任务和正文状态不一致。
- 没有稳定领域事件或任务边界时，embedding 很容易与最新笔记不同步。
- UI 状态与服务端请求混在单一组件中，会使流式生成、取消、diff 和冲突处理难以组合与测试。
- 缺少 AI 运行记录和可测试服务层时，后续无法建立可信 eval 和可观测性。

因此，重构不是独立的“代码洁癖”工作，而是 AI 功能的数据安全、可恢复性和可评测性的前置条件。

## 三、当前结构现状

### 前端主流程

1. `App.tsx` 负责路由和登录保护。
2. `Editor.tsx` 同时负责笔记列表、文件夹树、正文编辑、页面元信息、保存、删除、移动、快捷键、离开确认、弹窗和全部 UI。
3. Editor 直接调用 Orval 生成的 React Query hooks，并通过手动 `refetch` 同步数据。
4. 公开列表和详情页绕过生成客户端，使用手写 Axios 请求和重复 DTO。
5. `AuthContext` 负责用户态和 token 持久化，Axios 层反向引用 AuthContext 中的 token 函数。

### 后端主流程

1. Router 直接绑定 Gin handler。
2. Handler 负责参数解析、用户作用域、业务规则、GORM 查询、事务含义、Markdown 转换、响应 DTO 和错误映射。
3. 所有 handler 直接使用包级全局 `database.DB`。
4. `note.go` 同时承载笔记、文件夹、树重排、公开列表和 Markdown/slug 工具逻辑。
5. 数据库启动时使用 GORM `AutoMigrate`，没有显式迁移历史。

### 当前测试保护

- 后端已有认证、JWT、router 和数据库初始化测试。
- `handler_test.go` 当前主要覆盖认证流程，没有覆盖笔记、文件夹、发布、重排和并发冲突。
- 前端没有组件测试、hook 测试或端到端测试文件。
- CI 能执行 Go 测试、前端 lint 和构建，但前端安装时删除 lockfile 后执行 `npm install`，不能保证依赖解析完全可重复。

## 四、主要耦合与风险

| # | 问题 | 代码证据 | AI 化影响 | 严重度 |
|---|---|---|---|---|
| 1 | Editor 同时承担业务协调、服务端状态和大型 UI | `frontend/src/pages/Editor.tsx:131-1083` | AI 流式状态、候选 diff、冲突和保存状态会继续集中到单一组件 | 高 |
| 2 | 前端认证与 HTTP 客户端形成循环依赖 | `frontend/src/api/axios.ts:3`；`frontend/src/contexts/AuthContext.tsx:8` | token 刷新、登出、SSE 和 MCP 授权难以独立测试和替换 | 高 |
| 3 | API 使用方式不统一 | `frontend/src/pages/PostList.tsx:5-38`；`frontend/src/pages/PostDetail.tsx:5-53` | 手写 DTO 与 OpenAPI 生成类型可能漂移，错误与缓存策略不一致 | 中 |
| 4 | 登录、注册页面与 AuthModal 重复验证和表单逻辑 | `frontend/src/pages/Login.tsx:8-49`；`frontend/src/pages/Register.tsx:7-57`；`frontend/src/components/AuthModal.tsx:49-122` | 规则修改需要多处同步，现有前后端密码规则已经发生漂移 | 中 |
| 5 | Handler 直接依赖全局 GORM DB | `backend/internal/database/database.go:15`；`backend/internal/handler/note.go` 多处 | 业务规则难以单测，AI 任务和 HTTP 请求无法共享稳定用例层 | 高 |
| 6 | note handler 聚合多个领域职责 | `backend/internal/handler/note.go:22-1081` | RAG、版本、发布和异步索引继续加入后会放大变更影响面 | 高 |
| 7 | 用户资源归属规则散落在各查询分支 | `backend/internal/handler/note.go:169,424,474,741,906,924` | 已出现文件夹归属遗漏；未来可能扩散为跨用户检索泄漏 | 高 |
| 8 | 批量重排不是原子操作 | `backend/internal/handler/note.go:889-939` | 部分更新失败后树状态不一致，后续索引和版本任务难以判断真实状态 | 高 |
| 9 | 响应使用多处临时 `gin.H` 手工拼装 | `backend/internal/handler/note.go:199,351,442,502,537,1006,1061` | OpenAPI、实际响应和 AI 工具 schema 容易漂移 | 中 |
| 10 | 生产结构由 AutoMigrate 管理 | `backend/internal/database/database.go:42-50` | embedding、jobs、AI runs 等数据表需要可审计、可回滚的迁移过程 | 高 |
| 11 | NoteRevision 只有模型，没有写入与读取链路 | `backend/internal/model/note_revision.go:6-12`；`backend/internal/router/router.go:49-76` | AI 修改缺少可靠恢复点和版本审计 | 高 |
| 12 | 搜索直接使用 `LIKE` | `backend/internal/handler/note.go:85-99` | 无法稳定承载段落级引用、混合检索和可评测召回 | 中 |
| 13 | 认证 token 没有显式 token 类型和服务端会话状态 | `backend/internal/handler/auth.go:103-126,144-189` | access token 与 refresh token 的用途边界不够明确，也无法撤销单个会话 | 高 |
| 14 | 浏览器长期 token 直接存入 localStorage | `frontend/src/contexts/AuthContext.tsx:19-40` | XSS 后可直接获取长效 refresh token；不适合作为后续 MCP/OAuth 基础 | 高 |
| 15 | 公开 URL 的用户名和 slug 边界不稳定 | `backend/internal/model/user.go:10-11`；`backend/internal/handler/note.go:468-471` | 用户名未声明唯一约束，改标题会改变 slug，已发布内容可能断链 | 中 |
| 16 | API 契约声明与实现存在漂移 | `backend/docs/api/api-contract-v1.md:202,322`；`backend/internal/handler/note.go:285-313` | 契约声称 HTML 已清洗，但当前转换逻辑不满足该承诺 | 高 |

## 五、哪些部分必须先重构

### P0：AI 功能开始前完成

#### 1. 领域安全与数据一致性

- 修复 `docs/bug-report.md` 中的高、中风险问题。
- 将笔记、文件夹、父子关系和发布可见性规则集中到用例/service 层。
- 所有资源访问都使用显式的用户作用域，不允许调用方遗漏 `user_id`。
- 树重排、正文更新与版本创建使用明确事务边界。
- 明确 slug 是否不可变；如可变，需要重定向或历史 slug 策略。

#### 2. 后端依赖方向

目标依赖方向：

`HTTP handler → application/use-case → domain policy → repository/store interface → GORM adapter`

- Handler 只处理 HTTP 输入输出。
- Use-case 组合保存、版本、发布和权限规则。
- Repository 负责数据库查询和用户作用域。
- 配置、DB、clock、token signer 通过应用装配注入，逐步移除业务代码对包级全局变量和 `os.Getenv` 的直接依赖。

不要求为每张表建立完全通用的 repository；只为真实业务用例建立最小接口。

#### 3. 编辑器状态边界

建议将 Editor 拆为以下职责，而不是简单按 JSX 长度切文件：

- `note-tree`：笔记/文件夹加载、选择、移动和树状态。
- `editor-session`：当前草稿、dirty 状态、版本和冲突。
- `note-editor`：MDXEditor 适配与正文输入。
- `note-metadata`：标题、封面、可见性、description 和 tags。
- `save-coordinator`：防抖保存、取消、版本号和失败恢复。
- `publishing`：发布状态和公开链接。
- `ai-assistant`：后续独立接入流式生成、候选 diff 和反馈。

其中组件负责呈现，feature hook 或 reducer 负责编辑会话，React Query 负责服务端缓存；避免把全部状态搬到一个新的全局 store。

#### 4. API 与认证边界

- 让 HTTP 客户端依赖独立 token/session store，而不是依赖 React Context。
- AuthContext 消费 auth service，不反向被 Axios 导入。
- 统一使用 Orval requester 和生成类型；公开页面也进入同一数据访问策略。
- 统一 API error 类型、401 状态转换、队列成功与失败处理。
- refresh token 建议迁移到 Secure、HttpOnly、SameSite Cookie，并引入可撤销会话；access token 保持短期有效。

#### 5. 数据库迁移与测试保护

- 引入版本化 SQL 迁移，并将 AutoMigrate 限制到测试或移除生产使用。
- 先为现有行为建立 characterization tests，再移动代码。
- 为笔记更新、文件夹归属、树重排、发布可见性、版本冲突和 token refresh 增加后端测试。
- 为登录、编辑、保存、冲突和发布增加前端组件测试及最小 Playwright 主流程。

### P1：可随第一批 AI 功能演进

- `NoteRevision` 正式接入更新事务和版本恢复体验。
- 建立 note changed 事件或事务性 outbox，驱动分块与索引任务。
- 引入 jobs、ai_runs、prompt_versions 和 feedback。
- 添加 SSE 流式边界、请求取消和幂等键。
- 引入全文搜索、pgvector 和引用数据模型。
- 加入 OpenTelemetry、token/延迟/成本指标和敏感字段脱敏。

### P2：暂缓

- 微服务拆分。
- 独立消息集群；初期可使用 PostgreSQL-backed job/outbox。
- 多智能体框架。
- 通用插件系统。
- 多模型动态路由平台。
- Kubernetes 和复杂服务网格。

这些能力当前不会解决主要风险，反而会增加部署、调试和演示成本。

## 六、建议目标结构

### 前端目标结构

```text
src/
├── app/                    # Provider、router、全局错误边界
├── features/
│   ├── auth/
│   ├── notes/
│   ├── editor/
│   ├── publishing/
│   ├── search/
│   └── ai-assistant/
├── shared/
│   ├── api/                # requester、session store、生成客户端
│   ├── ui/
│   └── lib/
└── api/gen/                # OpenAPI 生成文件
```

### 后端目标结构

```text
backend/
├── cmd/api/                # 进程入口和依赖装配
├── migrations/
└── internal/
    ├── auth/
    ├── notes/              # 笔记、文件夹、版本与发布边界
    ├── search/
    ├── ai/
    ├── workflow/
    ├── jobs/
    └── platform/           # config、db、http、telemetry
```

目录不是验收目标；依赖方向、业务规则位置和可测试性才是验收目标。

## 七、不建议的重构方式

- 不进行一次性 big-bang rewrite。
- 不先创建大量空接口、BaseService、GenericRepository 或通用事件总线。
- 不把 Go 单体过早拆成多个网络服务。
- 不同时更换编辑器、状态库、路由、ORM 和部署平台。
- 不在没有测试保护时大范围移动与重命名代码。
- 不将所有编辑器状态放入全局 store；草稿状态应按编辑会话隔离。
- 不将 OpenAPI 生成代码与业务逻辑混合修改。
- 不让 AI 功能直接写 GORM model 或绕过笔记用例层。

## 八、推荐执行顺序

### R0：建立重构保护网（1–2 天）

- 固化当前测试、lint 和 build 基线。
- 增加笔记/文件夹关键行为测试和最小编辑发布 E2E。
- 在 CI 中改用锁文件确定性安装，并增加 pull request 触发。
- 记录 API 契约与当前行为差异。

### R1：修复数据与权限问题（2–3 天）

- 修复高风险 bug。
- 加入用户作用域、文件夹环检测和批量重排事务。
- 修复并发版本精度、unlisted 发布和 refresh 队列。
- 对修复后的行为补回归测试。

### R2：后端模块化（3–5 天）

**执行状态**：✅ 2026-08-10 已完成

- 抽出 notes 用例、repository 和 DTO mapping。
- 拆分 notes、folders、publishing 和 tree HTTP handler。
- 集中 slug、Markdown、权限与事务规则。
- 引入应用装配和显式配置。

### R3：前端模块化（3–5 天）

**执行状态**：✅ 2026-08-10 已完成

- 解除 AuthContext 与 Axios 循环依赖。
- 统一 OpenAPI 客户端与错误处理。
- 抽取 editor session、save coordinator、note tree 和 metadata。
- 合并重复认证表单逻辑，移除死代码与开发调试副作用。

### R4：数据演进基础（2–3 天）

**执行状态**：✅ 2026-08-10 已完成

- 引入正式迁移。
- 接通 NoteRevision。
- 为后续 jobs、outbox 和 AI runs 预留稳定边界，但不提前实现完整 AI 工作流。

建议总量约为 **8–14 个专注开发日**。实际工作可以按小 PR 交错进行，不需要先关闭所有产品开发数周。

## 九、代码支持度评估

**总体支持程度**：中
**改造复杂度**：高，但可以增量实施

| 维度 | 支持程度 | 结论 |
|---|---|---|
| UI 组件 | ✅ 已支持 | 笔记树、元信息、确认弹窗和保存协调已从 Editor 分离 |
| 数据获取 | ✅ 已支持 | 公开与私有页面统一使用 OpenAPI、Orval requester 和 React Query |
| 状态管理 | ✅ 已支持 | 服务端缓存、编辑快照、版本冲突和局部 UI 状态边界明确 |
| 权限控制 | ✅ 已支持 | 资源查询强制用户作用域，refresh 会话可轮换、重放拒绝和撤销 |
| 路由 | ✅ 已支持 | 路由规模简单，可保持现状并按模块注册 |
| 类型定义 | ✅ 已支持 | OpenAPI 是前端 API 类型与 hooks 的生成来源，生成漂移由 CI 拦截 |
| 数据层 | ✅ 已支持 | 版本化 SQL 迁移、repository 边界和显式事务已覆盖核心写路径 |
| 自动化测试 | ✅ 已支持 | 后端 use-case/HTTP/迁移、前端单测和 Playwright 主流程均已接入 CI |
| 部署 | ✅ 已支持 | Docker Compose 适合当前规模，无需升级为复杂编排平台 |

## 十、验收标准

### 行为保持

- [x] 纯结构重构 PR 不改变现有 API 契约和用户可见行为。
- [x] 每个已知 bug 修复均具有独立回归测试。
- [x] 前后端 lint、类型检查、构建和测试持续通过。

### 模块边界

- [x] Editor 页面不再直接协调所有笔记树、保存、元信息和弹窗行为。
- [x] HTTP handler 不直接承载核心权限、事务和保存规则。
- [x] AuthContext 与 Axios 不存在循环依赖。
- [x] 公开和私有页面使用统一的 API 类型、错误与缓存策略。
- [x] 新的 AI 模块只能通过笔记 use-case 修改内容，不能直接访问 GORM model。

### 数据与安全

- [x] 资源查询默认强制用户作用域，跨用户测试覆盖 notes、folders、revisions 和 search。
- [x] 树重排和笔记更新/版本创建具有原子事务。
- [x] 数据库变更有版本号、向前迁移和回滚说明。
- [x] refresh token 可被识别、轮换和撤销，不与 access token 混用。

### AI 就绪

- [x] 笔记更新能够可靠产生 revision 或变更事件。
- [x] 可以在不依赖 HTTP/Gin 的情况下测试笔记更新与发布用例。
- [x] 后续索引任务能够通过 note ID、version 和 user ID 判断数据是否仍有效。
- [x] AI 候选内容可以在确认前独立存在，不覆盖当前正文。

## 十一、待确认问题

- [ ] **[产品]** 是否接受先投入 8–14 个开发日建立可信基线，再开始 P0 AI 功能？
- [ ] **[产品]** 已发布文章的 slug 是否应保持稳定，标题修改是否允许改变公开 URL？
- [ ] **[产品]** description 和 tags 是正式产品字段，还是应先从 UI 移除？
- [ ] **[开发]** 是否确认采用模块化单体而非微服务？
- [ ] **[开发]** 是否接受先以 PostgreSQL job/outbox 支撑异步任务，暂不引入外部消息系统？
- [ ] **[产品+开发]** refresh token 是否迁移为 HttpOnly Cookie，并为未来 MCP 单独设计 OAuth 授权边界？

## 十二、变更记录

| 版本 | 日期 | 修改内容 | 修改人 |
|---|---|---|---|
| v1.0 | 2026-08-10 | 完成 AI 化前模块边界、耦合、风险和重构顺序评估 | AI |
| v1.1 | 2026-08-10 | 完成 R0–R2：测试保护网、数据/认证不变式与后端模块化 | AI |
| v1.2 | 2026-08-10 | 完成 R3：前端 API/认证解耦、Editor 模块化、依赖升级与回归测试 | AI |
| v1.3 | 2026-08-10 | 完成 R4：版本化 SQL 迁移、revision/outbox 原子写入与 AI 异步数据边界 | AI |

---

> 下一步：产品确认“先重构后 AI”的时间投入和 slug/tags 产品决策；开发确认代码支持度评估与模块化单体方向。确认后从 R0 测试保护网和 R1 数据/权限修复开始。
