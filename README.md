# Port Master - 端口与进程管理工具

[English](README_EN.md) | **简体中文**

Port Master 是一个跨平台的本机端口与进程管理工具。当前版本 **v2.1.0** 使用 Go 后端 + Vue 3 前端，发布时可构建成单个 Go 可执行文件，前端静态资源会嵌入二进制中。

本项目参考了此前 Java 版本项目 [MMCISAGOODMAN/port-master](https://github.com/MMCISAGOODMAN/port-master) 的功能设计与交互思路，并在此基础上使用 Go + Vue 重新实现。

## 界面预览

![Port Master 主界面](docs/images/port-master-screenshot.png)

## 功能（v2.1.0）

### 端口与探测

- 全量扫描本机 TCP/UDP 端口，支持 `refresh=true` 强制刷新与可配置扫描缓存 TTL。
- 按端口、端口表达式、范围、进程名、PID 查询；冲突检测、空闲端口、扫描对比。
- TCP / HTTP 健康探测 / TLS 证书探测（含过期天数）。
- 端口监控：浏览器本地配置 + 服务端后台轮询，WebSocket 推送告警。

### 远程与基础设施

- **SSH 远程主机**：test / info / scan / kill，支持密码与私钥；收藏主机仅存浏览器（不含凭据）。
- **Docker**：检测可用性、容器列表与端口映射、stop / restart。
- **Kubernetes**：kubectl 可用性、当前 context、Pods / Services / summary，namespace 可选。
- **网络接口**：本机网卡 IP / MAC / 状态列表。

### 界面与数据

- 扫描历史快照与趋势、远程主机收藏、分组 / 导出 / 主题。
- **zh-CN / en** 国际化（含认证界面）；设置中切换语言。
- 默认启用 **token 认证**；WebSocket 通过 `?token=` 鉴权，与 REST API 一致。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端 | Go 1.20+, chi, gopsutil, golang.org/x/crypto/ssh, gorilla/websocket, embed |
| 前端 | Vue 3, Vite, Element Plus, vue-i18n, Axios |
| 存储 | 无数据库，用户配置保存在浏览器 LocalStorage |
| 发布 | 单个 Go 可执行文件，内嵌 `backend/internal/web/dist` |
| 平台 | Windows, Linux, macOS |

## 快速开始

### 开发模式

```bash
cd backend
go run ./cmd/port-master --token dev-token
```

```bash
cd frontend
npm ci
npm run dev
```

前端开发地址为 `http://localhost:5173`，API 代理到 `http://localhost:8080`。登录时输入后端 token。

### 单文件构建

```bash
cd frontend
npm ci
npm run build

cd ../backend
go build -o port-master ./cmd/port-master
./port-master
```

Windows：

```powershell
cd backend
go build -o port-master.exe ./cmd/port-master
.\port-master.exe
```

默认监听 `127.0.0.1:8080`。外部访问示例：

```bash
./port-master --host 0.0.0.0 --port 8080 --token your-token
```

## 认证

默认启用认证。

| 方式 | 说明 |
| --- | --- |
| `--token your-token` | 固定 token |
| `PORT_MASTER_TOKEN` | 环境变量指定 token |
| 未指定 token | 启动时生成一次性 token 并打印 |
| `--no-auth` | 关闭认证 |

REST API（除 `/api/auth/*`）：

```http
Authorization: Bearer your-token
```

WebSocket 监控（`/ws/monitor`）在启用认证时需携带 token：

```
ws://host/ws/monitor?token=your-token
```

## 配置（服务端）

可通过 **CLI 参数** 或 **环境变量** 配置（启动时校验边界；`scan-cache-ttl-ms=0` 禁用缓存；监控轮询与 SSH 超时必须为正且有上限）：

| CLI 参数 | 环境变量 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--scan-cache-ttl-ms` | `PORT_MASTER_SCAN_CACHE_TTL_MS` | 3000 | 扫描结果缓存 TTL（毫秒），0 表示不缓存 |
| `--monitor-poll-ms` | `PORT_MASTER_MONITOR_POLL_MS` | 5000 | 后台监控轮询间隔（毫秒），最小 1000 |
| `--ssh-connect-timeout-ms` | `PORT_MASTER_SSH_CONNECT_TIMEOUT_MS` | 10000 | SSH TCP 连接与握手超时（毫秒） |
| `--ssh-command-timeout-sec` | `PORT_MASTER_SSH_COMMAND_TIMEOUT_SEC` | 60 | SSH 远程命令执行超时（秒） |

示例：

```bash
./port-master --token dev --scan-cache-ttl-ms 0 --monitor-poll-ms 10000 \
  --ssh-connect-timeout-ms 15000 --ssh-command-timeout-sec 60
```

`GET /api/system/config`（需认证）返回当前生效值：

| 字段 | 说明 |
| --- | --- |
| `scanCacheTtlMs` | 扫描缓存 TTL |
| `monitorPollIntervalMs` | 后台监控轮询间隔 |
| `sshConnectTimeoutMs` | SSH 连接超时 |
| `sshCommandTimeoutSec` | SSH 命令超时 |
| `version` | 当前版本 |

扫描强制刷新：`GET /api/ports/scan?refresh=true`

开发模式下 Vite 将 `/api` 与 `/ws` 代理到后端，WebSocket 使用同源连接。

## SSH 主机密钥策略

远程 SSH 使用 `golang.org/x/crypto/ssh`，当前策略为 **接受未知主机密钥**（`InsecureIgnoreHostKey`），便于内网/测试环境快速连接。

- 仅连接你信任的主机；生产环境建议在跳板机或 VPN 内使用。
- 密码/私钥仅用于当次请求，**不会**写入日志、API 响应或浏览器 LocalStorage。
- 收藏主机仅保存 host / port / username / authType，**不保存** password 或 private key。

## 主要 API（均需认证，除 auth）

| 模块 | 路径 |
| --- | --- |
| 端口 | `GET /api/ports/scan`, `GET /api/ports/probe`, `GET /api/ports/probe/http`, `GET /api/ports/probe/tls`, `POST /api/ports/monitor` |
| 远程 | `POST /api/remote/test`, `/info`, `/scan`, `/kill` |
| Docker | `GET /api/docker/available`, `/containers`, `POST /api/docker/stop`, `/restart` |
| K8s | `GET /api/k8s/available`, `/context`, `/pods`, `/services`, `/summary` |
| 网络 | `GET /api/network/interfaces` |
| 监控 | `POST /api/monitor/config`, `GET /api/monitor/status` |
| 系统 | `GET /api/system/stats`, `/info`, `/config` |
| WebSocket | `GET /ws/monitor` |

统一响应：`{ "code": 200, "message": "success", "data": ... }`

## 权限说明

Port Master 不会主动提权。杀进程和读取完整路径受当前运行用户权限限制。

- Windows：建议管理员身份运行。
- Linux/macOS：需要时使用 root 或 sudo。

## License

MIT
