package logging

import (
	"errors"
	"io"
)

//============================ Logger Interface ============================//

type Logger interface {
	Log(writer io.Writer, log string) error
}

//============================ CpuLogger Struct ============================//

type CpuLogger struct{}

func (cpuLogger *CpuLogger) Log(writer io.Writer, logMessage string) error {
	return errors.New("Non-Implemented")
}
