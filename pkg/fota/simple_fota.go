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
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/apn"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/go-ocf/go-coap/codes"
)

// This is the phone home endpoint for the devices. This allows the devices
// to use a simplified FOTA procedure since LwM2M can be a true PITA to get
// up and running.

// Report is the request body of the simple FOTA procedure
type Report struct {
	FirmwareVersion  string
	ManufacturerName string
	SerialNumber     string
	ModelNumber      string
}

const (
	firmwareVersionID  = 1
	manufacturerNameID = 2
	serialNumberID     = 3
	modelNumberID      = 4

	hostID      = 1
	portID      = 2
	pathID      = 3
	availableID = 4

	tlvFieldHeaderLength = 2
)

// Encode strings into TLV buffer. Multi-byte characters aren't supported
func encodeTLVString(id byte, idx *int, buf []byte, val string) error {
	buf[*idx] = id
	*idx++
	buf[*idx] = byte(len(val))
	*idx++
	for _, ch := range val {
		buf[*idx] = byte(ch)
		*idx++
	}
	return nil
}

// Decode TLV string in buffer. Multi-byte characters aren't supported
func decodeTLVString(idx *int, buf []byte, val *string) error {
	*val = ""
	len := buf[*idx]
	*idx++
	if len == 0 {
		return nil
	}
	for i := byte(0); i < len; i++ {
		*val += string(buf[*idx])
		*idx++
	}
	return nil
}

// Encode Uint32 into buffer. Big endian encoding.
func encodeTLVUInt32(id byte, idx *int, buf []byte, val uint32) error {
	buf[*idx] = id
	*idx++
	buf[*idx] = 4
	*idx++
	binary.BigEndian.PutUint32(buf[*idx:], val)
	*idx += 4
	return nil
}

// Encode boolean value into buffer. The encoding is a bit wasteful since it
// uses three bytes to represent a single boolean value but consistency is king.
func encodeTLVBool(id byte, idx *int, buf []byte, val bool) error {
	buf[*idx] = id
	*idx++
	buf[*idx] = 1
	*idx++
	if val {
		buf[*idx] = 1
	} else {
		buf[*idx] = 0
	}
	*idx++
	return nil
}

// UnmarshalBinary decodes a report payload.
func (r *Report) UnmarshalBinary(buf []byte) error {
	idx := 0
	firmwareVersion := ""
	manufacturerName := ""
	serialNumber := ""
	modelNumber := ""

	for idx < len(buf) {
		id := buf[idx]
		idx++
		switch id {
		case firmwareVersionID:
			if err := decodeTLVString(&idx, buf, &firmwareVersion); err != nil {
				return err
			}
		case manufacturerNameID:
			if err := decodeTLVString(&idx, buf, &manufacturerName); err != nil {
				return err
			}
		case serialNumberID:
			if err := decodeTLVString(&idx, buf, &serialNumber); err != nil {
				return err
			}
		case modelNumberID:
			if err := decodeTLVString(&idx, buf, &modelNumber); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unable to unmarshal id %d", buf[idx])
		}
	}

	r.FirmwareVersion = firmwareVersion
	r.ManufacturerName = manufacturerName
	r.SerialNumber = serialNumber
	r.ModelNumber = modelNumber
	return nil
}

// SimpleFOTAResponse is the response for the simple FOTA procedure. It is sent
// to the device as a response to the client. The host, port and path is sent
// as separate fields just to make it simpler to decode. The path is separated
// by slashes.
type SimpleFOTAResponse struct {
	Host           string
	Port           uint32
	Path           string
	ImageAvailable bool
}

// MarshalBinary encodes the response into a byte buffer that is sent to the
// client. The content is TLV encoded.
func (r *SimpleFOTAResponse) MarshalBinary() ([]byte, error) {

	buf := make([]byte,
		tlvFieldHeaderLength+len(r.Host)+
			tlvFieldHeaderLength+4+
			tlvFieldHeaderLength+len(r.Path)+
			tlvFieldHeaderLength+1)
	idx := 0
	if err := encodeTLVString(hostID, &idx, buf, r.Host); err != nil {
		return nil, err
	}
	if err := encodeTLVUInt32(portID, &idx, buf, r.Port); err != nil {
		return nil, err
	}
	if err := encodeTLVString(pathID, &idx, buf, r.Path); err != nil {
		return nil, err
	}
	if err := encodeTLVBool(availableID, &idx, buf, r.ImageAvailable); err != nil {
		return nil, err
	}
	return buf, nil
}

