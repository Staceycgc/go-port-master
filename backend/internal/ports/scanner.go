package ports

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type Scanner interface {
	ScanAllPorts(ctx context.Context) ([]PortInfo, error)
}

type GopsutilScanner struct{}

type processMeta struct {
	name string
	path string
}

func (GopsutilScanner) ScanAllPorts(ctx context.Context) ([]PortInfo, error) {
	connections, err := gnet.Connections("all")
	if err != nil {
		return nil, err
	}

	metaCache := map[int32]processMeta{}
	seen := map[string]struct{}{}
	result := make([]PortInfo, 0, len(connections))

	for _, conn := range connections {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		protocol := protocolName(conn.Type)
		if protocol == "" {
			continue
		}
		if conn.Laddr.Port == 0 {
			continue
		}

		state := normalizeState(conn.Status, protocol)
		local := formatAddr(conn.Laddr)
		foreign := formatAddr(conn.Raddr)
		if foreign == "" {
			foreign = "*:*"
		}

		var pid *int64
		name := ""
		path := ""
		if conn.Pid > 0 {
			value := int64(conn.Pid)
			pid = &value
			meta, ok := metaCache[conn.Pid]
			if !ok {
				meta = lookupProcessMeta(conn.Pid)
				metaCache[conn.Pid] = meta
			}
			name = meta.name
			path = meta.path
		}

		key := fmt.Sprintf("%s:%d:%s:%s:%s:%s", protocol, conn.Laddr.Port, local, foreign, state, pidKey(pid))
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		result = append(result, PortInfo{
			Protocol:       protocol,
			Port:           int(conn.Laddr.Port),
			LocalAddress:   local,
			ForeignAddress: foreign,
			PID:            pid,
			ProcessName:    name,
			ProgramPath:    path,
			State:          state,
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Port == result[j].Port {
			return result[i].Protocol < result[j].Protocol
		}
		return result[i].Port < result[j].Port
	})
	return result, nil
}

func protocolName(socketType uint32) string {
	switch int(socketType) {
	case syscall.SOCK_STREAM:
		return "TCP"
	case syscall.SOCK_DGRAM:
		return "UDP"
	default:
		return ""
	}
}

func normalizeState(state string, protocol string) string {
	state = strings.TrimSpace(strings.ToUpper(state))
	if state == "" {
		if protocol == "UDP" {
			return "LISTEN"
		}
		return "UNKNOWN"
	}
	if protocol == "UDP" && (state == "NONE" || state == "UNKNOWN") {
		return "LISTEN"
	}
	return state
}

func formatAddr(addr gnet.Addr) string {
	if addr.Port == 0 {
		return ""
	}
	port := strconv.Itoa(int(addr.Port))
	ip := strings.TrimSpace(addr.IP)
	if ip == "" || ip == "*" {
		return "*:" + port
	}
	if ip == "::" || ip == "[::]" {
		return "[::]:" + port
	}
	return net.JoinHostPort(ip, port)
}

func pidKey(pid *int64) string {
	if pid == nil {
		return "null"
	}
	return strconv.FormatInt(*pid, 10)
}

func lookupProcessMeta(pid int32) processMeta {
	proc, err := process.NewProcess(pid)
	if err != nil {
		return processMeta{}
	}
	name, _ := proc.Name()
	path, _ := proc.Exe()
	if path == "" {
		path, _ = proc.Cmdline()
	}
	if runtime.GOOS == "windows" && name == "" {
		name = path
	}
	return processMeta{name: name, path: path}
}
