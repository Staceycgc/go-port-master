package remote

import "testing"

func TestParseLsofConnectedUsesLocalPort(t *testing.T) {
	line := "curl  5678 user   10u  IPv4 0x0 0t0  TCP 127.0.0.1:1234->1.2.3.4:80 (ESTABLISHED)"
	entry, ok := parseLsofLine(line)
	if !ok {
		t.Fatal("expected connected lsof line to parse")
	}
	if entry.Port != 1234 {
		t.Fatalf("expected local port 1234, got %d", entry.Port)
	}
	if entry.LocalAddress != "127.0.0.1:1234" {
		t.Fatalf("unexpected local address: %q", entry.LocalAddress)
	}
	if entry.ForeignAddress != "1.2.3.4:80" {
		t.Fatalf("unexpected foreign address: %q", entry.ForeignAddress)
	}
	if entry.State != "ESTABLISHED" {
		t.Fatalf("unexpected state: %q", entry.State)
	}
}

func TestParseMacOSNetstatForeignAddress(t *testing.T) {
	line := "tcp4       0      0  192.168.1.10.8080      10.0.0.5.443           ESTABLISHED"
	entry, ok := parseMacOSNetstatLine(line)
	if !ok {
		t.Fatal("expected macOS netstat line to parse")
	}
	if entry.Port != 8080 {
		t.Fatalf("expected port 8080, got %d", entry.Port)
	}
	if entry.LocalAddress != "192.168.1.10.8080" {
		t.Fatalf("unexpected local address: %q", entry.LocalAddress)
	}
	if entry.ForeignAddress != "10.0.0.5.443" {
		t.Fatalf("unexpected foreign address: %q", entry.ForeignAddress)
	}
	if entry.State != "ESTABLISHED" {
		t.Fatalf("unexpected state: %q", entry.State)
	}
}

func TestParseLinuxNetstatUDP(t *testing.T) {
	line := "udp        0      0 0.0.0.0:5353            0.0.0.0:*                           4688/avahi-daemon"
	entry, ok := parseLinuxNetstatUDP(line)
	if !ok {
		t.Fatal("expected linux udp netstat line to parse")
	}
	if entry.Port != 5353 {
		t.Fatalf("expected port 5353, got %d", entry.Port)
	}
	if entry.State != "UNCONN" {
		t.Fatalf("expected UNCONN, got %q", entry.State)
	}
	if entry.LocalAddress != "0.0.0.0:5353" {
		t.Fatalf("unexpected local address: %q", entry.LocalAddress)
	}
	if entry.ForeignAddress != "0.0.0.0:*" {
		t.Fatalf("unexpected foreign address: %q", entry.ForeignAddress)
	}
	if entry.ProcessName != "avahi-daemon" {
		t.Fatalf("unexpected process name: %q", entry.ProcessName)
	}
	if entry.PID == nil || *entry.PID != 4688 {
		t.Fatalf("unexpected pid: %#v", entry.PID)
	}
}

func TestParseWindowsTCPAndUDP(t *testing.T) {
	lines := []string{
		"  TCP    0.0.0.0:8080           0.0.0.0:0              LISTENING       4321",
		"  UDP    0.0.0.0:5353           *:*                                    4688",
	}
	result := parseWindowsNetstat(lines)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %#v", result)
	}
	if result[0].Port != 8080 || result[0].State != "LISTENING" {
		t.Fatalf("unexpected tcp entry: %#v", result[0])
	}
	if result[1].Port != 5353 || result[1].Protocol != "UDP" {
		t.Fatalf("unexpected udp entry: %#v", result[1])
	}
}

func TestValidateRequest(t *testing.T) {
	_, _, _, err := validateRequest(HostRequest{Host: "127.0.0.1", Username: "root", AuthType: "password", Credential: "secret"})
	if err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	_, _, _, err = validateRequest(HostRequest{Host: "127.0.0.1", Username: "root", AuthType: "password"})
	if err == nil {
		t.Fatal("expected missing password error")
	}
}
