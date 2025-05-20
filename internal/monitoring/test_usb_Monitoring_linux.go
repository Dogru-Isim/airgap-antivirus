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
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"unsafe"
)

type Monitor struct {
	fd   C.int
	path string
}

func DetectingUSB() {
	processedDevices := make(map[string]bool)   // mountpoint-active state
	deviceMounts := make(map[string]string)     // devpath- mountpoint
	activeMonitors := make(map[string]*Monitor) // path - monitor ponter

	// Check eerst al aangesloten USB-apparaten
	if CheckPluggedUSB() {
		mountPaths, err := GetUSBPath()
		if err != nil {
			fmt.Printf("Error getting USB paths: %v\n", err)
		} else {
			for device, path := range mountPaths {
				go MonitorUSB(path, activeMonitors)
				processedDevices[path] = true
				devpath, err := getDevpath(device)
				fmt.Println("hello", devpath)
				if err != nil {
					fmt.Errorf("Was not able to get devpath: %w", err)
				}
				deviceMounts[devpath] = path
			}
		}
	}

	// Start udevadm monitor voor USB events
	cmd := exec.Command("udevadm", "monitor", "--kernel", "--subsystem-match=usb", "--property")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error creating stdout pipe:", err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting udevadm:", err)
		return
	}
	defer cmd.Process.Kill()

	scanner := bufio.NewScanner(stdout)
	fmt.Println("Monitoring USB devices...")

	var currentEvent map[string]string
	for scanner.Scan() {
		line := scanner.Text()

		// Begin van een nieuw event
		if strings.HasPrefix(line, "KERNEL[") {
			currentEvent = make(map[string]string)
		} else if strings.Contains(line, "=") && currentEvent != nil {
			// Parse key-value paren
			parts := strings.SplitN(line, "=", 2)
			currentEvent[parts[0]] = parts[1]
		} else if line == "" && currentEvent != nil {
			// Verwerk het event
			action := currentEvent["ACTION"]
			devpath := currentEvent["DEVPATH"]

			if action == "add" {
				vendor, model := getUSBInfo(devpath) // Gebruik udevadm info
				fmt.Printf("USB connected: %s %s (%s)\n", vendor, model, devpath)

				maxattempt := 10
				mountPaths := make(map[string]string)
				var err error

				for attempt := 1; attempt <= maxattempt; attempt++ {
					mountPaths, err = GetUSBPath()
					fmt.Println(mountPaths)
					fmt.Println(err)
					if err == nil && len(mountPaths) > 0 {
						break
					}
					if err != nil {
						fmt.Printf("Attempt %d failed:%v\n", attempt, err)
					}
					fmt.Printf("Next attempt will begin in 2 seconds\n")
					time.Sleep(2 * time.Second)
				}

				for _, path := range mountPaths {
					if processedDevices[path] {
						fmt.Printf("%s is already being monitored\n", path)
						continue
					}
					go MonitorUSB(path, activeMonitors)
					processedDevices[path] = true
					deviceMounts[devpath] = path
				}

			} else if action == "remove" {
				fmt.Printf("USB disconnected: %s\n", devpath)
				if path, exists := deviceMounts[devpath]; exists {
					delete(processedDevices, path)
					delete(deviceMounts, devpath)
					fmt.Printf("delete succeeded")
					if monitor, ok := activeMonitors[path]; ok {
						C.close(monitor.fd) // Sluit de file descriptor
						delete(activeMonitors, path)
						fmt.Printf("Stopped with monitoring for: %s\n", path)
					}
				}
			}

			currentEvent = nil
		}
	}
}

func getDevpath(device string) (string, error) {
	// Build command
	cmd := exec.Command("udevadm", "info", "--query=all", "--name="+device)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error running udevadm: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "E: DEVPATH=") {

			return strings.TrimPrefix(line, "E: DEVPATH="), nil
		}
	}

	return "", fmt.Errorf("DEVPATH not found for %s", device)
}

// Helper om USB-info op te halen met udevadm
func getUSBInfo(devpath string) (string, string) {
	cmd := exec.Command("udevadm", "info", "--query=property", "--path="+devpath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown", "unknown"
	}

	props := make(map[string]string)
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			props[parts[0]] = parts[1]
		}
	}

	return props["ID_VENDOR"], props["ID_MODEL"]
}

func MonitorUSB(path string, activeMonitors map[string]*Monitor) {
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
		//os.Exit(1)
	}
	if !fileInfo.IsDir() {
		fmt.Printf("%s is Not a directory!\n", path)
		//os.Exit(1)
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
		//os.Exit(1)
	}

	activeMonitors[path] = &Monitor{fd: fd, path: path}
	defer delete(activeMonitors, path) // Verwijder bij exit

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

func GetUSBPath() (map[string]string, error) {

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

	var allMountDevice = make(map[string]string)
	// Doorloop alle block devices
	for _, device := range result.Blockdevices {
		if device.Tran != nil && *device.Tran == "usb" {
			fmt.Printf("USB device: %s\n", device.Name)
			for _, child := range device.Children {
				//fmt.Println("Hello")
				if child.Mountpoints != nil {
					for _, mountpoint := range child.Mountpoints {
						if mountpoint != "" {
							allMountDevice[child.Name] = mountpoint
							fmt.Printf("  Partitie: %s, Mountpoint: %s\n", child.Name, mountpoint)
						}
					}
					fmt.Println(allMountDevice)
				} else {
					fmt.Printf("  Partitie: %s, geen mountpoint\n", child.Name)
				}
			}
		}
	}
	if len(allMountDevice) == 0 {
		return nil, fmt.Errorf("mountpoint is empty")
	}
	return allMountDevice, nil

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
