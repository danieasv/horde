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

// CollectionKey is the identifer for Collection instances
type CollectionKey storageKey

// NewCollectionKeyFromString creates a new CollectionKey from a string
func NewCollectionKeyFromString(id string) (CollectionKey, error) {
	k, err := newKeyFromString(id)
	return CollectionKey(k), err
}

// String returns the string representation of the key
func (c CollectionKey) String() string {
	return storageKey(c).String()
}

// Collection is a collection of devices. The name might be a bit generic but
// we have only *one* type of collection in the system -- the one with devices.
type Collection struct {
	ID        CollectionKey
	TeamID    TeamKey
	FieldMask FieldMask
	Firmware  CollectionFirmwareMetadata
	Tags
}

// NewCollection creates a new collection
func NewCollection() Collection {
	return Collection{Tags: NewTags(), Firmware: NewCollectionFirmwareMetadata()}
}

// FirmwareManagementSetting is the firmware management setting for the collection
type FirmwareManagementSetting rune

// The various firmware management settings.
const (
	DisabledManagement   = FirmwareManagementSetting(' ')
	CollectionManagement = FirmwareManagementSetting('c')
	DeviceManagement     = FirmwareManagementSetting('d')
)

// CollectionFirmwareMetadata is the firmware settings for the collections. If
// the management is set to disabled or device the version fields will be ignored.
type CollectionFirmwareMetadata struct {
	CurrentFirmwareID FirmwareKey
	TargetFirmwareID  FirmwareKey
	Management        FirmwareManagementSetting
}

// NewCollectionFirmwareMetadata creates a new empty metadata setting
func NewCollectionFirmwareMetadata() CollectionFirmwareMetadata {
	return CollectionFirmwareMetadata{Management: DisabledManagement}
}
