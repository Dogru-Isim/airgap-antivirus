package monitoring

/*
#include <stdlib.h>
#include <sys/fanotify.h>
#include <fcntl.h>
#include <unistd.h>
#include <errno.h>

static int getErrno() {
    return errno;
}
*/
import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
	"unsafe"
)

type USBDetector struct {
	Blockdevices []BlockDevice `json:"blockdevices"`
	NewUSB       []USB
	OutputUSB    []USB
	ExistingUSB  []USB
}

type BlockDevice struct {
	Name        string        `json:"name"`
	Mountpoints []string      `json:"mountpoints"`
	Tran        *string       `json:"tran"`
	Children    []BlockDevice `json:"children,omitempty"`
}

type USB struct {
	Name       string
	Partitions []*Partition
}

type Partition struct {
	Name        string
	Mountpoints []string
}

type Monitor struct {
	Mountpath string
	fd        C.int
}

func main() {
	newUSBDetector := USBDetector{}
	fmt.Println("Starting to monitor for USBs:")
	for {
		newUSBDetector.DetectNewUSB()
		//fmt.Println("hi")
		newUSBDetector.USBDifferenceChecker()
		// voor elke mountpoint -> monitor.NewMonitor() -> monitor.Start()
		if newUSBDetector.NewUSB != nil {
			for _, usb := range newUSBDetector.NewUSB {
				for _, partition := range usb.Partitions {
					for _, mountpoint := range partition.Mountpoints {
						monitor, err := NewMonitor(mountpoint)
						if err == nil {
							fmt.Printf("%s is starting\n", monitor.Mountpath)
							go monitor.Start(context.Background())
						}
					}
				}
			}
			newUSBDetector.NewUSB = nil
		}
		time.Sleep(2 * time.Second)

	}

}

func (u *USBDetector) USBDifferenceChecker() {
	if u.OutputUSB != nil {
		if u.ExistingUSB != nil {
			existingMap := make(map[string]bool)
			for _, usb := range u.ExistingUSB {
				existingMap[usb.Name] = true
			}

			var newUSBs []USB
			for _, usb := range u.OutputUSB {
				if !existingMap[usb.Name] {
					newUSBs = append(newUSBs, usb)
				}
			}
			u.NewUSB = newUSBs
			u.ExistingUSB = u.OutputUSB
			u.OutputUSB = nil
		} else {
			u.NewUSB = u.OutputUSB
			u.ExistingUSB = u.OutputUSB
			u.OutputUSB = nil
		}
	}
}
func (u *USBDetector) DetectNewUSB() error {
	cmd := exec.Command("lsblk", "-J", "-o", "NAME,MOUNTPOINTS,TRAN")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("lsblk command failed: %w\nOutput: %s", err, string(out))
	}

	var result struct {
		Blockdevices []BlockDevice `json:"blockdevices"`
	}

	if err := json.Unmarshal(out, &result); err != nil {
		return fmt.Errorf("JSON unmarshal failed: %w\nData: %s", err, string(out))
	}
	//fmt.Println(result)
	u.Blockdevices = result.Blockdevices
	u.OutputUSB = nil

	for _, device := range u.Blockdevices {
		if device.Tran != nil && *device.Tran == "usb" {
			//fmt.Printf("USB device: %s\n", device.Name)
			usb := USB{
				Name: device.Name,
			}

			var processChildren func(children []BlockDevice)
			processChildren = func(children []BlockDevice) {
				for _, child := range children {
					newPartition := &Partition{
						Name: child.Name,
					}
					if child.Mountpoints != nil {
						for _, mountpoint := range child.Mountpoints {
							if mountpoint != "" {
								newPartition.Mountpoints = append(newPartition.Mountpoints, mountpoint)
							}
						}
					}
					usb.Partitions = append(usb.Partitions, newPartition)
					processChildren(child.Children)
				}
			}
			processChildren(device.Children)
			u.OutputUSB = append(u.OutputUSB, usb)
		}
	}

	return nil
}

func NewMonitor(mountpath string) (*Monitor, error) {

	fd, err := C.fanotify_init(C.FAN_REPORT_FID|C.FAN_REPORT_DFID_NAME|C.FAN_CLASS_NOTIF|C.FAN_NONBLOCK, C.O_RDONLY)
	if fd == -1 {
		return nil, fmt.Errorf("fanotify_init failed: %v\n", err)
		//os.Exit(1)
	}

	fileInfo, err := os.Stat(mountpath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("%s does not exist!\n", mountpath)
		//os.Exit(1)
	}
	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("%s is Not a directory!\n", mountpath)
		//os.Exit(1)
	}

	return &Monitor{
		Mountpath: mountpath,
		fd:        fd,
	}, nil

}

