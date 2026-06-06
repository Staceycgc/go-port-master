package ports

type PortInfo struct {
	Protocol       string `json:"protocol"`
	Port           int    `json:"port"`
	LocalAddress   string `json:"localAddress"`
	ForeignAddress string `json:"foreignAddress"`
	PID            *int64 `json:"pid"`
	ProcessName    string `json:"processName"`
	ProgramPath    string `json:"programPath"`
	State          string `json:"state"`
}

type PortConflict struct {
	Port         int      `json:"port"`
	Protocol     string   `json:"protocol"`
	PIDs         []int64  `json:"pids"`
	ProcessNames []string `json:"processNames"`
	Message      string   `json:"message"`
}

type FreePortResult struct {
	StartPort int    `json:"startPort"`
	Count     int    `json:"count"`
	FreePorts []int  `json:"freePorts"`
	Message   string `json:"message"`
}

type PortSummary struct {
	Total             int `json:"total"`
	TCPCount          int `json:"tcpCount"`
	UDPCount          int `json:"udpCount"`
	ListenCount       int `json:"listenCount"`
	EstablishedCount  int `json:"establishedCount"`
	UniquePortCount   int `json:"uniquePortCount"`
	UniquePIDCount    int `json:"uniquePidCount"`
	LocalhostCount    int `json:"localhostCount"`
	AllInterfaceCount int `json:"allInterfaceCount"`
}

type PortProbeResult struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	Reachable bool   `json:"reachable"`
	LatencyMs int64  `json:"latencyMs"`
	Message   string `json:"message"`
}

type PortMonitorRequest struct {
	Ports []MonitorPortItem `json:"ports"`
}

type MonitorPortItem struct {
	Port          int    `json:"port"`
	Protocol      string `json:"protocol"`
	Remark        string `json:"remark"`
	ExpectedState string `json:"expectedState"`
}

type PortMonitorResult struct {
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	Occupied    bool   `json:"occupied"`
	ProcessName string `json:"processName"`
	PID         *int64 `json:"pid"`
	State       string `json:"state"`
	Remark      string `json:"remark"`
}
