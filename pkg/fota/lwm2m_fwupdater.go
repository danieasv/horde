package fota

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
import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/go-ocf/go-coap/codes"

	"github.com/eesrc/horde/pkg/apn"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/eesrc/horde/pkg/fota/lwm2m"
	"github.com/eesrc/horde/pkg/fota/lwm2m/objects"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"

	"github.com/ExploratoryEngineering/logging"
	coap "github.com/go-ocf/go-coap"
)

type checkData struct {
	device        model.Device
	remotePort    int32
	remoteAddress net.IP
}

// fwUpdater does a version check on the device's firmware and
// device information before initiating an update. The updater is used by
// the LwM2M server
type fwUpdater struct {
	checkChan  chan checkData
	config     Parameters
	mutex      *sync.Mutex
	store      storage.DataStore
	coapServer *apn.RxTxReceiver
	inProgress map[int64]bool
}

// newFirmwareUpdater creates a new updater
func newFirmwareUpdater(store storage.DataStore, coapServer *apn.RxTxReceiver, config Parameters) *fwUpdater {
	ret := &fwUpdater{
		checkChan:  make(chan checkData),
		config:     config,
		store:      store,
		mutex:      &sync.Mutex{},
		coapServer: coapServer,
		inProgress: make(map[int64]bool),
	}
	go ret.checkLoop()
	return ret
}

// isChecking returns true if a firmware update check is in progress.
func (f *fwUpdater) isChecking(imsi int64) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	check, ret := f.inProgress[imsi]
	if check && ret {
		return true
	}
	return false
}

// addCheck flags a device upgrade as in progress. If an upgrade is in progress
// another one will not be started
func (f *fwUpdater) addCheck(imsi int64) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.inProgress[imsi] = true
}

// removeCheck removes the in progress flag from the device.
func (f *fwUpdater) removeCheck(imsi int64) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	delete(f.inProgress, imsi)
}

// CheckDeviceVersion retrieves the device version and checks if it requires
// a firmware update. It will enqueue the check and return immediately unless another
// check is already in progress.
func (f *fwUpdater) CheckDeviceVersion(device model.Device, remotePort int32, remoteAddress net.IP) {
	f.checkChan <- checkData{
		device:        device,
		remotePort:    remotePort,
		remoteAddress: remoteAddress,
	}
}

func (f *fwUpdater) checkLoop() {
	for c := range f.checkChan {
		f.checkFirmwareOnDevice(c.device, c.remotePort, c.remoteAddress)
	}
}

func (f *fwUpdater) queryDevice(path string, device model.Device, remotePort int32, remoteAddress net.IP) (objects.TLVBuffer, error) {
	ctx, done := context.WithTimeout(context.Background(), coapLwM2MTimeoutSeconds*time.Second)
	defer done()
	resp, err := f.coapServer.Exchange(ctx, &device, &rxtx.Message{
		RemotePort:    remotePort,
		RemoteAddress: remoteAddress,
		Type:          rxtx.MessageType_CoAPPush,
		Coap: &rxtx.CoAPOptions{
			Path:           path,
			Code:           int32(codes.GET),
			Accept:         int32(coap.AppLwm2mTLV),
			TimeoutSeconds: coapLwM2MTimeoutSeconds,
		},
	})
	if err != nil {
		return objects.TLVBuffer{}, err
	}
	logging.Debug("TLV payload size: %d bytes", len(resp.Payload))
	return objects.NewTLVBuffer(resp.Payload), nil
}

func (f *fwUpdater) flagDevice(device model.Device, newState model.DeviceFirmwareState, reason string) {
	logging.Info("Flagging device with IMSI %d with state %s and reason '%s'", device.IMSI, newState.String(), reason)
	device.Firmware.State = newState
	device.Firmware.StateMessage = reason
	if err := f.store.UpdateDeviceMetadata(device); err != nil {
		logging.Warning("Unable to update the device state for device with IMSI %d: %v", device.IMSI, err)
	}
}

