# GAP Analysis: AtomCode2Api vs JoyCode2Api (Reference Project)

> 生成日期: 2026-07-26 | 最后更新: 2026-07-26
> 目标: 确保 AtomCode2Api 拥有 JoyCode2Api 的全部能力，不打折

---

## 架构差异说明

两个项目底层架构不同，因此某些 "缺失" 是架构差异而非功能缺失：

| 维度 | JoyCode2Api | AtomCode2Api |
|------|-------------|--------------|
| 上游 | JD 云端 API (joycode-api.jd.com) | 本地 AtomCode Daemon (localhost:13456) |
| 认证 | ptKey + HMAC + color gateway | 本地 daemon 令牌 |
| 客户端包 | `pkg/joycode/` | `pkg/atmc/` |

`joycode/client.go` 28 个方法中有 19 个是 JD 云 API 特有（签名、网关、gzip 等），在本地 daemon 架构中不存在也不需要。

---

## 功能对齐状态

### 后端 API 端点 — ✅ 全部对齐（还多了 3 个）

| 类别 | JoyCode2Api | AtomCode2Api | 状态 |
|------|:-----------:|:------------:|:----:|
| 账号管理 (9 个) | ✅ | ✅ | 对齐 |
| 设置 (2 个) | ✅ | ✅ | 对齐 |
| 统计 (2 个) | ✅ | ✅ | 对齐 |
| 认证 (4 个) | ✅ | ✅ | 对齐 |
| 登录 (4 个) | ✅ | ✅ | 对齐 |
| 模型 (1 个) | ✅ | ✅ | 对齐 |
| 其他 (2 个) | ✅ | ✅ | 对齐 |
| **模型目录** | ❌ | ✅ **独有** | 增强 |
| **套餐状态** | ❌ | ✅ **独有** | 增强 |
| **批量导入** | ❌ | ✅ **独有** | 增强 |

### 前端页面路由 — ✅ 全部对齐

| 路由 | JoyCode2Api | AtomCode2Api |
|------|:-----------:|:------------:|
| /setup, /login, /forgot-password, /oauth-error | ✅ | ✅ |
| /dashboard, /accounts, /accounts/:userId, /settings | ✅ | ✅ |
| / (OAuth callback) | ✅ | ✅ |

### 前端页面功能 — ✅ 全部对齐

| 页面 | 功能点 | 状态 |
|------|--------|:----:|
| Dashboard | 请求统计、图表、Token 用量、模型/端点分布 | ✅ |
| AccountDetail | 账号信息、一键复制命令、模型选择、CodingPlan、日志、图表、Token 统计、活跃会话、上次活跃、模型目录+定价 | ✅ |
| Accounts | 列表、一键导入、OAuth、批量导入导出、手动添加、备注修改、引导页、注册按钮 | ✅ |
| Settings | 配置项管理、修改密码 | ✅ |

### 后端组件 — ✅ 全部对齐

所有 Go 源文件（cmd/ 23 个 + pkg/ 20 个）已全部对齐，AtomCode2Api 还多了 `setup.go` 和 `omni.go`。

### 测试文件 — 🟡 部分对齐

差异的测试文件均为 JoyCode 特有（JD 云 API 相关），不影响核心功能。

### 文档与脚本 — ✅ 全部对齐

| 项目 | 状态 |
|------|:----:|
| README.md, Dockerfile | ✅ |
| docs/guides/AGENTS.md, docs/install/*.md | ✅ |
| scripts/build.sh, scripts/release.sh | ✅ |

---

## 结论

**核心功能对齐率: 100%**

所有用户可见的功能（API 端点、前端页面、后端逻辑）已全部对齐。差异点：
1. 架构性差异（JD 云 API vs 本地 daemon）— 不是功能缺失
2. 部分 JoyCode 特有的测试文件 — 不影响功能
3. AtomCode2Api 有 3 个增强功能（模型目录、套餐状态、批量导入）
4. AtomCode2Api 有 2 个独有文件（setup.go, omni.go）

**不需要再补什么了。**
