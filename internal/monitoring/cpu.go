package monitoring

import (
	"errors"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"sync"
	"time"
)

//===============================================================//

type ICpuStaticInfoProvider interface {
	GetGeneralInfo() []cpu.InfoStat
	GetLogicalCores() int
}

type CpuStaticInfoProvider struct {
	generalInfo  []cpu.InfoStat
	logicalCores int
}

func NewCpuStaticInfoProvider() (*CpuStaticInfoProvider, error) {
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("read cpu stats error: %w", err)
	}

	logicalCoresNum, err := cpu.Counts(true)
	if err != nil {
		return nil, fmt.Errorf("read cpu stats error: %w", err)
	}

	staticInfoProvider := &CpuStaticInfoProvider{
		generalInfo:  cpuInfo,
		logicalCores: logicalCoresNum,
	}

	return staticInfoProvider, nil
}

func (csip CpuStaticInfoProvider) GetGeneralInfo() []cpu.InfoStat {
	return csip.generalInfo

}

func (csip CpuStaticInfoProvider) GetLogicalCores() int {
	return csip.logicalCores
}

//=========================================================================//

type ICpuMetricsProvider interface {
	LogUsage() error
	getRecentUsage() []float64
	recordUsage(percentage float64)
}

type CpuMetricsProvider struct {
	mutex       sync.Mutex
	windowSize  int
	consumption []float64
}

func NewCpuMetricsProvider(windowSize int) (*CpuMetricsProvider, error) {
	metricsProvider := &CpuMetricsProvider{
		windowSize: windowSize,
	}

	if windowSize < 1 {
		return nil, errors.New("window size cannot be less than 1")
	}

	return metricsProvider, nil
}

func (metricsProvider *CpuMetricsProvider) recordUsage(percentage float64) {
	metricsProvider.consumption = append(metricsProvider.consumption, percentage)

	// Maintain rolling window
	if len(metricsProvider.consumption) > metricsProvider.windowSize {
		metricsProvider.consumption = metricsProvider.consumption[1:]
	}
}

func (metricsProvider *CpuMetricsProvider) getRecentUsage() []float64 {
	return append([]float64(nil), metricsProvider.consumption...)
}

func (metricsProvider *CpuMetricsProvider) LogUsage() error {
	metricsProvider.mutex.Lock()
	defer metricsProvider.mutex.Unlock()
	percentage, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		return err
	}

	if len(percentage) == 0 {
		return fmt.Errorf("no cpu data available")
	}

	metricsProvider.recordUsage(percentage[0])

	fmt.Printf("Current CPU: %.2f%%, Recent: %v\n",
		percentage[0],
		metricsProvider.getRecentUsage())

	return nil
}

//=========================================================================//

type CpuMonitor struct {
	CpuStaticInfoProvider ICpuStaticInfoProvider
	CpuMetricsProvider    ICpuMetricsProvider
}

func NewCpuMonitor(windowSize int) (*CpuMonitor, error) {
	staticInfoProvider, err := NewCpuStaticInfoProvider()
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	metricsProvider, err := NewCpuMetricsProvider(windowSize)
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	cpuMonitor := &CpuMonitor{
		CpuStaticInfoProvider: staticInfoProvider,
		CpuMetricsProvider:    metricsProvider,
	}

	return cpuMonitor, nil
}
