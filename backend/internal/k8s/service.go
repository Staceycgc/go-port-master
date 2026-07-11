package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"port-master/backend/internal/executil"
)

var (
	ErrUnavailable   = errors.New("kubectl is not available")
	ErrCommandFailed = errors.New("kubectl command failed")
)

var namespacePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

type PodPort struct {
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol"`
	Name          string `json:"name"`
	HostPort      string `json:"hostPort,omitempty"`
}

type Pod struct {
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Node      string    `json:"node"`
	PodIP     string    `json:"podIp"`
	Ports     []PodPort `json:"ports"`
}

type TargetPort struct {
	Number *int   `json:"-"`
	Name   string `json:"-"`
}

func (t TargetPort) MarshalJSON() ([]byte, error) {
	if t.Name != "" {
		return json.Marshal(t.Name)
	}
	if t.Number != nil {
		return json.Marshal(*t.Number)
	}
	return json.Marshal(nil)
}

func (t TargetPort) Display() string {
	if t.Name != "" {
		return t.Name
	}
	if t.Number != nil {
		return strconv.Itoa(*t.Number)
	}
	return "-"
}

type ServicePort struct {
	Name       string     `json:"name"`
	Port       int        `json:"port"`
	TargetPort TargetPort `json:"targetPort"`
	NodePort   *int       `json:"nodePort,omitempty"`
	Protocol   string     `json:"protocol"`
}

type Service struct {
	Namespace string        `json:"namespace"`
	Name      string        `json:"name"`
	Type      string        `json:"type"`
	ClusterIP string        `json:"clusterIp"`
	Ports     []ServicePort `json:"ports"`
}

type Summary struct {
	Context      string `json:"context"`
	PodCount     int    `json:"podCount"`
	RunningPods  int64  `json:"runningPods"`
	ServiceCount int    `json:"serviceCount"`
}

type Client struct {
	runner executil.Runner
}

func NewClient() *Client {
	return NewClientWithRunner(executil.OSRunner{})
}

func NewClientWithRunner(runner executil.Runner) *Client {
	return &Client{runner: runner}
}

func (c *Client) Available() bool {
	return c.runner.Available("kubectl")
}

func (c *Client) CurrentContext(ctx context.Context) (string, error) {
	if !c.Available() {
		return "", ErrUnavailable
	}
	lines, err := c.runner.Run(ctx, "kubectl", "config", "current-context")
	if err != nil {
		return "", wrapCommandError(err)
	}
	if len(lines) == 0 {
		return "", nil
	}
	return strings.TrimSpace(lines[0]), nil
}

func (c *Client) ListPods(ctx context.Context, namespace string) ([]Pod, error) {
	if !c.Available() {
		return nil, ErrUnavailable
	}
	if err := validateNamespace(namespace); err != nil {
		return nil, err
	}
	args := []string{"get", "pods", "-o", "json"}
	if strings.TrimSpace(namespace) != "" {
		args = append(args, "-n", namespace)
	} else {
		args = append(args, "-A")
	}
	lines, err := c.runner.Run(ctx, "kubectl", args...)
	if err != nil {
		return nil, wrapCommandError(err)
	}
	return parsePods(strings.Join(lines, "\n"))
}

func (c *Client) ListServices(ctx context.Context, namespace string) ([]Service, error) {
	if !c.Available() {
		return nil, ErrUnavailable
	}
	if err := validateNamespace(namespace); err != nil {
		return nil, err
	}
	args := []string{"get", "svc", "-o", "json"}
	if strings.TrimSpace(namespace) != "" {
		args = append(args, "-n", namespace)
	} else {
		args = append(args, "-A")
	}
	lines, err := c.runner.Run(ctx, "kubectl", args...)
	if err != nil {
		return nil, wrapCommandError(err)
	}
	return parseServices(strings.Join(lines, "\n"))
}

func (c *Client) Summary(ctx context.Context, namespace string) (Summary, error) {
	pods, err := c.ListPods(ctx, namespace)
	if err != nil {
		return Summary{}, err
	}
	services, err := c.ListServices(ctx, namespace)
	if err != nil {
		return Summary{}, err
	}
	contextName, err := c.CurrentContext(ctx)
	if err != nil {
		return Summary{}, err
	}
	var running int64
	for _, pod := range pods {
		if strings.EqualFold(pod.Status, "Running") {
			running++
		}
	}
	return Summary{
		Context:      contextName,
		PodCount:     len(pods),
		RunningPods:  running,
		ServiceCount: len(services),
	}, nil
}

func wrapCommandError(err error) error {
	if errors.Is(err, executil.ErrCommandNotFound) {
		return ErrUnavailable
	}
	var cmdErr *executil.CommandError
	if errors.As(err, &cmdErr) {
		return fmt.Errorf("%w: %s", ErrCommandFailed, cmdErr.Message)
	}
	return fmt.Errorf("%w: %s", ErrCommandFailed, err.Error())
}

