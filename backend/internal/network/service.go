package network

import (
	"context"
	"net"
	"regexp"
	"sort"
	"strings"

	"port-master/backend/internal/executil"
)

type Interface struct {
	Name       string `json:"name"`
	IPAddress  string `json:"ipAddress"`
	MACAddress string `json:"macAddress"`
	Status     string `json:"status"`
	Type       string `json:"type"`
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ListInterfaces(ctx context.Context) ([]Interface, error) {
	result, err := listViaNetInterfaces()
	if err == nil && len(result) > 0 {
		sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
		return result, nil
	}
	return listViaCommand(ctx), nil
}

func listViaNetInterfaces() ([]Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	result := make([]Interface, 0)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		mac := "-"
		if len(iface.HardwareAddr) > 0 {
			mac = iface.HardwareAddr.String()
		}
		status := "DOWN"
		if iface.Flags&net.FlagUp != 0 {
			status = "UP"
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		hasAddr := false
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
				continue
			}
			hasAddr = true
			result = append(result, Interface{
				Name:       iface.Name,
				IPAddress:  ipNet.IP.String(),
				MACAddress: mac,
				Status:     status,
				Type:       iface.Name,
			})
		}
		if !hasAddr && status == "UP" {
			result = append(result, Interface{
				Name:       iface.Name,
				IPAddress:  "-",
				MACAddress: mac,
				Status:     status,
				Type:       iface.Name,
			})
		}
	}
	return result, nil
}

func listViaCommand(ctx context.Context) []Interface {
	if executil.IsCommandAvailable("ip") {
		if result := parseLinuxIP(ctx); len(result) > 0 {
			return result
		}
	}
	if executil.IsCommandAvailable("ipconfig") {
		return parseWindowsIPConfig(ctx)
	}
	return []Interface{}
}

func parseLinuxIP(ctx context.Context) []Interface {
	lines, err := executil.Run(ctx, "ip", "-4", "addr", "show")
	if err != nil {
		return nil
	}
	result := make([]Interface, 0)
	currentName := ""
	ifacePattern := regexp.MustCompile(`^\d+:\s*([^:@]+)`)
	inetPattern := regexp.MustCompile(`inet (\S+)`)
	for _, line := range lines {
		if m := ifacePattern.FindStringSubmatch(line); len(m) > 1 {
			currentName = strings.TrimSpace(m[1])
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "inet ") && currentName != "" {
			if m := inetPattern.FindStringSubmatch(trimmed); len(m) > 1 {
				ip := strings.Split(m[1], "/")[0]
				result = append(result, Interface{
					Name:       currentName,
					IPAddress:  ip,
					MACAddress: "-",
					Status:     "UP",
					Type:       "network",
				})
			}
		}
	}
	return result
}

func parseWindowsIPConfig(ctx context.Context) []Interface {
	lines, err := executil.Run(ctx, "ipconfig")
	if err != nil {
		return nil
	}
	result := make([]Interface, 0)
	currentName := ""
	ipv4Pattern := regexp.MustCompile(`: ([\d.]+)`)
	for _, line := range lines {
		if strings.HasSuffix(strings.TrimSpace(line), ":") && !strings.HasPrefix(line, " ") {
			currentName = strings.TrimSuffix(strings.TrimSpace(line), ":")
		} else if strings.Contains(line, "IPv4") && currentName != "" {
			if m := ipv4Pattern.FindStringSubmatch(line); len(m) > 1 {
				result = append(result, Interface{
					Name:       currentName,
					IPAddress:  m[1],
					MACAddress: "-",
					Status:     "UP",
					Type:       "network",
				})
			}
		}
	}
	return result
}
