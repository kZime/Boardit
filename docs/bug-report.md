# Bug 排查报告

> 排查日期：2026-08-04
> 排查范围：backend (Go + Gin + GORM) 与 frontend (React + Vite + TS)
> 说明：#3/#11 为当前未被前端触发的隐患；其余均为可复现的实际问题。

## 维护状态（2026-08-10）

| # | 状态 | 回归保护 |
|---|---|---|
| 1 | ✅ R1 已修复 | 重排失败整体回滚，DB 错误不再继续保存零值 model |
| 2 | ✅ R1 已修复 | note/folder 外键强制用户作用域，覆盖自环、后代环和跨用户关联 |
| 3 | ✅ R1 已修复 | 时间版本使用微秒存储精度与 RFC3339Nano 响应，覆盖成功往返和过期冲突 |
| 4 | ✅ R1 已修复 | 并发 401 共享单个 refresh Promise，成功全部重试，失败全部 reject |
| 5 | ✅ R1 已修复 | Playwright 验证 unlisted 保存发送 `is_published: true` |
| 6 | ✅ R1 已修复 | 取消标题逐键 PATCH，显式保存后立即清理 dirty state |
| 7 | 🟡 计划在 R3 修复 | 与 Editor session/query 拆分一起引入完整分页 |
| 8 | ✅ R1 已修复 | 负 offset 统一钳制为 0 |
| 9 | ✅ R1 已修复 | 前后端统一为 3–32 位 URL-safe 用户名 |
| 10 | ✅ R1 已修复 | 前后端统一为 8 位最小密码，后端限制 bcrypt 72 字节上限 |
| 11 | ✅ R1 已修复 | 存储 HTML 前先转义原始 HTML，回归测试覆盖 script 标签 |

依赖审计已从 52 个告警降至 4 个；剩余 1 个 high 来自 MDXEditor 3.x 的 `js-yaml`，3 个 moderate 来自 React Router 6.x。两者均需要主版本升级，已放入 R3 并要求通过编辑器与路由 E2E 后才提交；CI 当前对生产依赖的 critical 告警失败。

## 🔴 高严重度

### 1. `ReorderTree` 笔记循环吞掉 DB 错误 → 静默插入脏数据
- 位置：`backend/internal/handler/note.go:924-935`
- 现象：笔记循环里，`First(&n)` 返回**非** `ErrRecordNotFound` 的 DB 错误时，没有 else 分支、没有 return，直接落下继续执行。此时 `n` 为零值，随后 `database.DB.Save(&n)` 因 `ID==0` 被 GORM 当作 **INSERT**，插入一条 `user_id=0`、空标题的空笔记。
- 对比：同函数的 folders 循环（906-918 行）正确处理了 else 分支，此处为遗漏。
- 触发条件：任何瞬态 DB 错误（连接断开、唯一约束冲突等）。

### 2. 跨用户文件夹授权缺失
- 位置：
  - `UpdateNote` 设置 `folder_id`：`note.go:474-477`（不校验文件夹归属）
  - `ReorderTree` 设置 `folder_id`：`note.go:930-931`（不校验文件夹归属）
  - `CreateFolder` / `UpdateFolder` 设置 `parent_id`：`note.go:671-675`、`note.go:758-761`（不校验父文件夹归属，且不防环）
- 影响：
  - 用户 A 可把笔记挂到用户 B 的 `folder_id` 下，导致 B 的 `DeleteFolder` 因 `noteCount > 0`（`note.go:852`）**无法删除自己的文件夹**。
  - 可把文件夹 `parent_id` 设为自己或后代，形成循环引用，前端树渲染死循环。
- 对比：`CreateNote`（`note.go:168-172`）有 `folder_id AND user_id` 校验，此处为遗漏。

## 🟠 中严重度

### 3. 乐观并发版本检查永远 409
- 位置：`backend/internal/handler/note.go:441`
- 现象：服务端 `note.UpdatedAt = time.Now()`（带纳秒），但响应用 `.Format(time.RFC3339)` 序列化（`note.go:513` 等），RFC3339 布局不含小数秒。客户端回传的 `updated_at` 解析后纳秒为 0，与 DB 内完整精度 `Equal()` 永远不等 → 按契约带 `updated_at` 的第二次更新必 409。
- 现状：前端未发送 `updated_at`，暂未触发；但该特性本身不可用。

### 4. axios 刷新失败时排队请求永久挂起
- 位置：`frontend/src/api/axios.ts:74-94`
- 现象：refresh 失败时 `catch` 只 reject 当前请求，`subscribers` 中已排队的请求**永远不 settle**，表现为按钮转圈、请求无限 pending。
- 修复方向：catch 里遍历 `subscribers` 逐个 reject，再清空数组。

### 5. "Unlisted" 可见性选项是死功能
- 位置：`frontend/src/pages/Editor.tsx:344`（以及 `:436`、`:469`）
  ```ts
  is_published: pageDetails.visibility === "public",
  ```
- 现象：后端 `GetPublicNote` 要求 `is_published = true AND visibility IN ('public','unlisted')`（`note.go:1051`），而前端把 `unlisted` 的 `is_published` 置为 false → Unlisted 笔记既不进公开列表、也无法按链接打开，等于不可见。

## 🟡 低严重度

### 6. 标题自动保存后永远处于"未保存"状态
- 位置：`frontend/src/pages/Editor.tsx:457-485`
- 现象：`handlePageDetailsChange("title", ...)` 每次击键发 PATCH 但**不更新 `lastSavedRef.current`**，`isDirtyRef`（`:196-199`）永久为 true，之后每次导航/刷新都弹"有未保存更改"。
- 附带：标题击键的全量 PATCH 与手动 Save 并发时可能乱序互相覆盖（无版本号，last-write-wins）。

### 7. 编辑器只加载前 50 条笔记
- 位置：`frontend/src/pages/Editor.tsx:208`：`useListNotes({ limit: 50, offset: 0 })` 无分页。笔记 >50 条时侧栏看不到，`?noteId=` 也打不开第 1 页之后的笔记。

### 8. `ListNotes` 负 offset 未钳制
- 位置：`note.go:60-68`。负数 offset 在 Postgres 上直接 500；`ListPublicNotes`（`:958-961`）有 `offset < 0 → 0`，两处不一致。

### 9. `Register` 用户名无校验
- 位置：`auth.go:33-59`。`binding:"required"` 挡不住 `"   "`；超长用户名报错时返回误导信息 "username or email already exists"。用户名进入公开 URL `/:username/:slug`，含空格/特殊字符时 URL 表现异常。

### 10. 前后端密码最小长度不一致
- 位置：前端 `AuthModal.tsx:104` 要求 ≥8，后端 `auth.go:30` 要求 ≥6，绕过前端可用 6 位密码注册。

### 11. 后端 `convertMarkdownToHTML` 不转义 HTML
- 位置：`note.go:285-313`。`content_html` 原样存库并随 API 下发，一旦有消费者用 `dangerouslySetInnerHTML` 渲染即存储型 XSS。当前前端用 ReactMarkdown（默认转义）渲染 `content_md`，尚未被利用。

## 建议修复顺序
1. #1（数据损坏）、#2（越权 + 阻止他人删文件夹）——优先
2. #5（功能 bug）
3. #3、#4（隐患，修复成本低）
4. 其余按需
