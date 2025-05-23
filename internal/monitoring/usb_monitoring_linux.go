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
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Dogru-Isim/airgap-antivirus/internal/logging"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"
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

/*
@note: the fd field is an int32 for conveniency in testing (cgo is not supported in with go tests)
this field is converted to a C.int and used with the fanotify interface from C which uses C's default 32 bit integer
*/
type USBMonitor struct {
	Mountpath string
	fd        int32
	logger    logging.USBLogger
}

// func main() {
// 	newUSBDetector := USBDetector{}
// 	fmt.Println("Starting to monitor for USBs:")
// 	for {
// 		newUSBDetector.DetectNewUSB()
// 		//fmt.Println("hi")
// 		newUSBDetector.USBDifferenceChecker()
// 		// voor elke mountpoint -> monitor.NewUSBMonitor() -> monitor.Start()
// 		if newUSBDetector.NewUSB != nil {
// 			for _, usb := range newUSBDetector.NewUSB {
// 				for _, partition := range usb.Partitions {
// 					for _, mountpoint := range partition.Mountpoints {
// 						monitor, err := NewUSBMonitor(mountpoint)
// 						if err == nil {
// 							fmt.Printf("%s is starting\n", monitor.Mountpath)
// 							go monitor.Start(context.Background())
// 						}
// 					}
// 				}
// 			}
// 			newUSBDetector.NewUSB = nil
// 		}
// 		time.Sleep(2 * time.Second)

// 	}

// }
func NewUSBDetector() *USBDetector {
	return &USBDetector{
		Blockdevices: make([]BlockDevice, 0),
		NewUSB:       make([]USB, 0),
		OutputUSB:    make([]USB, 0),
		ExistingUSB:  make([]USB, 0),
	}
}
func (u USB) Key() string {
	// Verzamel alle mountpoints van alle partities
	var mountpoints []string
	for _, partition := range u.Partitions {
		mountpoints = append(mountpoints, partition.Mountpoints...)
	}

	sort.Strings(mountpoints)
	hasher := sha1.New()
	hasher.Write([]byte(strings.Join(mountpoints, "|"))) //  | = scheidingsteken
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Combineer met naam en aantal partities
	return fmt.Sprintf("%s-%d-%s", u.Name, len(u.Partitions), hash)
}
func (u *USBDetector) USBDifferenceChecker() {
	if u.OutputUSB != nil {
		if u.ExistingUSB != nil {
			existingMap := make(map[string]bool)
			for _, usb := range u.ExistingUSB {
				existingMap[usb.Key()] = true
			}

			var newUSBs []USB
			for _, usb := range u.OutputUSB {
				if !existingMap[usb.Key()] {
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

type FanotifyInitializer interface {
	Initialize(flags, event_flags uint) (int32, error)
}

type Fanotify struct{}

// Return value is hardcoded because the reason why this returns an interface is to ease unit testing (dependency injection)
func NewFanotifyInitializer() FanotifyInitializer {
	return &Fanotify{}
}

func (f *Fanotify) Initialize(flags, event_flags uint) (int32, error) {
	fd, err := C.fanotify_init(C.uint(flags), C.uint(event_flags))
	if fd == -1 {
		return -1, fmt.Errorf("fanotify_init failed: %v", err)
	}
	return int32(fd), nil
}

func NewUSBMonitor(mountpath string, fanotify FanotifyInitializer) (*USBMonitor, error) {
	fileInfo, err := os.Stat(mountpath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("%s does not exist!\n", mountpath)
		//os.Exit(1)
	}
	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("%s is Not a directory!\n", mountpath)
		//os.Exit(1)
	}

	fd, err := fanotify.Initialize(
		C.FAN_REPORT_FID|C.FAN_REPORT_DFID_NAME|C.FAN_CLASS_NOTIF|C.FAN_NONBLOCK,
		C.O_RDONLY,
	)
	if err != nil {
		return nil, fmt.Errorf("fanotify initialization failed: %w", err)
	}

	logger, err := logging.NewUSBLogger()
	if err != nil {
		return nil, fmt.Errorf("func NewUSBMonitor: error while fetching logger")
	}

	return &USBMonitor{
		Mountpath: mountpath,
		fd:        fd,
		logger:    logger,
	}, nil
}

func (m *USBMonitor) Start(ctx context.Context) {

	cPath := C.CString(m.Mountpath)
	defer C.free(unsafe.Pointer(cPath))

	ret, err := C.fanotify_mark(
		C.int(m.fd),
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
		foundMountpoint, err := m.mountpointChecker()
		if err != nil && !foundMountpoint {
			fmt.Printf("Stopped monitoring on %s\n", m.Mountpath)
			defer C.close(C.int(m.fd))
			return
		}

		buf := make([]byte, 4096)
		n, _ := C.read(C.int(m.fd), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
		if n <= 0 {
			continue
		}
		metadata := (*C.struct_fanotify_event_metadata)(unsafe.Pointer(&buf[0]))

		msg := m.convertUSBAction(metadata)
		if msg != "" {
			m.logger.Log(slog.LevelInfo,
				fmt.Sprintf("[%s][%s] %s detected from PID: %d\n",
					time.Now().Format("15:04:05"),
					m.Mountpath,
					msg,
					metadata.pid),
			)
		}

	}
}

func (m *USBMonitor) mountpointChecker() (bool, error) {
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

/*
@brief Convert USB action from the fanotify interface to string values

@param (metadata *C.struct_fanotify_event_metadata): pointer to fanotify event metadata

@return string: converted string value, empty string on failure

@example:

	buf := make([]byte, 4096)
	n, _ := C.read(C.int(m.fd), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	if n <= 0 {
		continue
	}
	metadata := (*C.struct_fanotify_event_metadata)(unsafe.Pointer(&buf[0]))

	actionMsg := monitor.ConvertUSBAction(metadata)
	if actionMsg != "" {
		// do logging etc.
	}
*/
func (m *USBMonitor) convertUSBAction(metadata *C.struct_fanotify_event_metadata) string {
	var msg string
	switch {
	case metadata.mask&C.FAN_OPEN != 0:
		msg = "Open"
	case metadata.mask&C.FAN_CREATE != 0:
		msg = "Create"
	case metadata.mask&C.FAN_DELETE != 0:
		msg = "Delete"
	case metadata.mask&C.FAN_MOVED_FROM != 0:
		msg = "Moved FROM"
	case metadata.mask&C.FAN_MOVED_TO != 0:
		msg = "Moved TO"
	case metadata.mask&C.FAN_RENAME != 0:
		msg = "Rename"
	case metadata.mask&C.FAN_ATTRIB != 0:
		msg = "Attribute change"
	case metadata.mask&C.FAN_CLOSE_WRITE != 0:
		msg = "Write close"
	case metadata.mask&C.FAN_CLOSE_NOWRITE != 0:
		msg = "Read close"
	case metadata.mask&C.FAN_MODIFY != 0:
		msg = "Write"
	default:
		msg = ""
	}

	return msg
}
