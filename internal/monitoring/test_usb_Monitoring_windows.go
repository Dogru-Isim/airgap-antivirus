package monitoring

import (
	"context"
	"fmt"
	"os"
)

func MonitorUSB() {
	fmt.Println("MonitorUSB is unimplemented on Windows")
	os.Exit(1)
}

func DetectingUSB(ctx context.Context) {
	fmt.Println("MonitorUSB is unimplemented on Windows")
	os.Exit(1)
}

func GetUSBPath() ([]string, error) {
	fmt.Println("MonitorUSB is unimplemented on Windows")
	os.Exit(1)
	return nil, nil
}
