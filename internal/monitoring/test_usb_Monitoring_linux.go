package monitoring

/*
//#cgo pkg-config: libudev
//#include <libudev.h>
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
	"context"
	"encoding/json"
	"fmt"
	"github.com/Dogru-Isim/airgap-antivirus/internal/config"
	"github.com/Dogru-Isim/airgap-antivirus/internal/logging"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"unsafe"
)

func DetectingUSB(ctx context.Context) {
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

	C.udev_monitor_filter_add_match_subsystem_devtype(monitor, C.CString("usb"), nil)
	C.udev_monitor_enable_receiving(monitor)

	fmt.Println("Monitoring USB devices...")

	for {
		dev := C.udev_monitor_receive_device(monitor)
		if dev == nil {
			continue
		}

		action := C.GoString(C.udev_device_get_action(dev))
		devpath := C.GoString(C.udev_device_get_devpath(dev))
		vendor := C.GoString(C.udev_device_get_property_value(dev, C.CString("ID_VENDOR")))
		model := C.GoString(C.udev_device_get_property_value(dev, C.CString("ID_MODEL")))

		switch action {
		case "add":
			fmt.Printf("USB connected: %s %s (%s)\n", vendor, model, devpath)
			time.Sleep(5 * time.Second)
			mountPath, err := GetUSBPath()
			time.Sleep(5 * time.Second)
			if err != nil {
				fmt.Println("GetUSBPath() failed: ", err)
			}
			for _, path := range mountPath {
				MonitorUSB(path, ctx)
			}
		case "remove":
			fmt.Printf("USB disconnected: %s\n", devpath)
		}

		C.udev_device_unref(dev)
	}
}

func MonitorUSB(path string, ctx context.Context) {
	fmt.Println("helo")
	output, err := os.OpenFile(filepath.Join(config.Load().ExecutableLocation, "../../../"+config.Load().LogPath+"usb_traffic_json.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		fmt.Printf("cannot open log file")
		os.Exit(1)
	}
	logger, _ := logging.NewUSBLogger(
		logging.USBLoggerWithContext(ctx),
		logging.USBLoggerWithOutput(output),
	)

	// Initialize fanotify (zonder O_LARGEFILE)
	fd, err := C.fanotify_init(C.FAN_REPORT_FID|C.FAN_CLASS_NOTIF|C.FAN_NONBLOCK, C.O_RDONLY)
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
		C.FAN_ONDIR|C.FAN_CREATE|C.FAN_OPEN|C.FAN_MODIFY|C.FAN_EVENT_ON_CHILD,
		C.AT_FDCWD,
		cPath,
	)
	if ret == -1 {
		errno := C.getErrno()
		fmt.Printf("fanotify_mark failed: %v (errno=%d)\n", err, errno)
		os.Exit(1)
	}

	fmt.Println("Monitoring /mnt/usb...")
	for {
		buf := make([]byte, 4096)
		n, _ := C.read(C.int(fd), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
		if n <= 0 {
			continue
		}
		metadata := (*C.struct_fanotify_event_metadata)(unsafe.Pointer(&buf[0]))

		if metadata.mask&C.FAN_OPEN != 0 {
			logger.Log(slog.LevelInfo, logging.SuspicionLevelNormal, fmt.Sprintf("Read detected from PID in %s: %d\n", path, metadata.pid))
			//fmt.Printf("[%s]Open detected from PID: %d\n", path, metadata.pid)
		}
		if metadata.mask&C.FAN_CREATE != 0 {
			logger.Log(slog.LevelInfo, logging.SuspicionLevelSuspicious, fmt.Sprintf("Create detected from PID in %s: %d\n", path, metadata.pid))
			//fmt.Printf("[%s]Create detected from PID: %d\n", path, metadata.pid)
		}
		if metadata.mask&C.FAN_MODIFY != 0 {
			logger.Log(slog.LevelInfo, logging.SuspicionLevelSuspicious, fmt.Sprintf("Write detected from PID in %s: %d\n", path, metadata.pid))
			//fmt.Printf("[%s]Write detected from PID: %d\n", path, metadata.pid)
		}
	}
}

/////////////////////
////MOUNTPOINTS/////
///////////////////

type LSBLKOutput struct {
	Blockdevices []BlockDevice `json:"blockdevices"`
}

type BlockDevice struct {
	Name        string        `json:"name"`
	Mountpoints []string      `json:"mountpoints"`
	Tran        *string       `json:"tran"`
	Children    []BlockDevice `json:"children,omitempty"`
}

func GetUSBPath() ([]string, error) {
	// - lsblk -o NAME,MOUNTPOINT,TRAN | grep ' usb'|awk -F ' ' '{print $1}' //krijgt de fysieke usbs
	// - ls /dev/sdc* /// checkt of er partities zijn
	// - lsblk -o NAME,MOUNTPOINT,TRAN | grep 'sdc1'|awk -F ' ' '{print $2}' //checkt de mountpoints van de partities
	///kijk of de command gebruikt is
	cmd := exec.Command("lsblk", "-J", "-o", "NAME,MOUNTPOINTS,TRAN")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	var result LSBLKOutput
	if err := json.Unmarshal(out, &result); err != nil {
		panic(err)
	}
	fmt.Println(result)

	var allMountpoints []string

	// Doorloop alle block devices
	for _, device := range result.Blockdevices {
		if device.Tran != nil && *device.Tran == "usb" {
			fmt.Printf("USB device: %s\n", device.Name)
			for _, child := range device.Children {
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
