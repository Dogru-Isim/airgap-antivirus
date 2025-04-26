package logging

import (
	"io"
)

type Logger interface {
	Log(writer io.Writer, log string) error
}
