package fota

//
// Copyright 2020 Telenor Digital AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
import (
	"errors"
	"io/ioutil"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// Do an update check on the firmware
func firmwareUpdateCheck(device *model.Device, data Report, firmwareStore storage.DataStore) (bool, model.FirmwareKey, error) {
	updateDevice := false

	if data.FirmwareVersion != "" {
		fw, err := firmwareStore.RetrieveFirmwareByVersion(device.CollectionID, data.FirmwareVersion)
		// Not found errors are OK since there might not be a firmware image
		// that matches the reported version. Other errors aren't OK.
		if err != nil && err != storage.ErrNotFound {
			logging.Warning("Unable to look up the firmware version device with IMSI %d: %v", device.IMSI, err)
			return false, 0, err
		}
		if err == nil && device.Firmware.CurrentFirmwareID != fw.ID {
			logging.Debug("Updating firmware to matching version %s for device with IMSI %d", fw.ID.String(), device.IMSI)
			updateDevice = true
			device.Firmware.CurrentFirmwareID = fw.ID
		}
		if err == storage.ErrNotFound {
			logging.Debug("Unknown firmware version (%s) on device with IMSI %d. Setting to 0", data.FirmwareVersion, device.IMSI)
			updateDevice = true
			device.Firmware.CurrentFirmwareID = 0
		}
	}

	// Retrieve collection to check if firmware updates is managed by collection
	config, err := firmwareStore.RetrieveFirmwareConfig(device.CollectionID, device.ID)
	if err != nil {
		logging.Warning("Could not retrieve firmware config for device with IMSI %d: %v", device.IMSI, err)
		return false, 0, err
	}
	if config.Management == model.DisabledManagement {
		logging.Info("Device with IMSI %d disabled firmware management. Won't update", device.IMSI)
		return false, 0, errors.New("fota disabled")
	}
	// Mark as reverted if the device have been updated but the reported
	// version is the same as before.
	if data.FirmwareVersion == device.Firmware.FirmwareVersion && device.Firmware.State == model.Completed {
		logging.Debug("Firmware version not the expected one on device with IMSI %d. Expected something different from %s but it did not change. Marking as reverted",
			device.IMSI, data.FirmwareVersion)
		device.Firmware.State = model.Reverted
		updateDevice = true
	}
	if data.FirmwareVersion != device.Firmware.FirmwareVersion {
		device.Firmware.FirmwareVersion = data.FirmwareVersion
		updateDevice = true
	}
	if data.ManufacturerName != device.Firmware.Manufacturer {
		device.Firmware.Manufacturer = data.ManufacturerName
		updateDevice = true
	}
	if data.SerialNumber != device.Firmware.SerialNumber {
		device.Firmware.SerialNumber = data.SerialNumber
		updateDevice = true
	}
	if data.ModelNumber != device.Firmware.ModelNumber {
		device.Firmware.ModelNumber = data.ModelNumber
		updateDevice = true
	}

	// Mark state as current if the firmware is up to date
	if config.TargetVersion() == device.Firmware.CurrentFirmwareID && device.Firmware.State != model.Current {
		logging.Debug("Marking firmware as Current (State=%s, ID=%s, Version=%s) for device with IMSI %d",
			device.Firmware.State.String(), device.Firmware.CurrentFirmwareID.String(),
			device.Firmware.FirmwareVersion, device.IMSI)
		device.Firmware.State = model.Current
		device.Firmware.StateMessage = ""
		updateDevice = true
	}

	if updateDevice {
		if err := firmwareStore.UpdateDeviceMetadata(*device); err != nil {
			logging.Warning("Unable to update device with IMSI %d: %v", device.IMSI, err)
			return false, 0, err
		}
		logging.Info("Updated device with IMSI %d with new version information", device.IMSI)
	}

	// Check if the current version matches and if there's a new version set for the device
	if config.TargetVersion() == 0 || device.Firmware.CurrentFirmwareID == config.TargetVersion() {
		logging.Debug("No upgrade for device with IMSI %d. CurrentFirmwareID=%s, TargetFirmwareID=%s",
			device.IMSI, config.CurrentVersion().String(), config.TargetVersion().String())
		// Version is the same. No need for update
		return false, 0, nil
	}

	// If the device has failed the previous attempt skip the update
	if device.Firmware.State.IsError() {
		logging.Info("Device with IMSI %d has failed a previous attempt. Won't update again", device.IMSI)
		return false, 0, errors.New("device is in error state")
	}

	return true, config.TargetVersion(), nil
}

// findFirmware returns the latest firmware image for the device
func findFirmware(device *model.Device, store storage.DataStore, firmwareStore storage.FirmwareImageStore) ([]byte, bool) {

	// Check the collection
	config, err := store.RetrieveFirmwareConfig(device.CollectionID, device.ID)
	if err != nil {
		logging.Warning("Unable to locate firmware config for device with IMSI %d; %v", device.IMSI, err)
		return nil, false
	}
	logging.Debug("Firmware config for device with IMSI %d: %+v", device.IMSI, config)
	if !config.NeedsUpgrade() {
		return nil, false
	}
	image, err := firmwareStore.Retrieve(config.TargetVersion())
	if err != nil {
		if err != storage.ErrNotFound {
			logging.Warning("Unable to locate image %s for device with IMSI %d: %v", device.Firmware.TargetFirmwareID, device.IMSI, err)
		}
		return nil, false
	}
	defer image.Close()
	buf, err := ioutil.ReadAll(image)
	if err != nil {
		logging.Warning("Unable to read firmware image %s for device with IMSI %d: %v", device.Firmware.TargetFirmwareID, device.IMSI, err)
		return nil, false
	}
	return buf, true
}
