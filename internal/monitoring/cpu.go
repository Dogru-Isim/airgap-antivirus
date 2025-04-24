package monitoring

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"sync"
	"time"
)

type CpuMonitor struct {
	//deviceCpu   cpu.InfoStat
	logicalCores int
	consumption  []float64
	mu           sync.Mutex
	windowSize   int
}

// Getter for logicalCores
func (c *CpuMonitor) LogicalCores() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.logicalCores
}

func NewCpuMonitor(windowSize int) (*CpuMonitor, error) {
	cpuMonitor := &CpuMonitor{
		windowSize: windowSize,
	}

	if err := cpuMonitor.populate(); err != nil {
		return nil, err
	}

	return cpuMonitor, nil
}

func (c *CpuMonitor) populate() error {
	cores, err := cpu.Counts(true)
	if err != nil {
		return fmt.Errorf("cpu cores: %w", err)
	}
	c.logicalCores = cores
	return nil
}

func (c *CpuMonitor) RecordUsage(percentage float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.consumption = append(c.consumption, percentage)

	// Maintain rolling window
	if len(c.consumption) > c.windowSize {
		c.consumption = c.consumption[1:]
	}
}

func (c *CpuMonitor) GetRecentUsage() []float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]float64(nil), c.consumption...)
}

func (c *CpuMonitor) LogUsage() error {
	percentage, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		return err
	}

	if len(percentage) == 0 {
		return fmt.Errorf("no cpu data available")
	}

	c.RecordUsage(percentage[0])

	fmt.Printf("Current CPU: %.2f%%, Recent: %v\n",
		percentage[0],
		c.GetRecentUsage())

	return nil
}
