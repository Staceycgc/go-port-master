package system

type Stats struct {
	CPUUsage              float64 `json:"cpuUsage"`
	MemoryUsage           float64 `json:"memoryUsage"`
	MemoryUsedMB          float64 `json:"memoryUsedMb"`
	MemoryTotalMB         float64 `json:"memoryTotalMb"`
	ListenPortCount       int     `json:"listenPortCount"`
	ActiveConnectionCount int     `json:"activeConnectionCount"`
	ProcessCount          int     `json:"processCount"`
	OSType                string  `json:"osType"`
	NeedAdminHint         bool    `json:"needAdminHint"`
}
