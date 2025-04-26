package logging

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

//============================  Logger  ============================//

type Logger interface {
	LogPretty(writer io.Writer, log any) error // Log in a format pleasing to the human eye
	//LogJson(writer io.Writer, log any) error // Log in JSON format
}

//============================ CpuLogger ============================//

type CPULogger struct {
	logger *log.Logger
}

type CPULoggerOption func(*CPULogger) error

func WithOutput(w io.Writer) CPULoggerOption {
	return func(cl *CPULogger) error {
		if w == nil {
			return errors.New("writer cannot be nil")
		}
		cl.logger.SetOutput(w)
		return nil
	}
}

func WithPrefix(prefix string) CPULoggerOption {
	return func(cl *CPULogger) error {
		cl.logger.SetPrefix(prefix)
		return nil
	}
}

func WithFlags(flags int) CPULoggerOption {
	return func(cl *CPULogger) error {
		cl.logger.SetFlags(flags)
		return nil
	}
}

func NewCPULogger(opts ...CPULoggerOption) (*CPULogger, error) {
	// Initialize with defaults
	cpuLogger := &CPULogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(cpuLogger); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	return cpuLogger, nil
}

func (cpuLogger *CPULogger) Log(percentages []float64) error {
	/*
		currentMetrics := formatCoreMetrics(percentages) // Assuming percentages is [][]float64
		historical := formatHistorical(m.metrics.Recent(5))

		fmt.Printf("Current CPU metrics:\n%s\n%s\n",
			currentMetrics,
			historical)
	*/

	currentMetrics := formatCoreMetrics(percentages)
	cpuLogger.logger.Writer().Write([]byte(currentMetrics))

	return nil
}
