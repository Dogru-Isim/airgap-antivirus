package logging

import (
	"errors"
	"fmt"
	"github.com/Dogru-Isim/airgap-antivirus/internal/config"
	"io"
	"log"
	"log/slog"
	"os"
)

//============================  Logger  ============================//

type CPULogger interface {
	LogCPULoadPercentage(log []float64) error
}

type CPULoggerFactory func(opts ...any) (CPULogger, error)

func GetLoggerUsingConfig() (CPULogger, error) {
	appConfig, err := config.Load()
	if err != nil {
		return nil, err
	}

	var cpuLoggerFactories = map[string]CPULoggerFactory{
		"json": func(opts ...any) (CPULogger, error) {
			jsonOptions := make([]JsonCPULoggerOption, len(opts))
			for i, opt := range opts {
				jsonOptions[i], _ = opt.(JsonCPULoggerOption)
			}
			return NewJsonCPULogger(jsonOptions...)
		},
		"pretty": func(opts ...any) (CPULogger, error) {
			prettyOptions := make([]PrettyCPULoggerOption, len(opts))
			for i, opt := range opts {
				prettyOptions[i], _ = opt.(PrettyCPULoggerOption)
			}
			return NewPrettyCPULogger(prettyOptions...)
		},
	}
	factory, exists := cpuLoggerFactories[appConfig.CPULogger]
	if !exists {
		return nil, errors.New("unknown CPU logger type: " + appConfig.CPULogger)
	}
	cpuLogger, err := factory()
	if err != nil {
		return nil, errors.New("factory() failed in GetLoggerUsingConfig()")
	}
	return cpuLogger, nil
}

//============================ PrettyCpuLogger ============================//

type PrettyCPULogger struct {
	logger *log.Logger
}

type PrettyCPULoggerOption func(*PrettyCPULogger) error

func WithOutput(w io.Writer) PrettyCPULoggerOption {
	return func(cl *PrettyCPULogger) error {
		if w == nil {
			return errors.New("writer cannot be nil")
		}
		cl.logger.SetOutput(w)
		return nil
	}
}

func WithPrefix(prefix string) PrettyCPULoggerOption {
	return func(cl *PrettyCPULogger) error {
		cl.logger.SetPrefix(prefix)
		return nil
	}
}

func WithFlags(flags int) PrettyCPULoggerOption {
	return func(cl *PrettyCPULogger) error {
		cl.logger.SetFlags(flags)
		return nil
	}
}

func NewPrettyCPULogger(opts ...PrettyCPULoggerOption) (*PrettyCPULogger, error) {
	// Initialize with defaults
	cpuLogger := &PrettyCPULogger{
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

func (cpuLogger *PrettyCPULogger) LogCPULoadPercentage(percentages []float64) error {
	currentMetrics := formatCoreMetrics(percentages)
	cpuLogger.logger.Writer().Write([]byte(currentMetrics))

	return nil
}

//============================ JsonCpuLogger ============================//

type JsonCPULogger struct {
	logger *slog.Logger
}

type JsonCPULoggerOption func() (*slog.Logger, error)

func WithOutputJson(w io.Writer) JsonCPULoggerOption {
	return func() (*slog.Logger, error) {
		if w == nil {
			return slog.New(slog.NewJSONHandler(w, nil)), errors.New("writer cannot be nil")
		}
		return slog.New(slog.NewJSONHandler(w, nil)), nil
	}
}

func NewJsonCPULogger(opts ...JsonCPULoggerOption) (*JsonCPULogger, error) {
	jsonCpuLogger := &JsonCPULogger{
		slog.New(slog.NewJSONHandler(os.Stderr, nil)),
	}
	var err error

	// Apply options
	for _, opt := range opts {
		jsonCpuLogger.logger, err = opt()
		if err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	return jsonCpuLogger, nil
}

func (cpuLogger *JsonCPULogger) LogCPULoadPercentage(percentages []float64) error {
	cpuLogger.logger.Info("CPU metrics",
		"cores", percentages,
	)

	return nil
}
