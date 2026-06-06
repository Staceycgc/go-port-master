# Port Master - 端口与进程管理工具

[English](README_EN.md) | **简体中文**

Port Master 是一个跨平台的本机端口与进程管理工具。当前版本使用 Go 后端 + Vue 3 前端，发布时可构建成单个 Go 可执行文件，前端静态资源会嵌入二进制中。

本项目参考了此前 Java 版本项目 [MMCISAGOODMAN/port-master](https://github.com/MMCISAGOODMAN/port-master) 的功能设计与交互思路，并在此基础上使用 Go + Vue 重新实现。

## 界面预览

![Port Master 主界面](docs/images/port-master-screenshot.png)

## 功能

- 扫描本机 TCP/UDP 端口，查看协议、端口、本地地址、远端地址、PID、进程名、程序路径和连接状态。
- 按端口、端口表达式、端口范围、进程名、PID 查询。
- 生成空闲端口、TCP 连通性探测、端口监控、端口冲突检测和摘要统计。
- 查看进程列表和进程详情，支持按 PID、按端口、批量结束进程。
- Vue 前端保留分组、监控、历史、导出、主题等浏览器 LocalStorage 体验。
- 默认启用 token 保护；未传入固定 token 时，启动时生成一次性 token 并打印到控制台。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端 | Go, chi, gopsutil, embed |
| 前端 | Vue 3, Vite, Element Plus, Axios |
| 存储 | 无数据库，用户配置保存在浏览器 LocalStorage |
| 发布 | 单个 Go 可执行文件，内嵌 Vue 构建产物 |
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

前端开发地址为 `http://localhost:5173`，API 会代理到 `http://localhost:8080`。登录时输入后端 token。

### 单文件构建

```bash
cd frontend
npm ci
npm run build

cd ../backend
go build -o port-master ./cmd/port-master
./port-master
```

Windows 可执行文件示例：

```powershell
cd backend
go build -o port-master.exe ./cmd/port-master
.\port-master.exe
```

默认监听 `127.0.0.1:8080`。如需服务器外部访问：

```bash
./port-master --host 0.0.0.0 --port 8080 --token your-token
```

## 认证配置

默认启用认证。

- `--token your-token`：指定固定 token。
- `PORT_MASTER_TOKEN=your-token`：通过环境变量指定固定 token。
- 未指定 token：启动时生成一次性 token 并打印到控制台。
- `--no-auth`：显式关闭认证。
- `--host` / `--port` 或 `PORT_MASTER_HOST` / `PORT_MASTER_PORT`：修改监听地址。

除 `/api/auth/*` 外，所有 `/api/*` 请求在认证启用时都需要：

```http
Authorization: Bearer your-token
```


## 权限说明

Port Master 不会主动提权。杀进程和读取完整路径受当前运行用户权限限制。

- Windows：建议用管理员身份运行，以便结束受保护进程和读取更多路径信息。
- Linux/macOS：建议在需要完整权限时使用 root 或 sudo 运行。

## License

MIT
