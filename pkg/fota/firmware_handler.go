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
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/apn"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/go-ocf/go-coap"
	"github.com/go-ocf/go-coap/codes"
)

// firmwareHandler is the handler that returns the latest firmware image for the
// device. If there's no firmware assigned to the device the handler returns
// a not found response.
func newFirmwareHandler(receiver *apn.RxTxReceiver, timeout time.Duration, store storage.DataStore, firmwareStore storage.FirmwareImageStore) apn.CoAPHandler {
	return func(apnID int, nasID int, device *model.Device, req *rxtx.UpstreamRequest) (*rxtx.DownstreamResponse, apn.ResponseCallback, error) {
		notFound := &rxtx.DownstreamResponse{
			Msg: &rxtx.Message{
				Coap: &rxtx.CoAPOptions{
					Code: int32(codes.NotFound),
				},
			},
		}
		image, ok := findFirmware(device, store, firmwareStore)
		if !ok {
			logging.Info("Device with IMSI %d requested firmware image but none exists", device.IMSI)
			return notFound, nil, nil
		}

		logging.Debug("Sending firmware (%d bytes) to device with IMSI %d", len(image), device.IMSI)
		// success -- found firmware. Transfer to client
		ret := &rxtx.DownstreamResponse{
			Msg: &rxtx.Message{
				Type:    rxtx.MessageType_CoAPPull,
				Payload: image,
				Coap: &rxtx.CoAPOptions{
					Code:           int32(codes.Content),
					ContentFormat:  int32(coap.AppOctets),
					TimeoutSeconds: int32(timeout / time.Second),
				},
			},
		}
		store.UpdateFirmwareStateForDevice(device.IMSI, model.Downloading, "Device is downloading firmware image")

		callback := func(result rxtx.ErrorCode) {
			if result != rxtx.ErrorCode_SUCCESS {
				logging.Warning("Could not write the firmware image to device with IMSI %d: %v", device.IMSI, result.String())
				store.UpdateFirmwareStateForDevice(device.IMSI, model.UpdateFailed, "Device download has failed")
				return
			}
			logging.Debug("Device with IMSI %d has completed firmware download", device.IMSI)
			store.UpdateFirmwareStateForDevice(device.IMSI, model.Completed, "Device has completed image download")
		}
		return ret, callback, nil
	}
}
