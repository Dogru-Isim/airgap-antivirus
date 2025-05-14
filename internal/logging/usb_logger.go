package logging

import (
	"context"
	"errors"
	"fmt"
	"github.com/Dogru-Isim/airgap-antivirus/internal/config"
	"io"
	"log/slog"
	"os"
	"sync"
)

type SuspicionLevel int

const (
	SuspicionLevelSuspicious SuspicionLevel = 0
	SuspicionLevelNormal     SuspicionLevel = 1
)

func (suspicionLevel SuspicionLevel) String() string {
	switch suspicionLevel {
	case SuspicionLevelSuspicious:
		return "SUSPICIOUS"
	case SuspicionLevelNormal:
		return "NORMAL"
	default:
		return "UNCLASSIFIED"
	}
}

var lock = &sync.Mutex{}

var loggerInstance USBLogger // singleton

type USBLogger interface {
	Log(usbLogLevel slog.Level, suspicionLevel SuspicionLevel, msg string) error
	SetOutput(w io.Writer)
	SetContext(ctx context.Context)
}

func USBLoggerWithOutput(w io.Writer) USBLoggerOption {
	return func(usbLogger USBLogger) error {
		if w == nil {
			return errors.New("writer cannot be nil")
		}
		usbLogger.SetOutput(w)
		return nil
	}
}

func USBLoggerWithContext(ctx context.Context) USBLoggerOption {
	return func(usbLogger USBLogger) error {
		if ctx == nil {
			return errors.New("writer cannot be nil")
		}
		usbLogger.SetContext(ctx)
		return nil
	}
}

// provides options for USBLogger interface
type USBLoggerOption func(USBLogger) error

// implements USBLogger
type JsonUSBLogger struct {
	logger  *slog.Logger
	s       sync.Once
	context context.Context
	logFile string
}

func NewJsonUSBLogger(options ...USBLoggerOption) (*JsonUSBLogger, error) {
	lock.Lock()
	defer lock.Unlock()

	if loggerInstance == nil {
		// Create a new instance
		loggerInstance = &JsonUSBLogger{
			logger:  slog.New(slog.NewJSONHandler(os.Stdout, nil)),
			logFile: "usb_traffic_json.log",
		}

		// Apply options
		for _, option := range options {
			if err := option(loggerInstance); err != nil {
				return nil, fmt.Errorf("failed to apply option: %w", err)
			}
		}
	} else {
		loggerInstance.Log(slog.LevelDebug, SuspicionLevelNormal, "single USBLogger instance already created")
	}

	return loggerInstance.(*JsonUSBLogger), nil
}

func (jsonUsbLogger *JsonUSBLogger) Log(usbLogLevel slog.Level, suspicionLevel SuspicionLevel, logMsg string) error {
	if jsonUsbLogger.logger == nil {
		return errors.New("logger is not initialized")
	}
	jsonUsbLogger.logger.Log(jsonUsbLogger.context, slog.Level(usbLogLevel), fmt.Sprintf("[%s] %s", suspicionLevel.String(), logMsg))
	return nil
}

func (jsonUsbLogger *JsonUSBLogger) SetOutput(w io.Writer) {
	jsonUsbLogger.logger = slog.New(slog.NewJSONHandler(w, nil))
}

func (jsonUsbLogger *JsonUSBLogger) SetContext(ctx context.Context) {
	jsonUsbLogger.context = ctx
}

func NewUSBLogger(options ...USBLoggerOption) (USBLogger, error) {
	var err error

	switch config.Load().USBLogger {
	case "json":
		loggerInstance, err = NewJsonUSBLogger(options...)
	default:
		return nil, fmt.Errorf("USB logger %s is not an option, please check the documentation", config.Load().USBLogger)
	}

	if err != nil {
		return nil, fmt.Errorf("NewUSBLogger() failed: %w", err)
	}

	return loggerInstance, nil
}
