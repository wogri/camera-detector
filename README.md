# Studio Display Camera Detector

A Go program for macOS that detects whether your Studio Display's built-in camera is currently active using modern AVFoundation APIs.

## Features

- Uses AVFoundation (no deprecated APIs)
- Detect if the Studio Display camera is ON or OFF
- List all available video devices
- Watch mode for real-time monitoring
- Quiet mode for scripting
- Custom device name matching

## Build and Run

```bash
# Build the program
go build -o studio-cam-detector studio-cam-detector.go

# Check Studio Display camera status
./studio-cam-detector

# List all video devices
./studio-cam-detector --list

# Monitor for changes
./studio-cam-detector --watch

# Quiet mode (outputs only ON/OFF)
./studio-cam-detector --quiet
```

## Usage Options

- `--list`: List all video devices and exit
- `--watch`: Monitor for status changes
- `--quiet`: Output only ON/OFF
- `--name <substring>`: Match devices by name
- `--interval <duration>`: Polling interval for watch mode

## How It Works

Uses macOS's modern AVFoundation framework to:
1. Enumerate video capture devices
2. Check each device's `isInUseByAnotherApplication` property
3. Match devices by name (defaults to "Studio Display")

This approach uses current APIs and avoids deprecated CoreMediaIO functions.

## Examples

Check if your Studio Display camera is active:
```bash
./studio-cam-detector
# Output: Studio Display Camera is OFF
```

Use in a script:
```bash
if [ "$(./studio-cam-detector --quiet)" = "ON" ]; then
    echo "Camera is active!"
fi
```

Monitor for changes:
```bash
./studio-cam-detector --watch
# Output: Studio Display Camera  [OFF]
# (when camera turns on)
# Output: Studio Display Camera  [ON]
```

## Automatic Startup at Login

You can set up the studio camera detector to automatically start when you log in using macOS's launchd system. A launchd plist file (`com.wogri.studio-cam-detector.plist`) is included in the project.

### Setup Instructions (No Root Privileges Required)

1. **Copy the plist file to your user's LaunchAgents directory:**
   ```bash
   cp com.wogri.studio-cam-detector.plist ~/Library/LaunchAgents/
   ```

2. **Load the service:**
   ```bash
   launchctl load ~/Library/LaunchAgents/com.wogri.studio-cam-detector.plist
   ```

3. **Start the service immediately (optional):**
   ```bash
   launchctl start com.wogri.studio-cam-detector
   ```

### Managing the Service

**Check if the service is running:**
```bash
launchctl list | grep com.wogri.studio-cam-detector
```

**Stop the service:**
```bash
launchctl stop com.wogri.studio-cam-detector
```

**Unload the service:**
```bash
launchctl unload ~/Library/LaunchAgents/com.wogri.studio-cam-detector.plist
```

**View logs:**
```bash
tail -f logs/studio-cam-detector.log
tail -f logs/studio-cam-detector.error.log
```

### What the Service Does

The service automatically:
- Starts the camera detector when you log in
- Runs the `startup.sh` script which monitors your Studio Display camera
- Executes `hass_off.sh` when the camera turns off
- Executes `hass_on.sh` when the camera turns on
- Keeps the service running and restarts it if it crashes
- Logs output to files in the `logs/` directory

This is perfect for integrating with home automation systems or other workflows that need to respond to camera state changes.
