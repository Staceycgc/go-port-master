package remote

import (
	"regexp"
	"strconv"
	"strings"

	"port-master/backend/internal/ports"
)

var (
	winTCPPattern        = regexp.MustCompile(`(?i)^\s*TCP\s+(\S+):(\d+)\s+(\S+):(\S+)\s+(\S+)\s+(\d+)\s*$`)
	winUDPPattern        = regexp.MustCompile(`(?i)^\s*UDP\s+(\S+):(\d+)\s+(\S+):(\S+)\s+(\d+)\s*$`)
	linuxSSPattern       = regexp.MustCompile(`(?i)^(tcp|udp)\s+(\S+)\s+\d+\s+\d+\s+(\S+)\s+(\S+)`)
	lsofListenPattern    = regexp.MustCompile(`(?i)(TCP|UDP)\s+(\S+):(\d+|\*)(?:\s+\(([^)]+)\))?`)
	lsofConnectedPattern = regexp.MustCompile(`(?i)(TCP|UDP)\s+(\S+):(\d+)\s*->\s*(\S+):(\S+)(?:\s+\(([^)]+)\))?`)
	macosNetstat         = regexp.MustCompile(`(?i)^(tcp\d*|udp\d*)\s+\d+\s+\d+\s+(\S+)\.(\d+|\*)\s+(\S+)\.(\S+)\s*(\S+)?`)
	linuxNetstatTCP      = regexp.MustCompile(`(?i)^tcp\d*\s+\d+\s+\d+\s+(\S+):(\d+)\s+(\S+):(\S+)\s+(\S+)`)
	linuxNetstatUDP      = regexp.MustCompile(`(?i)^udp\d*\s+\d+\s+\d+\s+(\S+):(\d+)\s+(\S+):(\S+)\s+(\d+/\S+)`)
	pidPattern           = regexp.MustCompile(`pid=(\d+)`)
	processPattern       = regexp.MustCompile(`"([^"]+)"`)
)

func ParseRemoteOutput(osType string, lines []string) []ports.PortInfo {
	if strings.EqualFold(osType, "windows") {
		return parseWindowsNetstat(lines)
	}
	return parseUnixOutput(lines)
}

func parseWindowsNetstat(lines []string) []ports.PortInfo {
	result := make([]ports.PortInfo, 0)
	seen := map[string]struct{}{}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if m := winTCPPattern.FindStringSubmatch(line); len(m) == 7 {
			appendWindowsPort(&result, seen, m[1], m[2], m[3], m[4], m[5], m[6], "TCP")
			continue
		}
		if m := winUDPPattern.FindStringSubmatch(line); len(m) == 6 {
			appendWindowsPort(&result, seen, m[1], m[2], m[3], m[4], "*", m[5], "UDP")
		}
	}
	return result
}

func appendWindowsPort(result *[]ports.PortInfo, seen map[string]struct{}, localHost, portStr, foreignHost, foreignPort, state, pidStr, protocol string) {
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return
	}
	pid, _ := strconv.ParseInt(pidStr, 10, 64)
	if state == "*" {
		state = "LISTEN"
	}
	key := protocol + ":" + strconv.Itoa(port) + ":" + localHost + ":" + pidStr
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	entry := ports.PortInfo{
		Protocol:       strings.ToUpper(protocol),
		Port:           port,
		LocalAddress:   joinHostPort(localHost, portStr),
		ForeignAddress: joinHostPort(foreignHost, foreignPort),
		State:          strings.ToUpper(state),
		ProcessName:    "-",
		ProgramPath:    "-",
	}
	if pid > 0 {
		entry.PID = &pid
	}
	*result = append(*result, entry)
}

func parseUnixOutput(lines []string) []ports.PortInfo {
	result := make([]ports.PortInfo, 0)
	seen := map[string]struct{}{}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || isHeaderLine(line) {
			continue
		}
		if entry, ok := parseLsofLine(line); ok {
			addUnixEntry(&result, seen, entry)
			continue
		}
		if entry, ok := parseMacOSNetstatLine(line); ok {
			addUnixEntry(&result, seen, entry)
			continue
		}
		if entry, ok := parseLinuxNetstatLine(line); ok {
			addUnixEntry(&result, seen, entry)
			continue
		}
		if entry, ok := parseLinuxSSLine(line); ok {
			addUnixEntry(&result, seen, entry)
		}
	}
	return result
}

