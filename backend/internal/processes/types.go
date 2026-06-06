package processes

import "port-master/backend/internal/ports"

type ProcessInfo struct {
	PID           int64   `json:"pid"`
	ProcessName   string  `json:"processName"`
	CommandLine   string  `json:"commandLine"`
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryPercent float32 `json:"memoryPercent"`
	MemoryUsage   string  `json:"memoryUsage"`
	PortCount     int     `json:"portCount"`
}

type ProcessDetail struct {
	PID           int64            `json:"pid"`
	ProcessName   string           `json:"processName"`
	ProgramPath   string           `json:"programPath"`
	CommandLine   string           `json:"commandLine"`
	CPUPercent    float64          `json:"cpuPercent"`
	MemoryPercent float32          `json:"memoryPercent"`
	MemoryUsage   string           `json:"memoryUsage"`
	CreateTime    string           `json:"createTime"`
	BoundPorts    []ports.PortInfo `json:"boundPorts"`
}

type KillProcessRequest struct {
	PIDs  []int64 `json:"pids"`
	Force bool    `json:"force"`
}
