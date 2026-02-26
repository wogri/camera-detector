//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c -fmodules
#cgo LDFLAGS: -framework CoreMediaIO -framework CoreFoundation
#include <CoreMediaIO/CMIOHardware.h>
#include <CoreFoundation/CoreFoundation.h>

// Get a list of CMIO devices (both audio/video devices may appear). Returns up to maxCount devices.
static int GetDevices(CMIOObjectID* outDevices, uint32_t maxCount, uint32_t* outCount) {
    CMIOObjectID systemObject = kCMIOObjectSystemObject;
    CMIOObjectPropertyAddress addr = {
        .mSelector = kCMIOHardwarePropertyDevices,
        .mScope = kCMIOObjectPropertyScopeGlobal,
        .mElement = kCMIOObjectPropertyElementMain
    };

    UInt32 dataSize = 0;
    OSStatus err = CMIOObjectGetPropertyDataSize(systemObject, &addr, 0, NULL, &dataSize);
    if (err != noErr) return (int)err;

    UInt32 deviceCount = dataSize / sizeof(CMIOObjectID);
    if (deviceCount > maxCount) deviceCount = maxCount;

    err = CMIOObjectGetPropertyData(systemObject, &addr, 0, NULL, deviceCount * sizeof(CMIOObjectID), &dataSize, outDevices);
    if (err != noErr) return (int)err;

    *outCount = dataSize / sizeof(CMIOObjectID);
    return 0;
}

// Read the device's localized name into a provided buffer as UTF-8.
static int GetDeviceName(CMIOObjectID deviceId, char* buffer, size_t bufSize) {
    CMIOObjectPropertyAddress addr = {
        .mSelector = kCMIOObjectPropertyName,
        .mScope = kCMIOObjectPropertyScopeGlobal,
        .mElement = kCMIOObjectPropertyElementMain
    };

    CFStringRef name = NULL;
    UInt32 dataSize = sizeof(CFStringRef);
    OSStatus err = CMIOObjectGetPropertyData(deviceId, &addr, 0, NULL, dataSize, &dataSize, &name);
    if (err != noErr || name == NULL) return (int)((err != noErr) ? err : -1);

    Boolean ok = CFStringGetCString(name, buffer, (CFIndex)bufSize, kCFStringEncodingUTF8);
    CFRelease(name);
    if (!ok) return -2;
    return 0;
}

// Query whether the device is running somewhere (i.e., currently in use by any process).
static int GetDeviceIsRunningSomewhere(CMIOObjectID deviceId, uint32_t* outRunning) {
    CMIOObjectPropertyAddress addr = {
        .mSelector = kCMIODevicePropertyDeviceIsRunningSomewhere,
        .mScope = kCMIOObjectPropertyScopeGlobal,
        .mElement = kCMIOObjectPropertyElementMain
    };

    UInt32 isRunning = 0;
    UInt32 dataSize = sizeof(UInt32);
    OSStatus err = CMIOObjectGetPropertyData(deviceId, &addr, 0, NULL, dataSize, &dataSize, &isRunning);
    if (err != noErr) return (int)err;

    *outRunning = isRunning;
    return 0;
}

// Check if the device has video streams (to filter out audio-only devices)
static int HasVideoStreams(CMIOObjectID deviceId, uint32_t* outHasVideo) {
    CMIOObjectPropertyAddress addr = {
        .mSelector = kCMIODevicePropertyStreams,
        .mScope = kCMIODevicePropertyScopeInput,
        .mElement = kCMIOObjectPropertyElementMain
    };

    UInt32 dataSize = 0;
    OSStatus err = CMIOObjectGetPropertyDataSize(deviceId, &addr, 0, NULL, &dataSize);
    if (err != noErr) {
        *outHasVideo = 0;
        return 0; // Not an error, just no input streams
    }

    *outHasVideo = (dataSize > 0) ? 1 : 0;
    return 0;
}
*/
import "C"

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"unsafe"
)

type Device struct {
	ID       uint32
	Name     string
	Running  bool
	HasVideo bool
}

func listDevices() ([]Device, error) {
	// Allocate a reasonable buffer for devices
	const maxDevices = 128
	var cDevices [maxDevices]C.CMIOObjectID
	var cCount C.uint

	if rc := C.GetDevices(&cDevices[0], C.uint(maxDevices), &cCount); rc != 0 {
		return nil, fmt.Errorf("GetDevices failed with code %d", int(rc))
	}

	count := int(cCount)
	devs := make([]Device, 0, count)

	for i := 0; i < count; i++ {
		id := uint32(cDevices[i])

		// Get name
		nameBuf := make([]C.char, 512)
		var name string
		if rc := C.GetDeviceName(cDevices[i], (*C.char)(unsafe.Pointer(&nameBuf[0])), C.size_t(len(nameBuf))); rc == 0 {
			name = C.GoString((*C.char)(unsafe.Pointer(&nameBuf[0])))
		} else {
			// If we can't read name, skip
			continue
		}

		// Check if device has video streams
		var hasVideo C.uint
		C.HasVideoStreams(cDevices[i], &hasVideo)

		// Get running flag
		var running C.uint
		if rc := C.GetDeviceIsRunningSomewhere(cDevices[i], &running); rc != 0 {
			// If this fails, still include the device with Running=false
			devs = append(devs, Device{ID: id, Name: name, Running: false, HasVideo: hasVideo != 0})
			continue
		}

		devs = append(devs, Device{ID: id, Name: name, Running: running != 0, HasVideo: hasVideo != 0})
	}

	return devs, nil
}

