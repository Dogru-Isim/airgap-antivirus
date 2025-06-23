package config

import (
	"regexp"
	"testing"
)

func TestLoadCPULogger(t *testing.T) {
	appConfig := Load()
	want := regexp.MustCompile("pretty|json")
	cpuLogger := appConfig.CPULogger
	if !want.MatchString(cpuLogger) {
		t.Errorf(`appConfig().CPULogger = %q, want match for %#q`, cpuLogger, want)
	}
}

func TestLoadUSBLogger(t *testing.T) {
	appConfig := Load()
	want := regexp.MustCompile("json")
	usbLogger := appConfig.USBLogger
	if !want.MatchString(usbLogger) {
		t.Errorf(`appConfig().USBLogger = %q, want match for %#q`, usbLogger, want)
	}
}
