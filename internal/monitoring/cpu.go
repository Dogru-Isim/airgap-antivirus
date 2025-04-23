package monitoring

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

// Cpu represents CPU information
type Cpu struct {
	LogicalCores int
	cpuInfo      []cpu.InfoStat // Added to store CPU info
}

// GetUsagePercentage returns the current CPU usage
func (c *Cpu) GetUsagePercentage() (float64, error) {
	percentages, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		return 0, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	if len(percentages) == 0 {
		return 0, fmt.Errorf("no CPU usage data available")
	}

	return percentages[0], nil
}

// NewCpu creates and initializes a new Cpu instance
func NewCpu() (*Cpu, error) {
	c := &Cpu{}
	if err := c._populate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Populate gathers CPU information
func (c *Cpu) _populate() error {
	cores, err := cpu.Counts(true)
	if err != nil {
		return fmt.Errorf("failed to get CPU cores: %w", err)
	}
	c.LogicalCores = cores

	info, err := cpu.Info()
	if err != nil {
		return fmt.Errorf("failed to get CPU info: %w", err)
	}
	c.cpuInfo = info

	return nil
}