func findStudioDisplayCamera(devs []Device) (Device, bool) {
	// Look for devices that contain "Studio Display" in their name and have video capability
	for _, d := range devs {
		if d.HasVideo && strings.Contains(strings.ToLower(d.Name), "studio display") {
			return d, true
		}
	}
	return Device{}, false
}

func findDeviceByName(devs []Device, needle string) (Device, bool) {
	n := strings.ToLower(needle)
	for _, d := range devs {
		if strings.Contains(strings.ToLower(d.Name), n) {
			return d, true
		}
	}
	return Device{}, false
}

func printDevices(devs []Device, videoOnly bool) {
	if len(devs) == 0 {
		fmt.Println("No CMIO devices found.")
		return
	}

	filteredDevs := devs
	if videoOnly {
		filteredDevs = make([]Device, 0)
		for _, d := range devs {
			if d.HasVideo {
				filteredDevs = append(filteredDevs, d)
			}
		}
	}

	if len(filteredDevs) == 0 {
		fmt.Println("No video devices found.")
		return
	}

	for _, d := range filteredDevs {
		state := "OFF"
		if d.Running {
			state = "ON"
		}
		videoType := ""
		if d.HasVideo {
			videoType = " [VIDEO]"
		}
		fmt.Printf("- %s%s  [%s]\n", d.Name, videoType, state)
	}
}

func executeCommand(command string, deviceName string, state string) {
	if command == "" {
		return
	}

	// Replace placeholders in the command
	cmd := strings.ReplaceAll(command, "{device}", deviceName)
	cmd = strings.ReplaceAll(cmd, "{state}", state)

	// Execute the command using shell
	execCmd := exec.Command("sh", "-c", cmd)
	if err := execCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command '%s': %v\n", cmd, err)
	}
}

func main() {
	name := flag.String("name", "", "Substring of the camera name to match (case-insensitive). If empty, defaults to Studio Display camera.")
	camera := flag.String("camera", "", "Alias for --name. Substring of the camera name to match (case-insensitive).")
	list := flag.Bool("list", false, "List all CMIO devices and their in-use status, then exit.")
	videoOnly := flag.Bool("video-only", false, "When listing, show only video devices.")
	watch := flag.Bool("watch", false, "Watch for status changes and print updates.")
	interval := flag.Duration("interval", 1*time.Second, "Polling interval when using --watch.")
	quiet := flag.Bool("quiet", false, "When not watching, print only ON or OFF without extra text.")
	onCommand := flag.String("on-command", "", "Command to execute when camera turns ON (only used with --watch). Use {device} and {state} as placeholders.")
	offCommand := flag.String("off-command", "", "Command to execute when camera turns OFF (only used with --watch). Use {device} and {state} as placeholders.")

	flag.Parse()

	// Use camera flag if name is not provided
	cameraName := *name
	if cameraName == "" && *camera != "" {
		cameraName = *camera
	}

	devs, err := listDevices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing devices: %v\n", err)
		os.Exit(2)
	}

	if *list {
		printDevices(devs, *videoOnly)
		return
	}

	var target Device
	var ok bool

	if cameraName == "" {
		// Default: look for Studio Display camera
		target, ok = findStudioDisplayCamera(devs)
		if !ok {
			fmt.Fprintf(os.Stderr, "No Studio Display camera found.\n")
			fmt.Fprintln(os.Stderr, "Tip: run with --list --video-only to see available video devices, or use --name/--camera to specify a different device.")
			os.Exit(1)
		}
	} else {
		// Look for user-specified device name
		target, ok = findDeviceByName(devs, cameraName)
		if !ok {
			fmt.Fprintf(os.Stderr, "No device found matching name substring: %q\n", cameraName)
			fmt.Fprintln(os.Stderr, "Tip: run with --list to see available devices.")
			os.Exit(1)
		}
	}

	if *watch {
		// Watch loop: report on state changes
		var last *bool

		// Execute command for initial state
		if target.Running {
			executeCommand(*onCommand, target.Name, "ON")
		} else {
			executeCommand(*offCommand, target.Name, "OFF")
		}

		for {
			devs, err := listDevices()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error refreshing devices: %v\n", err)
				time.Sleep(*interval)
				continue
			}

			var d Device
			var ok bool
			if cameraName == "" {
				d, ok = findStudioDisplayCamera(devs)
			} else {
				d, ok = findDeviceByName(devs, cameraName)
			}

			if !ok {
				fmt.Fprintf(os.Stderr, "Device disappeared\n")
				time.Sleep(*interval)
				continue
			}

			cur := d.Running
			if last == nil || *last != cur {
				state := "OFF"
				if cur {
					state = "ON"
					executeCommand(*onCommand, d.Name, state)
				} else {
					executeCommand(*offCommand, d.Name, state)
				}
				fmt.Printf("%s  [%s]\n", d.Name, state)
				last = &cur
			}
			time.Sleep(*interval)
		}
	} else {
		// One-shot
		if *quiet {
			if target.Running {
				fmt.Println("ON")
			} else {
				fmt.Println("OFF")
			}
		} else {
			state := "OFF"
			if target.Running {
				state = "ON"
			}
			fmt.Printf("%s is %s\n", target.Name, state)
		}
	}
}
