//go:build windows

package system

import (
	"context"
	"math"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	pdhStatusOK  uintptr = 0
	pdhFmtDouble uintptr = 0x00000200
)

var (
	pdhDLL                          = windows.NewLazySystemDLL("pdh.dll")
	procPdhOpenQuery                = pdhDLL.NewProc("PdhOpenQueryW")
	procPdhAddEnglishCounter        = pdhDLL.NewProc("PdhAddEnglishCounterW")
	procPdhCollectQueryData         = pdhDLL.NewProc("PdhCollectQueryData")
	procPdhGetFormattedCounterValue = pdhDLL.NewProc("PdhGetFormattedCounterValue")
	procPdhCloseQuery               = pdhDLL.NewProc("PdhCloseQuery")
)

type windowsCPUUsageProvider struct {
	mu      sync.Mutex
	query   uintptr
	counter uintptr
	latest  float64
	ready   bool
}

type pdhFormattedCounterValueDouble struct {
	status uint32
	_      uint32
	value  float64
}

func newCPUUsageProvider() cpuUsageProvider {
	provider := &windowsCPUUsageProvider{}
	if !provider.open(`\Processor Information(_Total)\% Processor Utility`) {
		if !provider.open(`\Processor(_Total)\% Processor Time`) {
			return nil
		}
	}
	provider.collect()
	go provider.sampleLoop()
	return provider
}

func (p *windowsCPUUsageProvider) Usage(ctx context.Context) (float64, bool) {
	select {
	case <-ctx.Done():
		return 0, false
	default:
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	return p.latest, p.ready
}

func (p *windowsCPUUsageProvider) sampleLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		p.sample()
	}
}

func (p *windowsCPUUsageProvider) sample() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.collect() {
		return
	}
	var value pdhFormattedCounterValueDouble
	ret, _, _ := procPdhGetFormattedCounterValue.Call(
		p.counter,
		pdhFmtDouble,
		0,
		uintptr(unsafe.Pointer(&value)),
	)
	if ret != pdhStatusOK || uintptr(value.status) != pdhStatusOK {
		return
	}
	if math.IsNaN(value.value) || math.IsInf(value.value, 0) || value.value < 0 {
		return
	}
	if value.value > 100 {
		value.value = 100
	}
	p.latest = value.value
	p.ready = true
}

func (p *windowsCPUUsageProvider) open(counterPath string) bool {
	p.close()

	var query uintptr
	ret, _, _ := procPdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&query)))
	if ret != pdhStatusOK {
		return false
	}

	path, err := windows.UTF16PtrFromString(counterPath)
	if err != nil {
		procPdhCloseQuery.Call(query)
		return false
	}

	var counter uintptr
	ret, _, _ = procPdhAddEnglishCounter.Call(
		query,
		uintptr(unsafe.Pointer(path)),
		0,
		uintptr(unsafe.Pointer(&counter)),
	)
	if ret != pdhStatusOK {
		procPdhCloseQuery.Call(query)
		return false
	}

	p.query = query
	p.counter = counter
	return true
}

func (p *windowsCPUUsageProvider) collect() bool {
	ret, _, _ := procPdhCollectQueryData.Call(p.query)
	return ret == pdhStatusOK
}

func (p *windowsCPUUsageProvider) close() {
	if p.query != 0 {
		procPdhCloseQuery.Call(p.query)
		p.query = 0
		p.counter = 0
	}
}
