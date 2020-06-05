package model

//
//Copyright 2019 Telenor Digital AS
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
import "time"

// DeviceKey is the identifier for Device instances
type DeviceKey storageKey

// NewDeviceKeyFromString creates a new DeviceKey  from a string
// representation
func NewDeviceKeyFromString(id string) (DeviceKey, error) {
	k, err := newKeyFromString(id)
	return DeviceKey(k), err
}

// String returns the string representation of the DeviceKey instance
func (d DeviceKey) String() string {
	return storageKey(d).String()
}

// Device is the entity that represent actual NB-IoT devices.
type Device struct {
	ID           DeviceKey
	IMSI         int64
	IMEI         int64
	CollectionID CollectionKey
	Network      DeviceNetworkMetadata
	Firmware     DeviceFirmwareMetadata
	Tags
}

// NewDevice creates a new empty device
func NewDevice() Device {
	return Device{Tags: NewTags(), Firmware: DeviceFirmwareMetadata{State: Unknown}}
}

// DeviceFirmwareState is the state for the firmware update cycle. If the device
// is in the TimedOut or Rollback state it won't be updated. The state is
// changed to "pending" whenever the target firmware id is set on the device
type DeviceFirmwareState rune

// Firmware states for devices
const (
	Unknown      = DeviceFirmwareState(' ') // Unknown state
	Current      = DeviceFirmwareState('c') // The firmware is current. This is the default state
	Initializing = DeviceFirmwareState('i') // The firmware update has started
	Pending      = DeviceFirmwareState('p') // A firmware update is pending
	Downloading  = DeviceFirmwareState('d') // Firmware is downloading to the device
	Completed    = DeviceFirmwareState('u') // Device has downloaded and written the firmware
	UpdateFailed = DeviceFirmwareState('f') // Update operation has failed
	TimedOut     = DeviceFirmwareState('t') // Update timed out
	Reverted     = DeviceFirmwareState('r') // Device was updated but did not report the updated version
)

// IsError returns true if the firmware state represents an error
func (d DeviceFirmwareState) IsError() bool {
	if d == UpdateFailed || d == TimedOut || d == Reverted {
		return true
	}
	return false
}

func (d DeviceFirmwareState) String() string {
	switch d {
	case Current:
		return "Current"
	case Initializing:
		return "Initializing"
	case Pending:
		return "Pending"
	case Downloading:
		return "Downloading"
	case Completed:
		return "Completed"
	case UpdateFailed:
		return "UpdateFailed"
	case TimedOut:
		return "TimedOut"
	case Reverted:
		return "Reverted"
	}
	return "Unknown"
}

// DeviceFirmwareMetadata contains internal metadata about devices. The information
// is for internal housekeeping mostly and *might* be relevant for devices.
// The information is pulled from the LwM2M responses from the device.
type DeviceFirmwareMetadata struct {
	CurrentFirmwareID FirmwareKey // The current firmware version installed. 0 if it isn't set or is unknown
	TargetFirmwareID  FirmwareKey // The desired version of the firmware for the device. It might be the same as the current version
	FirmwareVersion   string      // From the device information resource
	SerialNumber      string      // From the device information resource
	ModelNumber       string      // From the device information resource
	Manufacturer      string      // From the device information resource
	State             DeviceFirmwareState
	StateMessage      string
}

// DeviceNetworkMetadata is the current state of the device.
type DeviceNetworkMetadata struct {
	AllocatedIP string
	AllocatedAt time.Time
	CellID      int64
	ApnID       int
	NasID       int
}