func (m *Monitor) Start(ctx context.Context) {

	cPath := C.CString(m.Mountpath)
	defer C.free(unsafe.Pointer(cPath))

	ret, err := C.fanotify_mark(
		m.fd,
		C.FAN_MARK_ADD,
		C.FAN_CREATE|C.FAN_OPEN|C.FAN_MODIFY|
			C.FAN_MOVED_TO|C.FAN_MOVED_FROM|C.FAN_RENAME|
			C.FAN_DELETE|C.FAN_ATTRIB|C.FAN_CLOSE_WRITE|
			C.FAN_CLOSE_NOWRITE|C.FAN_ONDIR|C.FAN_EVENT_ON_CHILD,
		C.AT_FDCWD,
		cPath,
	)
	if ret == -1 {
		errno := C.getErrno()
		fmt.Printf("fanotify_mark failed: %v (errno=%d)\n", err, errno)
		//os.Exit(1)
	}

	fmt.Printf("Monitoring %s...\n", m.Mountpath)
	for {
		foundMountpoint, err := m.MountpointChecker()
		if err != nil && !foundMountpoint {
			fmt.Printf("Stopped monitoring on %s\n", m.Mountpath)
			defer C.close(m.fd)
			return
		}

		buf := make([]byte, 4096)
		n, _ := C.read(C.int(m.fd), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
		if n <= 0 {
			continue
		}
		metadata := (*C.struct_fanotify_event_metadata)(unsafe.Pointer(&buf[0]))

		if metadata.mask&C.FAN_OPEN != 0 {
			fmt.Printf("[%s][%s]Open detected from PID: %d\n", time.Now().Format("15:04:05"), m.Mountpath, metadata.pid)
		}
		if metadata.mask&C.FAN_CREATE != 0 {
			fmt.Printf("[%s][%s]Create detected from PID: %d\n", time.Now().Format("15:04:05"), m.Mountpath, metadata.pid)
		}
		if metadata.mask&C.FAN_DELETE != 0 {
			fmt.Printf("[%s][%s]Delete detected from PID: %d\n", time.Now().Format("15:04:05"), m.Mountpath, metadata.pid)
		}
		if metadata.mask&C.FAN_MOVED_FROM != 0 {
			fmt.Printf("[%s][%s] Moved FROM detected from PID: %d\n", time.Now().Format("15:04:05"), m.Mountpath, metadata.pid)
		}
		if metadata.mask&C.FAN_MOVED_TO != 0 {
			fmt.Printf("[%s][%s] Moved TO detected from PID: %d\n", time.Now().Format("15:04:05"), m.Mountpath, metadata.pid)
		}
		if metadata.mask&C.FAN_RENAME != 0 {
			fmt.Printf("[%s][%s] Rename detected from PID: %d\n", time.Now().Format("15:04:05"), m.Mountpath, metadata.pid)
		}
		if metadata.mask&C.FAN_ATTRIB != 0 {
			fmt.Printf("[%s][%s]Attribute change detected from PID: %d\n", time.Now().Format("15:04:05"), m.Mountpath, metadata.pid)
		}
		if metadata.mask&C.FAN_CLOSE_WRITE != 0 {
			fmt.Printf("[%s][%s]Write close  detected from PID: %d\n", time.Now().Format("15:04:05"), m.Mountpath, metadata.pid)
		}
		if metadata.mask&C.FAN_CLOSE_NOWRITE != 0 {
			fmt.Printf("[%s][%s]Read close detected from PID: %d\n", time.Now().Format("15:04:05"), m.Mountpath, metadata.pid)
		}
		if metadata.mask&C.FAN_MODIFY != 0 {
			fmt.Printf("[%s][%s]Write detected from PID: %d\n", time.Now().Format("15:04:05"), m.Mountpath, metadata.pid)
		}
	}
}

func (m *Monitor) MountpointChecker() (bool, error) {
	cmd := exec.Command("lsblk", "-J", "-o", "NAME,MOUNTPOINTS,TRAN")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("lsblk command failed: %w\nOutput: %s", err, string(out))
	}

	var result struct {
		Blockdevices []BlockDevice `json:"blockdevices"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return false, fmt.Errorf("JSON unmarshal failed: %w\nData: %s", err, string(out))
	}
	//fmt.Println(result)

	//foundMountpoint := false
	for _, device := range result.Blockdevices {
		if device.Tran != nil && *device.Tran == "usb" {
			for _, child := range device.Children {
				for _, mountpoint := range child.Mountpoints {
					if mountpoint == m.Mountpath {
						//fmt.Println("found")
						return true, nil
					}
				}
			}
		}
	}
	return false, fmt.Errorf("mountpoint not found")
}
