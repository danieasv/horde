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

// FirmwareKey is the identifier for firmware images
type FirmwareKey storageKey

// NewFirmwareKeyFromString converts a string into a FirmwareKey
func NewFirmwareKeyFromString(id string) (FirmwareKey, error) {
	k, err := newKeyFromString(id)
	return FirmwareKey(k), err
}

func (f FirmwareKey) String() string {
	return storageKey(f).String()
}

// Firmware is the metadata for a firmware image. Most fields are immutable,
// except the TeamID and Tags.
type Firmware struct {
	ID           FirmwareKey // Image ID
	Version      string      // Unique for collection
	Filename     string      // Original file name - for informational purposes only
	Length       int         // Size of image (in bytes)
	SHA256       string      // SHA256 checksum of image. Computed when uploading. 64 hex characters (and 32 byte)
	Created      time.Time   // Time the image was created
	CollectionID CollectionKey
	Tags
}

// NewFirmware creates a new empty Firmware instance
func NewFirmware() Firmware {
	return Firmware{Tags: NewTags()}
}

// FirmwareConfig is a meta-type for firmware config; the fields are gathered
// into one type for convenience.
type FirmwareConfig struct {
	Management               FirmwareManagementSetting
	CollectionCurrentVersion FirmwareKey
	CollectionTargetVersion  FirmwareKey
	DeviceCurrentVersion     FirmwareKey
	DeviceTargetVersion      FirmwareKey
}

// CurrentVersion returns the currently running firmware version (or the one
// it should be running right now)
func (f *FirmwareConfig) CurrentVersion() FirmwareKey {
	switch f.Management {
	case DeviceManagement:
		return f.DeviceCurrentVersion
	case CollectionManagement:
		return f.CollectionCurrentVersion
	default:
		return FirmwareKey(0)
	}
}

// TargetVersion runs the target firmware version
func (f *FirmwareConfig) TargetVersion() FirmwareKey {
	switch f.Management {
	case DeviceManagement:
		return f.DeviceTargetVersion
	case CollectionManagement:
		return f.CollectionTargetVersion
	default:
		return FirmwareKey(0)
	}
}

// NeedsUpgrade returns true if the firmware should be upgraded
func (f *FirmwareConfig) NeedsUpgrade() bool {
	return f.TargetVersion() != f.CurrentVersion()
}

// FirmwareUse shows the firmware images in use for a collection
type FirmwareUse struct {
	FirmwareID FirmwareKey
	Current    []DeviceKey
	Targeted   []DeviceKey
}
