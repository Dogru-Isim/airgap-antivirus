package monitoring

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"sync"
	"time"
)

// ==================== Static CPU Info ====================
type StaticCPUInfo struct {
	ModelName     string
	LogicalCores  int
	PhysicalCores int
}

type CPUInfoProvider interface {
	GetInfo() (StaticCPUInfo, error)
}

// type SystemCPUInfo struct{}
type SystemCPUInfo struct {
	once    sync.Once
	info    StaticCPUInfo
	initErr error
}

func (s *SystemCPUInfo) GetInfo() (StaticCPUInfo, error) {
	s.once.Do(func() {
		info, err := cpu.Info()
		if err != nil {
			s.initErr = fmt.Errorf("failed to get CPU info: %w", err)
			return
		}
		if len(info) == 0 {
			s.initErr = fmt.Errorf("no CPU info available")
			return
		}

		logical, err := cpu.Counts(true)
		if err != nil {
			s.initErr = fmt.Errorf("failed to get logical cores: %w", err)
			return
		}

		physical, err := cpu.Counts(false)
		if err != nil {
			s.initErr = fmt.Errorf("failed to get physical cores: %w", err)
			return
		}

		s.info = StaticCPUInfo{
			ModelName:     info[0].ModelName,
			LogicalCores:  logical,
			PhysicalCores: physical,
		}
	})

	return s.info, s.initErr
}

// ==================== CPU Metrics ====================
type CPUMetrics struct {
	mu          sync.Mutex
	consumption [][]float64 // [[core1_con, core2_con...], [core1_con, core2_con...]]
	windowSize  int
}

func NewCPUMetrics(windowSize int) (*CPUMetrics, error) {
	if windowSize < 1 {
		return nil, fmt.Errorf("window size must be positive, got %d", windowSize)
	}

	return &CPUMetrics{
		windowSize: windowSize,
	}, nil
}

func (m *CPUMetrics) Record(usage []float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.consumption = append(m.consumption, usage)
	if len(m.consumption) > m.windowSize {
		m.consumption = m.consumption[1:]
	}
}

func (m *CPUMetrics) Recent(n int) [][]float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if n <= 0 || n > len(m.consumption) {
		n = len(m.consumption)
	}

	// Return a copy of the last n elements
	start := len(m.consumption) - n
	return append([][]float64(nil), m.consumption[start:]...)
}

// ==================== CPU Monitor ====================
type CPUMonitor struct {
	infoProvider CPUInfoProvider
	metrics      *CPUMetrics
	Interval     time.Duration
	logger       func(format string, args ...any) (int, error)
	Sync         sync.Once
}

type CPUMonitorOption func(*CPUMonitor)

func WithLogger(logger func(string, ...any) (int, error)) CPUMonitorOption {
	return func(m *CPUMonitor) {
		m.logger = logger
	}
}

func WithInfoProvider(provider CPUInfoProvider) CPUMonitorOption {
	return func(m *CPUMonitor) {
		m.infoProvider = provider
	}
}

func WithInterval(interval time.Duration) CPUMonitorOption {
	return func(m *CPUMonitor) {
		m.Interval = interval
	}
}

func NewCPUMonitor(windowSize int, opts ...CPUMonitorOption) (*CPUMonitor, error) {
	metrics, err := NewCPUMetrics(windowSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics: %w", err)
	}

	monitor := &CPUMonitor{
		infoProvider: &SystemCPUInfo{},
		metrics:      metrics,
		logger:       fmt.Printf, // Default to fmt.Printf
	}

	for _, opt := range opts {
		opt(monitor)
	}

	return monitor, nil
}

func (m *CPUMonitor) GetCPUInfo() (StaticCPUInfo, error) {
	return m.infoProvider.GetInfo()
}

func (m *CPUMonitor) CollectMetrics() error {
	percentages, err := cpu.Percent(m.Interval, true)
	if err != nil {
		return fmt.Errorf("failed to get CPU usage: %w", err)
	}
	if len(percentages) == 0 {
		return fmt.Errorf("no CPU usage data available")
	}

	m.metrics.Record(percentages)
	currentMetrics := formatCoreMetrics(percentages) // Assuming percentages is [][]float64
	historical := formatHistorical(m.metrics.Recent(5))

	fmt.Printf("Current CPU metrics:\n%s\n%s\n",
		currentMetrics,
		historical)

	return nil

}