func isHeaderLine(line string) bool {
	upper := strings.ToUpper(line)
	return strings.HasPrefix(upper, "NETID") ||
		strings.HasPrefix(upper, "PROTO") ||
		strings.HasPrefix(upper, "ACTIVE") ||
		strings.HasPrefix(upper, "STATE") ||
		strings.HasPrefix(upper, "COMMAND")
}

func parseLsofLine(line string) (ports.PortInfo, bool) {
	if entry, ok := parseLsofConnectedLine(line); ok {
		return entry, true
	}
	return parseLsofListenLine(line)
}

func parseLsofConnectedLine(line string) (ports.PortInfo, bool) {
	m := lsofConnectedPattern.FindStringSubmatch(line)
	if m == nil {
		return ports.PortInfo{}, false
	}
	port := parsePortToken(m[3])
	if port == nil {
		return ports.PortInfo{}, false
	}
	state := "ESTABLISHED"
	if len(m) > 6 && m[6] != "" {
		state = strings.ToUpper(m[6])
	}
	localHost := normalizeLsofHost(m[2])
	foreignHost := normalizeLsofHost(m[4])
	foreignPort := m[5]
	return ports.PortInfo{
		Protocol:       strings.ToUpper(m[1]),
		Port:           *port,
		LocalAddress:   joinHostPort(localHost, m[3]),
		ForeignAddress: joinHostPort(foreignHost, foreignPort),
		State:          state,
		PID:            extractLsofPID(line),
		ProcessName:    firstNonEmpty(extractProcessName(line), extractLsofCommand(line), "-"),
		ProgramPath:    "-",
	}, true
}

func parseLsofListenLine(line string) (ports.PortInfo, bool) {
	m := lsofListenPattern.FindStringSubmatch(line)
	if m == nil {
		return ports.PortInfo{}, false
	}
	if strings.Contains(m[0], "->") {
		return ports.PortInfo{}, false
	}
	port := parsePortToken(m[3])
	if port == nil {
		return ports.PortInfo{}, false
	}
	state := "LISTEN"
	if len(m) > 4 && m[4] != "" {
		state = strings.ToUpper(m[4])
	}
	host := normalizeLsofHost(m[2])
	return ports.PortInfo{
		Protocol:       strings.ToUpper(m[1]),
		Port:           *port,
		LocalAddress:   joinHostPort(host, m[3]),
		ForeignAddress: "*:*",
		State:          state,
		PID:            extractLsofPID(line),
		ProcessName:    firstNonEmpty(extractProcessName(line), extractLsofCommand(line), "-"),
		ProgramPath:    "-",
	}, true
}

func normalizeLsofHost(host string) string {
	if host == "*" {
		return "0.0.0.0"
	}
	return host
}

func extractLsofPID(line string) *int64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}
	pid, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || pid <= 0 {
		return nil
	}
	return &pid
}