// checkFirmwareOnDevice checks the version on the device vs the current
// version.
func (f *fwUpdater) checkFirmwareOnDevice(device model.Device, remotePort int32, remoteAddress net.IP) {
	if f.isChecking(device.IMSI) {
		logging.Warning("Device with IMSI %d is already doing a firmware upgrade.", device.IMSI)
		return
	}
	fu := objects.FirmwareUpdate{}

	// Get state of firmware update first. if it is updating skip check
	tlv, err := f.queryDevice(lwm2m.FirmwareStatePath, device, remotePort, remoteAddress)
	if err != nil {
		logging.Warning("Unable to query firmware state (%s) for device with IMSI %d: %v", lwm2m.FirmwareStatePath, device.IMSI, err)
		f.flagDevice(device, model.UpdateFailed, "Device did not respond to LwM2M firmware state request (/5/0/3)")
		return
	}
	fu.SetState(tlv)
	logging.Info("Device state is %s (%d)", fu.State.String(), fu.State)
	if fu.State != objects.Idle && fu.State != objects.Downloaded {
		logging.Warning("Device with IMSI %d has firmware state set to other than Idle or Downloaded (%d). Won't update", device.IMSI, fu.State)
		f.flagDevice(device, model.UpdateFailed, fmt.Sprintf("Device is reporting update state as %s. It must be Idle  or Downloaded for firmware updates", fu.State))
		return
	}

	// The Zephyr LwM2M client uses the device information object to
	// report the version. Use the FirmwareVersion field for the version.
	tlv, err = f.queryDevice(lwm2m.DeviceInformationPath, device, remotePort, remoteAddress)
	if err != nil {
		logging.Warning("Unable to query device information (%s) for device with IMSI %d: %v", lwm2m.DeviceInformationPath, device.IMSI, err)
		f.flagDevice(device, model.UpdateFailed, "Device did not respond to LwM2M device information request (/3/0)")
		return
	}

	di := objects.NewDeviceInformation(tlv)

	logging.Debug("Device info: %+v", di)
	report := Report{
		FirmwareVersion:  di.FirmwareVersion,
		ManufacturerName: di.Manufacturer,
		SerialNumber:     di.SerialNumber,
		ModelNumber:      di.ModelNumber,
	}

	needsUpdate, firmwareID, err := firmwareUpdateCheck(&device, report, f.store)
	if err != nil {
		logging.Warning("Unable to check firmware via gRPC for IMSI %d. Ignoring: %v", device.IMSI, err)
		return
	}
	if !needsUpdate {
		logging.Info("No firmware update scheduled for device with IMSI %d", device.IMSI)
		return
	}
	logging.Info("Device with IMSI %d is is scheduled to get new firmware (%s)", device.IMSI, firmwareID.String())
	switch fu.State {
	case objects.Idle:
		f.flagDevice(device, model.Initializing, "Update is started")
	case objects.Downloaded:
		f.flagDevice(device, model.Completed, "Image is downloaded")
	}
	go f.updateFirmware(fu.State, device, remotePort, remoteAddress, firmwareID)

}

