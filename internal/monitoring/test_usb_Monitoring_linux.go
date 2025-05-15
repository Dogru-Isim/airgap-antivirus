package monitoring

/*
#cgo pkg-config: libudev
#include <libudev.h>
#include <stdio.h>
#include <fcntl.h>
#include <sys/fanotify.h>
#include <unistd.h>
#include <stdlib.h>
#include <errno.h>

static int getErrno() {
    return errno;
}
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
	"unsafe"
)

func DetectingUSB() {
	processedDevices := make(map[string]bool)
	if CheckPluggedUSB() {
		mountPaths, err := GetUSBPath()
		if err != nil {
			fmt.Printf("Error getting USB paths: %v\n", err)
		} else {
			for _, path := range mountPaths {
				go MonitorUSB(path) // Start monitoring in aparte goroutine
				processedDevices[path] = true
			}
		}
	}

	udev := C.udev_new()
	if udev == nil {
		fmt.Println("Failed to create udev context")
		return
	}
	defer C.udev_unref(udev)

	monitor := C.udev_monitor_new_from_netlink(udev, C.CString("udev"))
	if monitor == nil {
		fmt.Println("Failed to create monitor")
		return
	}
	defer C.udev_monitor_unref(monitor)

	C.udev_monitor_filter_add_match_subsystem_devtype(monitor, C.CString("usb"), C.CString("usb_device"))
	C.udev_monitor_enable_receiving(monitor)

	fmt.Println("Monitoring USB devices...")

	for {

		dev := C.udev_monitor_receive_device(monitor)
		if dev == nil {
			time.Sleep(2 * time.Second) //<so that cpu does not overload>
			continue
		}

		action := C.GoString(C.udev_device_get_action(dev))
		devpath := C.GoString(C.udev_device_get_devpath(dev))
		vendor := C.GoString(C.udev_device_get_property_value(dev, C.CString("ID_VENDOR")))
		model := C.GoString(C.udev_device_get_property_value(dev, C.CString("ID_MODEL")))

		if action == "add" {
			fmt.Printf("USB connected: %s %s (%s)\n", vendor, model, devpath)
			maxattempt := 10
			var mountPath []string
			var err error

			for attempt := 1; attempt <= maxattempt; attempt++ {
				mountPath, err = GetUSBPath()
				if err == nil && len(mountPath) > 0 {
					break
				}
				if err != nil {
					fmt.Printf("Attempt %d failed:%v\n", attempt, err)
				}

				fmt.Printf("Next attempt will begin in 2 seconds\n")
				time.Sleep(2 * time.Second)

			}

			for _, path := range mountPath {
				//fmt.Println("hello usb")
				if processedDevices[path] {
					fmt.Printf("%s is already being monitored\n", path)
					continue
				}
				go MonitorUSB(path)
				processedDevices[path] = true
			}
		} else if action == "remove" {
			fmt.Printf("USB disconnected: %s\n", devpath)
		}
		C.udev_device_unref(dev)
	}
}

func MonitorUSB(path string) {
	// Initialize fanotify (zonder O_LARGEFILE)
	fd, err := C.fanotify_init(C.FAN_REPORT_FID|C.FAN_REPORT_DFID_NAME|C.FAN_CLASS_NOTIF|C.FAN_NONBLOCK, C.O_RDONLY)
	if fd == -1 {
		fmt.Printf("fanotify_init failed: %v\n", err)
		os.Exit(1)
	}
	defer C.close(fd)

	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Printf("%s does not exist!\n", path)
		os.Exit(1)
	}
	if !fileInfo.IsDir() {
		fmt.Printf("%s is Not a directory!\n", path)
		os.Exit(1)
	}

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret, err := C.fanotify_mark(
		fd,
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
		os.Exit(1)
	}

	fmt.Printf("Monitoring %s...\n", path)
	for {
		buf := make([]byte, 4096)
		n, _ := C.read(C.int(fd), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
		if n <= 0 {
			continue
		}
		metadata := (*C.struct_fanotify_event_metadata)(unsafe.Pointer(&buf[0]))

		if metadata.mask&C.FAN_OPEN != 0 {
			fmt.Printf("[%s][%s]Open detected from PID: %d\n", time.Now().Format("15:04:05"), path, metadata.pid)
		}
		if metadata.mask&C.FAN_CREATE != 0 {
			fmt.Printf("[%s][%s]Create detected from PID: %d\n", time.Now().Format("15:04:05"), path, metadata.pid)
		}
		if metadata.mask&C.FAN_DELETE != 0 {
			fmt.Printf("[%s][%s]Delete detected from PID: %d\n", time.Now().Format("15:04:05"), path, metadata.pid)
		}
		if metadata.mask&C.FAN_MOVED_FROM != 0 {
			fmt.Printf("[%s][%s] Moved FROM detected from PID: %d\n", time.Now().Format("15:04:05"), path, metadata.pid)
		}
		if metadata.mask&C.FAN_MOVED_TO != 0 {
			fmt.Printf("[%s][%s] Moved TO detected from PID: %d\n", time.Now().Format("15:04:05"), path, metadata.pid)
		}
		if metadata.mask&C.FAN_RENAME != 0 {
			fmt.Printf("[%s][%s] Rename detected from PID: %d\n", time.Now().Format("15:04:05"), path, metadata.pid)
		}
		if metadata.mask&C.FAN_ATTRIB != 0 {
			fmt.Printf("[%s][%s]Attribute change detected from PID: %d\n", time.Now().Format("15:04:05"), path, metadata.pid)
		}
		if metadata.mask&C.FAN_CLOSE_WRITE != 0 {
			fmt.Printf("[%s][%s]Attribute change detected from PID: %d\n", time.Now().Format("15:04:05"), path, metadata.pid)
		}
		if metadata.mask&C.FAN_CLOSE_NOWRITE != 0 {
			fmt.Printf("[%s][%s]Attribute change detected from PID: %d\n", time.Now().Format("15:04:05"), path, metadata.pid)
		}
		if metadata.mask&C.FAN_MODIFY != 0 {
			fmt.Printf("[%s][%s]Write detected from PID: %d\n", time.Now().Format("15:04:05"), path, metadata.pid)
		}
	}
}

/////////////////////
////MOUNTPOINTS/////
///////////////////

type LSBLKMountpointOutput struct {
	Blockdevices []MountpointBlockDevice `json:"blockdevices"`
}

type MountpointBlockDevice struct {
	Name        string                  `json:"name"`
	Mountpoints []string                `json:"mountpoints"`
	Tran        *string                 `json:"tran"`
	Children    []MountpointBlockDevice `json:"children,omitempty"`
}

func GetUSBPath() ([]string, error) {

	cmd := exec.Command("lsblk", "-J", "-o", "NAME,MOUNTPOINTS,TRAN")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk command failed: %w\nOutput: %s", err, string(out))
	}

	var result LSBLKMountpointOutput
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w\nData: %s", err, string(out))
	}
	//fmt.Println(result)

	var allMountpoints []string
	// Doorloop alle block devices
	for _, device := range result.Blockdevices {
		if device.Tran != nil && *device.Tran == "usb" {
			fmt.Printf("USB device: %s\n", device.Name)
			for _, child := range device.Children {
				//fmt.Println("Hello")
				if child.Mountpoints != nil {
					for _, mountpoint := range child.Mountpoints {
						allMountpoints = append(allMountpoints, mountpoint)
						fmt.Printf("  Partitie: %s, Mountpoint: %s\n", child.Name, mountpoint)

					}
					fmt.Println(allMountpoints)
				} else {
					fmt.Printf("  Partitie: %s, geen mountpoint\n", child.Name)
				}
			}
		}
	}

	return allMountpoints, nil

}

type LSBLKTranOutput struct {
	Blockdevices []TranBlockDevice `json:"blockdevices"`
}

type TranBlockDevice struct {
	Name string  `json:"name"`
	Tran *string `json:"tran"`
}

func CheckPluggedUSB() bool {
	cmd := exec.Command("lsblk", "-J", "-o", "NAME,TRAN")
	out, err := cmd.Output()
	if err != nil {
		fmt.Errorf("lsblk command failed: %w\nOutput: %s", err, string(out))
	}

	var result LSBLKTranOutput
	if err := json.Unmarshal(out, &result); err != nil {
		fmt.Errorf("JSON unmarshal failed: %w\nData: %s", err, string(out))
	}
	//fmt.Println(result)

	// Doorloop alle block devices
	for _, device := range result.Blockdevices {
		//fmt.Println(device.Tran)
		if device.Tran != nil && *device.Tran == "usb" {
			fmt.Printf("USB device: %s is already connected\n", device.Name)
		}
		return true
	}
	fmt.Println("NO USB device found")
	return false

}