func extractLsofCommand(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func parseMacOSNetstatLine(line string) (ports.PortInfo, bool) {
	m := macosNetstat.FindStringSubmatch(line)
	if m == nil {
		return ports.PortInfo{}, false
	}
	protocol := "TCP"
	if strings.HasPrefix(strings.ToLower(m[1]), "udp") {
		protocol = "UDP"
	}
	port := parsePortToken(m[3])
	if port == nil {
		return ports.PortInfo{}, false
	}
	state := "LISTEN"
	if len(m) > 6 && m[6] != "" {
		state = strings.ToUpper(m[6])
	} else if protocol == "UDP" {
		state = "UNCONN"
	}
	return ports.PortInfo{
		Protocol:       protocol,
		Port:           *port,
		LocalAddress:   joinMacNetstatHostPort(m[2], m[3]),
		ForeignAddress: joinMacNetstatHostPort(m[4], m[5]),
		State:          state,
		ProcessName:    "-",
		ProgramPath:    "-",
	}, true
}

func joinMacNetstatHostPort(host, portToken string) string {
	if portToken == "*" {
		return host + ".*"
	}
	return host + "." + portToken
}

func parseLinuxNetstatLine(line string) (ports.PortInfo, bool) {
	if entry, ok := parseLinuxNetstatTCP(line); ok {
		return entry, true
	}
	return parseLinuxNetstatUDP(line)
}

func parseLinuxNetstatTCP(line string) (ports.PortInfo, bool) {
	m := linuxNetstatTCP.FindStringSubmatch(line)
	if m == nil {
		return ports.PortInfo{}, false
	}
	port, err := strconv.Atoi(m[2])
	if err != nil || port < 1 || port > 65535 {
		return ports.PortInfo{}, false
	}
	return ports.PortInfo{
		Protocol:       "TCP",
		Port:           port,
		LocalAddress:   m[1] + ":" + m[2],
		ForeignAddress: m[3] + ":" + m[4],
		State:          strings.ToUpper(m[5]),
		ProcessName:    "-",
		ProgramPath:    "-",
	}, true
}

func parseLinuxNetstatUDP(line string) (ports.PortInfo, bool) {
	m := linuxNetstatUDP.FindStringSubmatch(line)
	if m == nil {
		return ports.PortInfo{}, false
	}
	port, err := strconv.Atoi(m[2])
	if err != nil || port < 1 || port > 65535 {
		return ports.PortInfo{}, false
	}
	pidProgram := m[5]
	processName := "-"
	var pid *int64
	if parts := strings.SplitN(pidProgram, "/", 2); len(parts) == 2 {
		if p, err := strconv.ParseInt(parts[0], 10, 64); err == nil && p > 0 {
			pid = &p
		}
		processName = parts[1]
	}
	return ports.PortInfo{
		Protocol:       "UDP",
		Port:           port,
		LocalAddress:   m[1] + ":" + m[2],
		ForeignAddress: m[3] + ":" + m[4],
		State:          "UNCONN",
		PID:            pid,
		ProcessName:    processName,
		ProgramPath:    "-",
	}, true
}

func parseLinuxSSLine(line string) (ports.PortInfo, bool) {
	m := linuxSSPattern.FindStringSubmatch(line)
	if m == nil {
		return ports.PortInfo{}, false
	}
	protocol := strings.ToUpper(m[1])
	local := m[3]
	foreign := m[4]
	port := extractPortFromAddr(local)
	if port == nil {
		return ports.PortInfo{}, false
	}
	state := strings.ToUpper(m[2])
	return ports.PortInfo{
		Protocol:       protocol,
		Port:           *port,
		LocalAddress:   local,
		ForeignAddress: foreign,
		State:          state,
		PID:            extractPID(line),
		ProcessName:    firstNonEmpty(extractProcessName(line), "-"),
		ProgramPath:    "-",
	}, true
}

func addUnixEntry(result *[]ports.PortInfo, seen map[string]struct{}, entry ports.PortInfo) {
	key := entry.Protocol + ":" + strconv.Itoa(entry.Port) + ":" + entry.LocalAddress + ":" + entry.State
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*result = append(*result, entry)
}

func extractPortFromAddr(addr string) *int {
	if strings.HasPrefix(addr, "[") {
		host, portStr, err := netSplitHostPortBracket(addr)
		if err != nil {
			return nil
		}
		_ = host
		return parsePortToken(portStr)
	}
	parts := strings.Split(addr, ":")
	if len(parts) < 2 {
		return nil
	}
	return parsePortToken(parts[len(parts)-1])
}

func netSplitHostPortBracket(addr string) (string, string, error) {
	if !strings.HasPrefix(addr, "[") {
		return "", "", strconv.ErrSyntax
	}
	end := strings.Index(addr, "]")
	if end < 0 {
		return "", "", strconv.ErrSyntax
	}
	host := addr[1:end]
	rest := addr[end+1:]
	if !strings.HasPrefix(rest, ":") {
		return "", "", strconv.ErrSyntax
	}
	return host, rest[1:], nil
}

func parsePortToken(token string) *int {
	if token == "" || token == "*" {
		return nil
	}
	port, err := strconv.Atoi(token)
	if err != nil || port < 1 || port > 65535 {
		return nil
	}
	return &port
}

func joinHostPort(host, port string) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return "[" + host + "]:" + port
	}
	return host + ":" + port
}

func extractPID(line string) *int64 {
	if m := pidPattern.FindStringSubmatch(line); len(m) > 1 {
		pid, _ := strconv.ParseInt(m[1], 10, 64)
		return &pid
	}
	return nil
}

func extractProcessName(line string) string {
	if m := processPattern.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
