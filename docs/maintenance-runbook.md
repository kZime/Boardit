# Boardit 分阶段维护运行手册

**状态**：生效

**适用范围**：`refactoring-readiness-assessment.md` 中的 R0–R4

**基本原则**：保留现有技术栈和公开行为，小步修改，每阶段单独验证、单独本地提交。

## 1. 自主执行边界

在用户不在场时，可以自主执行以下操作：

- 阅读代码、文档、Git 历史和本地配置。
- 在已约定的 R0–R4 范围内修改代码、测试、CI 和迁移文件。
- 安装项目开发依赖，运行本地数据库和浏览器测试。
- 为每个完整阶段创建独立本地 commit。

出现以下情况必须停止执行并记录 blocker：

- 需要 push、部署、发布或向外部系统写入。
- 需要新凭据、越权或更改生产数据。
- 需要删除不可恢复数据或重写 Git 历史。
- 需要产品决策，例如已发布 slug 是否固定、tags 是否保留、refresh token 是否迁移到 Cookie。
- 无法在保留公开 API 或用户可见行为的前提下继续。

## 2. 阶段提交协议

| 阶段 | 内容 | 提交前最低证据 | 建议 commit |
|---|---|---|---|
| R0 | 测试保护网、可重复依赖、PR CI | 全部基线门禁 | `test: establish refactoring safety net` |
| R1 | 权限、并发、认证和交互 bug | 每个 bug 的回归测试 + 全部门禁 | `fix: enforce data and auth invariants` |
| R2 | 后端分层与事务边界 | use-case 无 Gin 测试 + 契约不变 | `refactor: modularize note backend` |
| R3 | 前端 Editor、Auth 和 API 解耦 | 组件/hook 测试 + E2E | `refactor: modularize editor frontend` |
| R4 | 版本化迁移、revision/outbox 边界 | 向前迁移、回滚说明、版本测试 | `refactor: add versioned persistence boundaries` |

不将两个阶段压入同一 commit。某一阶段门禁未通过时，不创建该阶段 commit，也不继续下一阶段。

## 3. 本地完整门禁

### 后端

```bash
cd backend
test -z "$(gofmt -l .)"
go vet ./...
go test ./...
```

### 前端

```bash
cd frontend
npm ci
npm run lint
npm test
npm run build
npm run orval
git diff --exit-code -- src/api/gen
npm run test:e2e
```

`npm ci` 为确定性安装标准；`package-lock.json` 是唯一锁文件。不使用会更改锁文件的安装方式代替 CI 门禁。

## 4. 提交前检查

1. 确认 `git diff --check` 无空白错误。
2. 确认变更只属于当前阶段，没有覆盖用户的无关修改。
3. 运行完整门禁，记录命令、结果和已知非阻断警告。
4. 用 `git diff --cached --stat` 和 `git diff --cached` 复核暂存内容。
5. 只创建本地 commit，不 push。

## 5. 失败处理

- 可重试的网络或包下载失败：保持代码不变，重试相同命令。
- 新回归测试揭示现存 bug：将它与“希望行为”测试一起放入 R1，R0 仅固化可依赖的当前行为。
- 公开契约漂移：先停止，判断是生成文件过期还是 API 无意变更。
- 门禁在当前阶段内无法修复：不提交，在交接中记录最小复现、根因和所需决策。