func validateNamespace(namespace string) error {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil
	}
	if !namespacePattern.MatchString(namespace) {
		return errors.New("invalid namespace")
	}
	return nil
}

func parsePods(raw string) ([]Pod, error) {
	result := make([]Pod, 0)
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("%w: empty kubectl pods output", ErrCommandFailed)
	}
	var root struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return nil, fmt.Errorf("%w: invalid pods json: %v", ErrCommandFailed, err)
	}
	for _, itemRaw := range root.Items {
		var item struct {
			Metadata struct {
				Namespace string `json:"namespace"`
				Name      string `json:"name"`
			} `json:"metadata"`
			Spec struct {
				NodeName   string `json:"nodeName"`
				Containers []struct {
					Ports []struct {
						ContainerPort int    `json:"containerPort"`
						Protocol      string `json:"protocol"`
						Name          string `json:"name"`
						HostPort      int    `json:"hostPort"`
					} `json:"ports"`
				} `json:"containers"`
			} `json:"spec"`
			Status struct {
				Phase string `json:"phase"`
				PodIP string `json:"podIP"`
			} `json:"status"`
		}
		if err := json.Unmarshal(itemRaw, &item); err != nil {
			return nil, fmt.Errorf("%w: invalid pod item: %v", ErrCommandFailed, err)
		}
		ports := make([]PodPort, 0)
		for _, container := range item.Spec.Containers {
			for _, p := range container.Ports {
				port := PodPort{
					ContainerPort: p.ContainerPort,
					Protocol:      defaultString(p.Protocol, "TCP"),
					Name:          p.Name,
				}
				if p.HostPort > 0 {
					port.HostPort = strconv.Itoa(p.HostPort)
				}
				ports = append(ports, port)
			}
		}
		node := item.Spec.NodeName
		if node == "" {
			node = "-"
		}
		podIP := item.Status.PodIP
		if podIP == "" {
			podIP = "-"
		}
		result = append(result, Pod{
			Namespace: item.Metadata.Namespace,
			Name:      item.Metadata.Name,
			Status:    item.Status.Phase,
			Node:      node,
			PodIP:     podIP,
			Ports:     ports,
		})
	}
	return result, nil
}

func parseServices(raw string) ([]Service, error) {
	result := make([]Service, 0)
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("%w: empty kubectl services output", ErrCommandFailed)
	}
	var root struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return nil, fmt.Errorf("%w: invalid services json: %v", ErrCommandFailed, err)
	}
	for _, itemRaw := range root.Items {
		var item struct {
			Metadata struct {
				Namespace string `json:"namespace"`
				Name      string `json:"name"`
			} `json:"metadata"`
			Spec struct {
				Type      string `json:"type"`
				ClusterIP string `json:"clusterIP"`
				Ports     []struct {
					Name       string          `json:"name"`
					Port       int             `json:"port"`
					TargetPort json.RawMessage `json:"targetPort"`
					NodePort   int             `json:"nodePort"`
					Protocol   string          `json:"protocol"`
				} `json:"ports"`
			} `json:"spec"`
		}
		if err := json.Unmarshal(itemRaw, &item); err != nil {
			return nil, fmt.Errorf("%w: invalid service item: %v", ErrCommandFailed, err)
		}
		ports := make([]ServicePort, 0)
		for _, p := range item.Spec.Ports {
			sp := ServicePort{
				Name:       p.Name,
				Port:       p.Port,
				Protocol:   defaultString(p.Protocol, "TCP"),
				TargetPort: parseTargetPort(p.TargetPort),
			}
			if p.NodePort > 0 {
				nodePort := p.NodePort
				sp.NodePort = &nodePort
			}
			ports = append(ports, sp)
		}
		clusterIP := item.Spec.ClusterIP
		if clusterIP == "" {
			clusterIP = "-"
		}
		result = append(result, Service{
			Namespace: item.Metadata.Namespace,
			Name:      item.Metadata.Name,
			Type:      defaultString(item.Spec.Type, "ClusterIP"),
			ClusterIP: clusterIP,
			Ports:     ports,
		})
	}
	return result, nil
}

func parseTargetPort(raw json.RawMessage) TargetPort {
	if len(raw) == 0 || string(raw) == "null" {
		return TargetPort{}
	}
	var asInt int
	if err := json.Unmarshal(raw, &asInt); err == nil {
		return TargetPort{Number: &asInt}
	}
	var asStr string
	if err := json.Unmarshal(raw, &asStr); err == nil && asStr != "" {
		return TargetPort{Name: asStr}
	}
	return TargetPort{}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func DefaultTimeout() time.Duration {
	return 60 * time.Second
}
