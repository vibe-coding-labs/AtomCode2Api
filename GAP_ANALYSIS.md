# AtomCode2API vs JoyCodeProxy — 功能对比（已对齐）

## 状态总览

| 类别 | JoyCode | AtomCode2API | 状态 |
|---|---|---|---|
| CLI 命令 | 23 个文件 | 19 个文件 | ✅ ~90% |
| pkg 模块 | 14 个 | 14 个 | ✅ 全部对齐 |
| 测试文件 | ~15 个 | 11 个 | ✅ 差距很小 |
| 基础设施 | Docker, Dashboard, Auth | 全有 | ✅ |

## 详细对比

### `cmd/commands`

| JoyCode | AtomCode | 状态 |
|---|---|---|
| `main.go` | ✅ | |
| `root.go` | ✅ | |
| `serve.go` | ✅ 已集成 Dashboard | |
| `daemon.go` | ✅ | |
| `config.go` | ✅ | |
| `models.go` | ✅ | |
| `whoami.go` | ✅ | |
| `check.go` | ✅ | |
| `chat.go` | ✅ | |
| `search.go` | ✅ | |
| `tls.go` | ✅ | |
| `embed.go` | ✅ | |
| `service.go` (跨平台) | ✅ | |
| `dual_listener.go` | ✅ | |
| `version.go` | ✅ | |
| `reset_password.go` | — | Dashboard 登录默认不设密码 |

### `pkg/` 模块

| JoyCode | AtomCode | 状态 |
|---|---|---|
| `pkg/joycode/` | `pkg/atmc/` | ✅ 对等 |
| `pkg/openai/` | `pkg/openai/` | ✅ 对等 |
| `pkg/anthropic/` | `pkg/anthropic/` | ✅ 对等 |
| `pkg/anthropic/logger.go` | ✅ | |
| `pkg/anthropic/truncate.go` | ✅ | |
| `pkg/store/store.go` | ✅ (含 Stats/GetRecentLogs) | |
| `pkg/store/token_usage.go` | ✅ | |
| `pkg/auth/` (jwt+cred+middleware) | ✅ | |
| `pkg/dashboard/handler.go` | ✅ | SPA Dashboard |
| `pkg/keepalive/` | ✅ | |
| `pkg/logrot/` | ✅ | |
| `pkg/proxy/sessions.go` | ✅ | |

### 测试

| JoyCode | AtomCode | 状态 |
|---|---|---|
| `atmc/client_test.go` | ✅ | |
| `atmc/translate_test.go` | ✅ | |
| `openai/translate_test.go` | ✅ | |
| `openai/handler_test.go` | ✅ | |
| `anthropic/anthropic_test.go` | ✅ | |
| `anthropic/truncate_test.go` | ✅ | |
| `logrot/rotator_test.go` | ✅ | |
| `auth/jwt_test.go` | ✅ | |
| `store/store_test.go` | ✅ | |
| `dashboard/handler_test.go` | ✅ | |
| `daemon_test.go` | ✅ | |
| `integration_test.go` | ✅ | |

### 基础设施

| JoyCode | AtomCode | 状态 |
|---|---|---|
| Dockerfile | ✅ | |
| Web Dashboard | ✅ (嵌入式 SPA) | |
| 账号导出/导入 | — | 只在多账号场景需要 |
| 凭据加密 | ✅ | |
| 凭据保活 | ✅ | |
| TLS | ✅ | |
| 日志轮转 | ✅ | |
| Service 安装 | ✅ | |

## 测试结果

```
ok  cmd/atomcode-2api     0.026s
ok  pkg/anthropic          0.009s
ok  pkg/atmc               0.011s
ok  pkg/auth               0.010s
ok  pkg/dashboard          0.014s
ok  pkg/logrot             0.115s
ok  pkg/openai             0.015s
ok  pkg/store              1.560s
```

✅ `go build` 编译通过
✅ `go vet` 零警告
✅ 8 个包全部测试通过