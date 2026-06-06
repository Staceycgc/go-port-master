//go:build !windows

package system

func newCPUUsageProvider() cpuUsageProvider {
	return nil
}
