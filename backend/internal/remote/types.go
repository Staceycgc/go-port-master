package remote

type HostRequest struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Credential string `json:"credential"`
	AuthType   string `json:"authType"`
}

type KillRequest struct {
	HostRequest
	PID   int64 `json:"pid"`
	Force bool  `json:"force"`
}
