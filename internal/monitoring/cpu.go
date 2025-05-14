package monitoring

import (
	"context"
	"fmt"
	"github.com/Dogru-Isim/airgap-antivirus/internal/logging"
	"github.com/shirou/gopsutil/cpu"
	"log"
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
	mu                 sync.Mutex
	consumptionPerCore [][]float64 // [[core1_con, core2_con, ...], [core1_con, core2_con, ...]]
	consumptionAverage [][]float64 // [average1_con, average2_con, ...]
	windowSize         int
}

func NewCPUMetrics(windowSize int) (*CPUMetrics, error) {
	if windowSize < 1 {
		return nil, fmt.Errorf("window size must be positive, got %d", windowSize)
	}

	return &CPUMetrics{
		windowSize: windowSize,
	}, nil
}

func (m *CPUMetrics) RecordPerCore(usagePerCore []float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.consumptionPerCore = append(m.consumptionPerCore, usagePerCore)
	if len(m.consumptionPerCore) > m.windowSize {
		m.consumptionPerCore = m.consumptionPerCore[1:]
	}
}

func (m *CPUMetrics) RecordAverage(usageAverage []float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.consumptionAverage = append(m.consumptionAverage, usageAverage)
	if len(m.consumptionPerCore) > m.windowSize {
		m.consumptionPerCore = m.consumptionPerCore[1:]
	}
}

func (m *CPUMetrics) Recent(n int) [][]float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if n <= 0 || n > len(m.consumptionPerCore) {
		n = len(m.consumptionPerCore)
	}

	// Return a copy of the last n elements
	start := len(m.consumptionPerCore) - n
	return append([][]float64(nil), m.consumptionPerCore[start:]...)
}

// ==================== CPU Monitor ====================
type CPUMonitor struct {
	infoProvider CPUInfoProvider
	metrics      *CPUMetrics
	cpuInfo      *StaticCPUInfo
	Interval     time.Duration
	logger       logging.CPULogger //*logging.CPULogger
	Sync         sync.Once
}

type CPUMonitorOption func(*CPUMonitor)

// func WithLogger(logger func(string, ...any) (int, error)) CPUMonitorOption {
func WithLogger(logger *logging.CPULogger) CPUMonitorOption {
	return func(m *CPUMonitor) {
		m.logger = *logger
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

	logger, err := logging.NewJsonCPULogger()
	if err != nil {
		return nil, fmt.Errorf("failed to create CPU logger: %w", err)
	}

	monitor := &CPUMonitor{
		infoProvider: &SystemCPUInfo{},
		metrics:      metrics,
		logger:       logger,
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
	percentagesPerCore, err := cpu.Percent(0, true)
	percentageAverage, err := cpu.Percent(0, false)
	if err != nil {
		return fmt.Errorf("failed to get CPU usage: %w", err)
	}
	if len(percentagesPerCore) == 0 {
		return fmt.Errorf("no CPU usage data available")
	}

	m.metrics.RecordPerCore(percentagesPerCore)
	m.metrics.RecordAverage(percentageAverage)

	// Log average
	m.logger.LogCPULoadPercentageAverage(percentageAverage)
	// Log per core
	m.logger.LogCPULoadPercentagePerCore(percentagesPerCore)

	/*
		historical := logging.FormatHistorical(m.metrics.Recent(5))
		fmt.Printf("%s\n", historical)
	*/
	/*
		currentMetrics := logging.formatCoreMetrics(percentageAverage) // Assuming percentages is [][]float64
		historical := logging.formatHistorical(m.metrics.Recent(5))

		fmt.Printf("Current CPU metrics:\n%s\n%s\n",
			currentMetrics,
			historical)
	*/

	return nil

}

func (m *CPUMonitor) Start(ctx context.Context) error {
	ticker := time.NewTicker(m.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gracefully")
			return nil
		case <-ticker.C:
			cpuInfo, err := m.GetCPUInfo()
			m.Sync.Do(func() {
				log.Printf("Number of logical cores: %d", cpuInfo.LogicalCores)
			})
			if err := m.CollectMetrics(); err != nil {
				return fmt.Errorf("cpu monitoring error: %w", err)
			}
			if err != nil {
				return fmt.Errorf("cpu monitoring error: %w", err)
			}
		}
	}
}
