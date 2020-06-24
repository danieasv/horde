package apitoolbox

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
	"bytes"
	"encoding/json"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/model"
)

// applyFieldMask masks fields from the data store. This is normally done when
// converting from model.Device to entities.Device but since these come
// straight out of the data store we have to apply them here. It might make
// more sense to store the model.Device instances in the database but that's
// for later.
func applyFieldMask(device *apipb.Device, fm model.FieldMask) {
	if fm.IsSet(model.IMSIMask) {
		device.Imsi = nil
	}
	if fm.IsSet(model.IMEIMask) {
		device.Imei = nil
	}
	if fm.IsSet(model.LocationMask) && device.Network != nil {
		device.Network.CellId = nil
	}
}

var pbUnmarshaler = jsonpb.Unmarshaler{}

// JSONMarshaler returns a jsonpb.Marshaler instance with options compatible
// with the REST API.
func JSONMarshaler() jsonpb.Marshaler {
	return jsonpb.Marshaler{
		OrigName:     false,
		EnumsAsInts:  false,
		EmitDefaults: false,
	}
}

// fixMetadata makes the (JSON)-serialised metadata backwards compatible.
// In retrospect it would have been nicer to have a protobuf struct here
// but live and learn.
// Sidenote: Aaahh.... They joys of NoSQL-like storage strategies.
func fixMetadata(metadata []byte) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(metadata, &data); err != nil {
		return nil, err
	}
	if _, ok := data["deviceId"]; ok {
		// Move deviceId, IMSI, IMEI, collectionId, tags to the device element
		// This is the very first metadata struct
		tmpDevice := make(map[string]interface{})
		tmpDevice["deviceId"] = data["deviceId"]
		tmpDevice["imsi"] = data["imsi"]
		tmpDevice["imei"] = data["imei"]
		tmpDevice["collectionId"] = data["collectionId"]
		tmpDevice["tags"] = data["tags"]
		data["device"] = tmpDevice
		delete(data, "deviceId")
		delete(data, "imsi")
		delete(data, "imei")
		delete(data, "collectionId")
		delete(data, "tags")
	}
	// Earliest devices are missing the "transport" field
	if _, ok := data["transport"]; !ok {
		data["transport"] = "udp"
	}
	if d, ok := data["device"]; ok {
		if fw, ok := d.(map[string]interface{})["firmware"]; ok {
			firmware := fw.(map[string]interface{})
			if c, ok := firmware["currentVersion"]; ok {
				firmware["currentFirmwareId"] = c
				delete(firmware, "currentVersion")
			}
			if c, ok := firmware["targetVersion"]; ok {
				firmware["targetFirmwareId"] = c
				delete(firmware, "targetVersion")
			}
			if c, ok := firmware["currentFirmareId"]; ok {
				firmware["currentFirmwareId"] = c
				delete(firmware, "currentFirmareId")
			}
			if v, ok := firmware["state"]; ok {
				firmware["state"] = strings.ToLower(v.(string))
			}
			d.(map[string]interface{})["firmware"] = firmware
		}
	}
	return json.Marshal(data)
}

// UnmarshalDataStoreMetadata unmarshals a data store message from a binary
func UnmarshalDataStoreMetadata(metadata []byte, fieldMask model.FieldMask, payload []byte, created int64) (*apipb.OutputDataMessage, error) {
	ret := &apipb.OutputDataMessage{}

	// TODO: Remove this when all data is migrated. Spelling errors, field names and so on
	fixBuf, err := fixMetadata(metadata)
	if err != nil {
		logging.Warning("Metadata fix error: %v", err)
		return nil, err
	}
	// end TODO

	if err := pbUnmarshaler.Unmarshal(bytes.NewReader(fixBuf), ret); err != nil {
		logging.Warning("Error unmarshaling metadata: %v (sr=%s)", err, string(fixBuf))
		return nil, err
	}
	applyFieldMask(ret.Device, fieldMask)
	ret.Payload = payload
	ret.Received = &wrappers.DoubleValue{Value: nanosToMillis(created)}
	return ret, nil
}