// The LwM2M library doesn't work with push which is OK - just use pull delivery
// It might be possible to get it to work but the delivery methods are quite
// similar from our point of view. This method blocks until the firmware update
// is complete (ie success or failure)
func (f *fwUpdater) updateFirmware(currentState objects.FirmwareUpdateState, device model.Device, remotePort int32, remoteAddress net.IP, firmwareID model.FirmwareKey) {
	f.addCheck(device.IMSI)
	defer f.removeCheck(device.IMSI)
	if currentState != objects.Downloaded {
		if !f.startDownload(device, remoteAddress, remotePort) {
			// Couldn't initialize download. Stop
			return
		}
	}

	// We *could* use observation here but we'll just poll once every minute until it reports
	// state "downloaded" or it times out. The Zephyr implementation reports every 30 seconds
	// but once the observation is turned on it must be cancelled to release the connection.
	start := time.Now()

	const statePollInterval = 5 * time.Second

	// The state is set implicitly by checking the download state. There's no
	// point polling the device when we already know what it is doing.
	state := model.Downloading
	for state != model.Completed {
		time.Sleep(statePollInterval)
		dev, err := f.store.RetrieveDeviceByIMSI(device.IMSI)
		if err != nil {
			logging.Warning("Error retrieving device with IMSI %d: %v", device.IMSI, err)
			continue
		}
		state = dev.Firmware.State
		logging.Debug("Device with IMSI %d has state %s", device.IMSI, state.String())
		if state.IsError() {
			logging.Warning("Download error for device with IMSI %d (%s). Stopping upgrade.", device.IMSI, state.String())
			return
		}
	}

	durationSec := float64(time.Since(start)) / float64(time.Second)

	logging.Info("Firmware has downloaded to device with IMSI %d in %3.2fs", device.IMSI, durationSec)

	// TODO: Get the update result before continuing here. Alternatively check what the
	// client sends. It might not send the Idle/Downloading/Downloaded/Updating state
	// correctly so we're ignoring it here.
	logging.Debug("Requesting update (POST to %s) for device with IMSI %d", lwm2m.FirmwareUpdatePath, device.IMSI)

	// This will need a new context since this is a new request
	ctx2, done2 := context.WithTimeout(context.Background(), coapLwM2MTimeoutSeconds*time.Second)
	defer done2()
	_, err := f.coapServer.Exchange(ctx2, &device, &rxtx.Message{
		Type:          rxtx.MessageType_CoAPPush,
		RemoteAddress: remoteAddress,
		RemotePort:    remotePort,
		Payload:       make([]byte, 1), // Payload is required, even when it is 0 bytes
		Coap: &rxtx.CoAPOptions{
			Code:           int32(codes.POST),
			Path:           lwm2m.FirmwareUpdatePath,
			ContentFormat:  int32(coap.AppOctets),
			TimeoutSeconds: coapLwM2MTimeoutSeconds,
		},
	})
	if err != nil {
		logging.Warning("Unable to trigger firmware update for device with IMSI %d: %v", device.IMSI, err)
		f.flagDevice(device, model.UpdateFailed, "Could not trigger update on device (/5/0/2)")
		return
	}

	f.flagDevice(device, model.Completed, "Device has downloaded firmware image and is performing update")

}

// startDownload initiates a download on the device via the LwM2M update object
func (f *fwUpdater) startDownload(device model.Device, remoteAddress net.IP, remotePort int32) bool {
	logging.Debug("Pointing to firmware image at \"%s\"", f.config.FirmwareEndpoint)
	// Flag the device with "downloading" in case it starts right away. If it
	// fails it will be flagged with an error. There's an extra access here but
	// only when in error mode.
	f.flagDevice(device, model.Downloading, "Waiting for device to download firmware image")
	buf := objects.EncodeString(1, f.config.FirmwareEndpoint)
	ctx, done := context.WithTimeout(context.Background(), coapLwM2MTimeoutSeconds*time.Second)
	defer done()
	res, err := f.coapServer.Exchange(ctx, &device, &rxtx.Message{
		Payload:       buf,
		RemoteAddress: remoteAddress,
		RemotePort:    remotePort,
		Type:          rxtx.MessageType_CoAPPush,
		Coap: &rxtx.CoAPOptions{
			Code:           int32(codes.PUT),
			Path:           lwm2m.FirmwareImageURIPath,
			ContentFormat:  int32(coap.AppLwm2mTLV),
			Accept:         int32(coap.AppLwm2mTLV),
			TimeoutSeconds: coapLwM2MTimeoutSeconds,
		},
	})
	if err != nil {
		logging.Warning("Error performing CoAP request to device with IMSI %d: %v", device.IMSI, err)
		f.flagDevice(device, model.UpdateFailed, "Unable to set image URI on device (/5/0/1)")
		return false
	}
	if codes.Code(res.Coap.Code) == codes.NotFound {
		logging.Warning("Device does not know how to handle path %s", lwm2m.FirmwareImageURIPath)
		f.flagDevice(device, model.UpdateFailed, "Device responded with NotFound for image URI (/5/0/1)")
		return false
	}

	// Success - device should start downloading the image now
	return true
}
