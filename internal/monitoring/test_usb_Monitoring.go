package monitoring

/*
#include <fcntl.h>
#include <sys/fanotify.h>
#include <unistd.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"os"
	"unsafe"
)

func MonitorUSB() {
	// Initialize fanotify (zonder O_LARGEFILE)
	fd, err := C.fanotify_init(C.FAN_CLASS_PRE_CONTENT|C.FAN_NONBLOCK, C.O_RDONLY)
	if fd == -1 {
		fmt.Printf("fanotify_init failed: %v\n", err)
		os.Exit(1)
	}
	defer C.close(fd)

	path := C.CString("/mnt/usb")
	defer C.free(unsafe.Pointer(path))

	ret, err := C.fanotify_mark(
		fd,
		C.FAN_MARK_ADD|C.FAN_MARK_MOUNT,
		C.FAN_OPEN|C.FAN_MODIFY,
		C.AT_FDCWD,
		path,
	)
	if ret == -1 {
		fmt.Printf("fanotify_mark failed: %v\n", err)
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
			fmt.Printf("Read detected from PID: %d\n", metadata.pid)
		}
		if metadata.mask&C.FAN_MODIFY != 0 {
			fmt.Printf("Write detected from PID: %d\n", metadata.pid)
		}
	}
}
