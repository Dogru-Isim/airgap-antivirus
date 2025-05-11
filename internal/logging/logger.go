package logging

import (
	"errors"
	"fmt"
	"github.com/Dogru-Isim/airgap-antivirus/internal/config"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
)

//============================  Logger  ============================//

type CPULogger interface {
	LogCPULoadPercentageAverage(log []float64) error
	LogCPULoadPercentagePerCore(log []float64) error
}

type CPULoggerFactory func(opts ...any) (CPULogger, error)

func GetLoggerUsingConfig() (CPULogger, error) {
	appConfig := config.Load()

	// Map of logger factory functions
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

	// Select the logger type from config
	factory, exists := cpuLoggerFactories[appConfig.CPULogger]
	if !exists {
		return nil, fmt.Errorf("unknown CPU logger type: %s", appConfig.CPULogger)
	}

	// Build appropriate options based on logger type
	var opts []any

	switch appConfig.CPULogger {
	case "json":
		logOutput, err := os.OpenFile(filepath.Join(config.Load().ExecutableLocation, "../../../"+config.Load().LogPath+"cpu_load_json.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return nil, fmt.Errorf("cannot open file %s: %w", config.Load().LogPath+"cpu_load_json.log", err)
		}
		opts = append(opts, WithOutputJson(logOutput))
	case "pretty":
		logOutput, err := os.OpenFile(filepath.Join(config.Load().ExecutableLocation, "../../../"+config.Load().LogPath+"cpu_load_pretty.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return nil, fmt.Errorf("cannot open file %s: %w", config.Load().LogPath+"cpu_load_pretty.log", err)
		}
		opts = append(opts,
			WithOutputPretty(logOutput),
			WithPrefix("[CPU] "),
			WithFlags(log.LstdFlags|log.Lshortfile),
		)
	default:
		return nil, fmt.Errorf("unsupported logger type: %s", appConfig.CPULogger)
	}

	return factory(opts...)
}

//============================ PrettyCpuLogger ============================//

type PrettyCPULogger struct {
	logger *log.Logger
}

type PrettyCPULoggerOption func(*PrettyCPULogger) error

func WithOutputPretty(w io.Writer) PrettyCPULoggerOption {
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

func (cpuLogger *PrettyCPULogger) LogCPULoadPercentageAverage(percentage []float64) error {
	cpuLogger.logger.Writer().Write([]byte(fmt.Sprintf("[ Average CPU Load: %5.1f%% ] ", percentage[0])))

	return nil
}

func (cpuLogger *PrettyCPULogger) LogCPULoadPercentagePerCore(percentages []float64) error {
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

func (cpuLogger *JsonCPULogger) LogCPULoadPercentageAverage(percentage []float64) error {
	cpuLogger.logger.Info("CPU metrics",
		"average_load", percentage,
	)

	return nil
}

func (cpuLogger *JsonCPULogger) LogCPULoadPercentagePerCore(percentages []float64) error {
	cpuLogger.logger.Info("CPU metrics",
		"cores", percentages,
	)

	return nil
}
