# AtomCodeProxy vs JoyCodeProxy — 完整功能对比

## 1. `cmd/` — CLI 命令层

| JoyCodeProxy | AtomCodeProxy | 状态 |
|---|---|---|
| `main.go` | ✅ | |
| `root.go` | ✅ | |
| `serve.go` | ✅ | 未集成 logrot、TLS、daemon 模式到 serve 的完整循环 |
| `daemon.go` | ✅ | supervisor + child 模式 |
| `config.go` | ✅ | |
| `models.go` | ✅ | |
| `whoami.go` | ✅ | |
| `check.go` | ✅ (合并到 whoami.go) | |
| `chat.go` | ❌ **缺失** | 发消息的 CLI 命令 |
| `search.go` | ❌ **缺失** | web‑search CLI |
| `reset_password.go` | ❌ **缺失** | Dashboard 密码重置 |
| `tls.go` | ✅ | |
| `embed.go` | ✅ | |
| `service_linux.go` / `service_darwin.go` | ✅ | |
| `service.go` | ❌ **缺失** | 跨平台 service install/uninstall |
| `dual_listener.go` | ❌ **缺失** | HTTP + HTTPS 双端口监听 |
| `version.go` | ✅ | |

## 2. `pkg/` — 核心包

| JoyCodeProxy | AtomCodeProxy | 状态 |
|---|---|---|
| `pkg/joycode/` — upstream API 客户端 | `pkg/atmc/` | ✅ 同等级 |
| `pkg/openai/` | `pkg/openai/` | ✅ 基本对等 |
| `pkg/anthropic/` | `pkg/anthropic/` | ✅ 基本对等 |
| `pkg/anthropic/truncate.go` | ✅ | |
| `pkg/anthropic/logger.go` | ❌ **缺失** | 请求日志中间件（anthropic 专用） |
| `pkg/store/store.go` | ✅ (精简版) | ⚠️ JoyCode 多了账号 CRUD、加密、统计、导出导入 |
| `pkg/store/token_usage.go` | ✅ | |
| `pkg/auth/jwt.go` | ✅ | |
| `pkg/auth/middleware.go` | ✅ | |
| `pkg/auth/credentials.go` | ✅ | |
| `pkg/auth/jdlogin.go` | ❌ **缺失** | JD 扫码登录 (对于 AtomCode 不需要) |
| `pkg/auth/password.go` | ❌ **缺失** | Dashboard 密码认证 |
| `pkg/dashboard/handler.go` | ❌ **缺失** | Dashboard API + 静态文件服务 |
| `pkg/keepalive/keepalive.go` | ✅ | |
| `pkg/logrot/rotator.go` | ✅ | |
| `pkg/proxy/sessions.go` | ✅ | |

## 3. 测试覆盖率

| JoyCodeProxy | AtomCodeProxy | 状态 |
|---|---|---|
| `pkg/atmc/client_test.go` | ✅ | |
| `pkg/atmc/translate_test.go` | ✅ | |
| `pkg/openai/translate_test.go` | ✅ | |
| `pkg/openai/handler_test.go` | ❌ **缺失** | OpenAI handler 测试 |
| `pkg/anthropic/truncate_test.go` | ✅ | |
| `pkg/anthropic/anthropic_test.go` | ❌ **缺失** | Anthropic 核心测试 |
| `pkg/logrot/rotator_test.go` | ✅ | |
| `pkg/store/store_test.go` | ❌ **缺失** | 存储层测试 |
| `cmd/.../integration_test.go` | ✅ (mock daemon) | |
| `cmd/.../daemon_test.go` | ❌ **缺失** | 守护进程测试 |
| `cmd/.../dashboard_test.go` | ❌ **缺失** | Dashboard 测试 |

## 4. 其他基础设施

| JoyCodeProxy | AtomCodeProxy | 状态 |
|---|---|---|
| Dockerfile | ✅ | |
| Web Dashboard (React) | ❌ **缺失** | 管理后台前端 |
| 账号导出/导入 | ❌ **缺失** | 迁移功能 |
| 凭据加密 (AES-GCM) | ✅ | |
| SSE 事件翻译 — 基础 (text/reasoning/tokens/done/error) | ✅ | |
| SSE 事件翻译 — 工具调用 (tool_start/output/result) | ✅ | |
| 上下文截断 (保留 tool 完整性) | ✅ | JoyCode 多了 tool_pair 保持逻辑 |

## 5. 核心功能评分

| 功能 | 完成度 | 备注 |
|---|---|---|
| OpenAI Chat Completions (流式+非流式) | 100% | 已验证 mock 集成测试 |
| Anthropic Messages API (流式+非流式) | 90% | 代码就绪，缺少集成测试 |
| 工具调用翻译 | 100% | text/reasoning/tool_start/tokens/done 全覆盖 |
| 多轮对话 session 追踪 | 100% | 30分钟过期自动清理 |
| Daemon 模式 (supervisor+child) | 100% | 指数退避重启 |
| 日志轮转 | 100% | 大小触发，保留 N 个 |
| JWT 认证 | 100% | 签发+验证+中间件 |
| 凭据加密 | 100% | AES-GCM |
| 凭据保活 | 100% | 周期性刷新 |
| TLS 自签名证书 | 100% | |
| Docker | 100% | |
| **Web Dashboard** | **0%** | 前端+API 均未实现 |
| **Anthropic handler 集成测试** | **⚠️ 缺失** | 需要 mock daemon 验证完整流 |
| **OpenAI handler 单元测试** | **⚠️ 缺失** | handler_test.go |
| **store_test.go** | **⚠️ 缺失** | 存储层测试 |