var lwm2mHandler *LwM2MHandler

// SetupFOTA sets up the simple FOTA endpoint
func SetupFOTA(config Parameters, receiver *apn.RxTxReceiver, datastore storage.DataStore, firmwareStore storage.FirmwareImageStore) error {
	if lwm2mHandler != nil {
		return errors.New("already started FOTA")
	}
	logging.Info("Registering handler for /u /fw and /rd endpoints in CoAP server")
	receiver.AddCoAPHandler("u", newSimpleCoAPHandler(config, datastore))
	receiver.AddCoAPHandler("fw", newFirmwareHandler(receiver, config.DownloadTimeout, datastore, firmwareStore))

	lwm2mHandler = NewLwM2MHandler(receiver, datastore, config)
	receiver.AddCoAPHandler("rd", lwm2mHandler.RegistrationHandler())
	return nil
}

func writeResponse(c codes.Code, payload []byte) (*rxtx.DownstreamResponse, apn.ResponseCallback, error) {
	return &rxtx.DownstreamResponse{
		Msg: &rxtx.Message{
			Type:      rxtx.MessageType_CoAPPull,
			Timestamp: time.Now().UnixNano(),
			Payload:   payload,
			Coap: &rxtx.CoAPOptions{
				Code: int32(c),
			},
		},
	}, nil, nil
}

func newSimpleCoAPHandler(config Parameters, datastore storage.DataStore) apn.CoAPHandler {
	return func(apnID int, nasID int, device *model.Device, req *rxtx.UpstreamRequest) (*rxtx.DownstreamResponse, apn.ResponseCallback, error) {

		// Device will send a buffer with TLV encoded parameters and the
		// server responds with a TLV encoded buffer pointing to the
		// firmware download endpoint.
		switch codes.Code(req.Msg.Coap.Code) {
		case codes.POST:
			// Will only accept POST requests to this endpoint
			data := Report{}
			if err := data.UnmarshalBinary(req.Msg.Payload); err != nil {
				logging.Warning("Couldn't unmarshal request: %v. Sending 4.00 bad request", err)
				return writeResponse(codes.BadRequest, nil)
			}
			logging.Debug("Device fw version: %s", data.FirmwareVersion)
			logging.Debug("        manuf ver: %s", data.ManufacturerName)
			logging.Debug("            model: %s", data.ModelNumber)
			logging.Debug("           serial: %s", data.SerialNumber)

			needsUpdate, _, err := firmwareUpdateCheck(device, data, datastore)

			if err != nil {
				logging.Warning("Got error checking firmware for device with IMSI %d: %v", device.IMSI, err)
				return writeResponse(codes.InternalServerError, nil)
			}

			// Extract host, port, path from the firmware URL in the configuration
			host, port, path, err := config.GetFirmwareHostPortPath()
			if err != nil {
				logging.Error("CoAP firmware path (%s) is invalid. Couldn't parse: %v ", config.FirmwareEndpoint, err)

				return writeResponse(codes.InternalServerError, nil)
			}

			// The Zephyr CoAP library prefers 2.04 Created response to to POSTs
			// but it should not make a difference as long as it is 2.xx.
			resp := SimpleFOTAResponse{
				Host:           host,
				Port:           uint32(port),
				Path:           path,
				ImageAvailable: needsUpdate,
			}
			payload, err := resp.MarshalBinary()
			if err != nil {
				logging.Error("Error marshaling response: %v. Sending 5.00 internal server error", err)
				return writeResponse(codes.InternalServerError, nil)
			}
			logging.Debug("Sending response Host:%s  Port:%d  Path:%s  Image:%t",
				resp.Host, resp.Port, resp.Path, resp.ImageAvailable)
			return writeResponse(codes.Created, payload)

		default:
			logging.Warning("Expecting POST request from device. Sending 4.00 bad request")
			return writeResponse(codes.BadRequest, nil)
		}
	}
}
