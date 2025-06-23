package monitoring

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Mock implementation for tests
type MockFanotify struct {
	Fail bool
}

func (m *MockFanotify) Initialize(flags, event_flags uint) (int32, error) {
	if m.Fail {
		return -1, errors.New("simulated fanotify error")
	}
	return 123, nil // Return dummy fd when not testing failure
}

// Test the case where fanotify fails
func TestNewUSBMonitor_FanotifyFailure(t *testing.T) {
	tempDir := t.TempDir()

	mockFanotify := &MockFanotify{Fail: true}

	_, err := NewUSBMonitor(tempDir, mockFanotify)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	expected := "fanotify initialization failed: simulated fanotify error"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestNewUSBMonitor_ValidPath(t *testing.T) {
	tempDir := t.TempDir()

	mockFanotify := &MockFanotify{Fail: false}

	mon, err := NewUSBMonitor(tempDir, mockFanotify)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if mon == nil {
		t.Fatal("Expected monitor instance, got nil")
	}
	if mon.Mountpath != tempDir {
		t.Errorf("Expected mountpath %q, got %q", tempDir, mon.Mountpath)
	}
}

func TestNewUSBMonitor_InvalidPaths(t *testing.T) {
	// Setup a temporary test directory
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "nonexistent")
	filePath := filepath.Join(tempDir, "file")

	// Create a regular file (not directory)
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name      string
		mountpath string
		wantErr   string
	}{
		{
			name:      "nonexistent path",
			mountpath: nonExistentPath,
			wantErr:   nonExistentPath + " does not exist",
		},
		{
			name:      "not a directory",
			mountpath: filePath,
			wantErr:   filePath + " is Not a directory",
		},
	}

	mockFanotify := &MockFanotify{Fail: false}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewUSBMonitor(tt.mountpath, mockFanotify)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("Expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

// It's crucial that the type of USBMonitor.fd is int32 so that it can be safely converted to C.int
func TestNewUSBMonitorFdTypeInt32(t *testing.T) {
	usbMonitor, _ := NewUSBMonitor(t.TempDir(), &MockFanotify{Fail: false})

	x := reflect.TypeOf(usbMonitor.fd).Kind()
	switch x {
	case reflect.Int32:
	default:
		t.Errorf("Expected type of USBMonitor.fd to be int32, got %s", x.String())
	}
}
