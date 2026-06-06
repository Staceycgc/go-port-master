# Port Master - Port & Process Management

**English** | [简体中文](README.md)

Port Master is a cross-platform local port and process management tool. This version uses a Go backend and a Vue 3 frontend. Production builds can be shipped as one Go executable with the Vue assets embedded.

This project references the feature design and interaction ideas from the earlier Java version, [MMCISAGOODMAN/port-master](https://github.com/MMCISAGOODMAN/port-master), and reimplements them with Go + Vue.

## Screenshot

![Port Master dashboard](docs/images/port-master-screenshot.png)

## Features

- Scan local TCP/UDP ports with protocol, port, local address, remote address, PID, process name, executable path, and connection state.
- Query by port, port expression, range, process name, or PID.
- Generate free ports, run TCP probes, monitor ports, detect conflicts, and view summary stats.
- List processes, view process details, and terminate by PID, port, or batch request.
- Keeps the existing Vue local experience: groups, monitoring, history, export, theme, and browser LocalStorage settings.
- Token protection is enabled by default. If no fixed token is provided, a one-time token is generated at startup and printed to the console.

## Stack

| Layer | Technology |
| --- | --- |
| Backend | Go, chi, gopsutil, embed |
| Frontend | Vue 3, Vite, Element Plus, Axios |
| Storage | No database; user settings live in browser LocalStorage |
| Distribution | Single Go executable with embedded Vue build |
| Platforms | Windows, Linux, macOS |

## Quick Start

### Development

```bash
cd backend
go run ./cmd/port-master --token dev-token
```

```bash
cd frontend
npm ci
npm run dev
```

The frontend dev server runs at `http://localhost:5173` and proxies API requests to `http://localhost:8080`. Log in with the backend token.

### Single-Binary Build

```bash
cd frontend
npm ci
npm run build

cd ../backend
go build -o port-master ./cmd/port-master
./port-master
```

Windows example:

```powershell
cd backend
go build -o port-master.exe ./cmd/port-master
.\port-master.exe
```

Default bind address is `127.0.0.1:8080`. To expose it on a server:

```bash
./port-master --host 0.0.0.0 --port 8080 --token your-token
```

## Authentication

Authentication is enabled by default.

- `--token your-token`: use a fixed token.
- `PORT_MASTER_TOKEN=your-token`: set a fixed token through the environment.
- No token provided: a one-time token is generated and printed at startup.
- `--no-auth`: explicitly disable authentication.
- `--host` / `--port` or `PORT_MASTER_HOST` / `PORT_MASTER_PORT`: change the bind address.

All `/api/*` routes except `/api/auth/*` require this header when auth is enabled:

```http
Authorization: Bearer your-token
```

## Permissions

Port Master does not elevate privileges. Killing processes and reading full executable paths depend on the permissions of the current user.

- Windows: run as Administrator for protected processes and more complete path information.
- Linux/macOS: use root or sudo when full permissions are needed.

## License

MIT
