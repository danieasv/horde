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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-ocf/go-coap/codes"

	"github.com/eesrc/horde/pkg/apn"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"

	"github.com/ExploratoryEngineering/logging"
)

const (
	coapLwM2MTimeoutSeconds = 30
	coapDownloadTimeout     = time.Minute * 60
)

// LwM2MHandler takes care of all the nitty gritty bits wrt LwM2M registration
// and clients.
type LwM2MHandler struct {
	firmwareUpdater *fwUpdater
	config          Parameters
}

// NewLwM2MHandler creates a new LwM2M server. The server itselfs piggybacks on top
// of an existing CoAP server
func NewLwM2MHandler(coapServer *apn.RxTxReceiver, store storage.DataStore, config Parameters) *LwM2MHandler {
	ret := &LwM2MHandler{
		config:          config,
		firmwareUpdater: newFirmwareUpdater(store, coapServer, config),
	}
	return ret
}

// RegistrationHandler is the registration handler for the LwM2M FOTA process.
// When a device registers the version is checked by querying the /3/0 resource.
// If the device has an outdated version the /5/0/1 resource is set to point to
// the firmware download endpoint and the state of the device download is
// observed. Once completed a reboot command is issued to the device.
func (s *LwM2MHandler) RegistrationHandler() apn.CoAPHandler {
	return func(apnID int, nasID int, device *model.Device, req *rxtx.UpstreamRequest) (*rxtx.DownstreamResponse, apn.ResponseCallback, error) {
		returnCode := func(code codes.Code) (*rxtx.DownstreamResponse, apn.ResponseCallback, error) {
			return &rxtx.DownstreamResponse{
				Msg: &rxtx.Message{
					Coap: &rxtx.CoAPOptions{
						Code:           int32(code),
						TimeoutSeconds: coapLwM2MTimeoutSeconds,
					},
				},
			}, nil, nil
		}

		code := codes.Code(req.Msg.Coap.Code)

		if code == codes.DELETE {
			// Client is removing itself
			return returnCode(codes.Deleted)
		}

		if code != codes.POST && code != codes.PUT {
			logging.Debug("Bad request (code = %s)", code.String())
			return returnCode(codes.BadRequest)
		}

		// This is informational.
		binding := ""
		lifetimeMs := 0
		version := ""
		for _, opt := range req.Msg.Coap.UriQuery {
			f := strings.Split(opt, "=")
			if len(f) != 2 {
				continue
			}
			if f[0] == "b" {
				binding = f[1]
			}
			if f[0] == "lt" {
				lt, err := strconv.ParseInt(f[1], 10, 32)
				if err == nil {
					lifetimeMs = int(lt) * 1000
				}
			}
			if f[0] == "lwm2m" {
				version = f[1]
			}
		}
		// No version checks, just log the version in use. We're wire compatible
		// with all versions but we don't use any of the older features. Also
		// bootstrapping is completely ignored.
		logging.Info("Device %d uses LwM2M version %s, binding %s and session length %dms", device.IMSI, version, binding, lifetimeMs)

		go s.firmwareUpdater.CheckDeviceVersion(*device, req.Msg.RemotePort, req.Msg.RemoteAddress)
		registrationCode := codes.Created
		if req.Msg.Coap.Code == int32(codes.PUT) {
			registrationCode = codes.Valid
		}
		return &rxtx.DownstreamResponse{
			Msg: &rxtx.Message{
				RemotePort:    req.Msg.RemotePort,
				RemoteAddress: req.Msg.RemoteAddress,
				Coap: &rxtx.CoAPOptions{
					Code:         int32(registrationCode),
					LocationPath: []string{"rd", fmt.Sprintf("%d", device.IMSI)},
				},
			},
		}, nil, nil
	}
}
