package logging

import (
	"errors"
	"io"
	"log"
)

//============================  Logger  ============================//

type Logger interface {
	Log(writer io.Writer, log string) error
}

//============================ CpuLogger ============================//

type CPULogger struct {
	writer io.Writer
}

type CPULoggerOption func(*CPULogger)

func WithWriter(writer io.Writer) CPULoggerOption {
	return func(logger *CPULogger) {
		logger.writer = writer
	}
}

func (cpuLogger *CPULogger) Log(writer io.Writer, logMessage string) error {
	return errors.New("Non-Implemented")
}
