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
	"database/sql"
	"os"
	"testing"

	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/model"
	"github.com/stretchr/testify/require"

	//PostgreSQL driver
	_ "github.com/lib/pq"
)

// Ensure we don't break the old serialization for messages. TODO: Replace with
// new types when the data store has been migrated to a new format.
func TestBackwardsCompatibleMarshal(t *testing.T) {
	assert := require.New(t)

	// This is a JSON-serialized buffer from the old entities package
	buf := []byte(`
	{
		"type":"data",
		"device":{
			"deviceId":"device-1",
			"collectionId":"collection-1",
			"imei":"123",
			"imsi":"321",
			"tags":{
				"name":"something"
			},
			"network":{
				"allocatedIp":"127.0.0.1",
				"allocatedAt":1000,
				"cellId":2390
			},
			"firmware":{
				"currentFirmareId":"current-1",
				"targetFirmwareId":"target-1",
				"firmwareVersion":"version-1",
				"serialNumber":"serial-1",
				"modelNumber":"model-1",
				"manufacturer":"manuf-1"
			}
		},
		"payload":null,
		"received":0,
		"transport":"coap",
		"udpMetaData":{
			"localPort":1,
			"remotePort":2
		},
		"coapMetaData":{
			"code":"POST",
			"path":"/path"
		}
	}
	`)

	fieldMask := model.FieldMask(0)

	n, err := UnmarshalDataStoreMetadata(buf, fieldMask, []byte("Hello there"), 4711000000)
	assert.NoError(err)

	// Replace payload and received
	assert.Equal(apipb.OutputDataMessage_data, n.Type)
	assert.Equal("device-1", n.Device.DeviceId.Value)
	assert.Equal("collection-1", n.Device.CollectionId.Value)
	assert.Equal("123", n.Device.Imei.Value)
	assert.Equal("321", n.Device.Imsi.Value)
	assert.Equal("something", n.Device.Tags["name"])
	assert.NotNil(n.Device.Firmware)
	assert.Equal("current-1", n.Device.Firmware.CurrentFirmwareId.Value)
	assert.Equal("target-1", n.Device.Firmware.TargetFirmwareId.Value)
	assert.Equal("serial-1", n.Device.Firmware.SerialNumber.Value)
	assert.Equal("manuf-1", n.Device.Firmware.Manufacturer.Value)
	assert.Equal("version-1", n.Device.Firmware.FirmwareVersion.Value)
	assert.Equal("model-1", n.Device.Firmware.ModelNumber.Value)
	assert.Equal(int64(1000), n.Device.Network.AllocatedAt.Value)
	assert.Equal("127.0.0.1", n.Device.Network.AllocatedIp.Value)
	assert.Equal(int64(2390), n.Device.Network.CellId.Value)
	assert.Equal("Hello there", string(n.Payload))
	assert.Equal("coap", n.Transport)
	assert.Equal(int64(4711), int64(n.Received.Value))
}

// Ensure old metadata is processed as they should be
func TestOldDataFixed(t *testing.T) {
	assert := require.New(t)
	// invalid json
	src := `{`
	m, err := UnmarshalDataStoreMetadata([]byte(src), model.FieldMask(0), []byte("payload"), 1)
	assert.Error(err)
	assert.Nil(m)

	// Valid json but invalid fields
	src = `{"modus":"operandi"}`
	m, err = UnmarshalDataStoreMetadata([]byte(src), model.FieldMask(0), []byte("payload"), 1)
	assert.Error(err)
	assert.Nil(m)

	// old school json, no tags included
	src = `{"deviceId":"a","imsi":"1","imei":"2","collectionId":"aa"}`
	m, err = UnmarshalDataStoreMetadata([]byte(src), model.FieldMask(0), []byte("payload"), 1)
	assert.NoError(err)
	assert.NotNil(m)

	// Old school, with tags, no transport
	src = `{"deviceId":"a","imsi":"1","imei":"2","collectionId":"aa","tags":{"name":"some name"}}`
	m, err = UnmarshalDataStoreMetadata([]byte(src), model.FieldMask(0), []byte("payload"), 1)
	assert.NoError(err)
	assert.NotNil(m)

	// Old school with transport
	src = `{"deviceId":"a","imsi":"1","imei":"2","collectionId":"aa","tags":{"name":"some name"}, "transport":"udp"}`
	m, err = UnmarshalDataStoreMetadata([]byte(src), model.FieldMask(0), []byte("payload"), 1)
	assert.NoError(err)
	assert.NotNil(m)

	// Old school with invalid transport
	src = `{"deviceId":"a","imsi":"1","imei":"2","collectionId":"aa","tags":{"name":"some name"}, "transport":"tcp"}`
	m, err = UnmarshalDataStoreMetadata([]byte(src), model.FieldMask(0), []byte("payload"), 1)
	assert.NoError(err)
	assert.NotNil(m)

	// New style, old current version fields in firmware
	src = `
		{
			"device":{
				"deviceId":"a",
				"imsi":"1",
				"imei":"2",
				"collectionId":"aa",
				"firmware": {
					"currentVersion": "b",
					"targetVersion": "c",
					"state": "current"
				},
				"tags":{
					"name":"some name"
				}
			},
			"transport":"udp"
		}`
	m, err = UnmarshalDataStoreMetadata([]byte(src), model.FieldMask(0), []byte("payload"), 1)
	assert.NoError(err)
	assert.NotNil(m)

	// New style, new version
	src = `
		{
			"device":{
				"deviceId":"a",
				"imsi":"1",
				"imei":"2",
				"collectionId":"aa",
				"firmware": {
					"currentFirmwareId": "b",
					"targetFirmwareId": "c",
					"state": "current"
				},
				"network": {
					"cellId": 199
				},
				"tags":{
					"name":"some name"
				}
			},
			"transport":"coap"
		}`
	m, err = UnmarshalDataStoreMetadata([]byte(src), model.FieldMask(0), []byte("payload"), 1)
	assert.NoError(err)
	assert.NotNil(m)

	// Ensure field mask is applied
	m, err = UnmarshalDataStoreMetadata([]byte(src), model.FieldMask(model.IMSIMask), []byte("payload"), 1)
	assert.NoError(err)
	assert.NotNil(m)
	assert.Nil(m.Device.Imsi)
	assert.Equal("2", m.Device.Imei.Value)
	assert.Equal(int64(199), m.Device.Network.CellId.Value)

	// Ensure field mask is applied
	m, err = UnmarshalDataStoreMetadata([]byte(src), model.FieldMask(model.IMEIMask), []byte("payload"), 1)
	assert.NoError(err)
	assert.NotNil(m)
	assert.Equal("1", m.Device.Imsi.Value)
	assert.Nil(m.Device.Imei)
	assert.Equal(int64(199), m.Device.Network.CellId.Value)

	// Ensure field mask is applied
	m, err = UnmarshalDataStoreMetadata([]byte(src), model.FieldMask(model.LocationMask), []byte("payload"), 1)
	assert.NoError(err)
	assert.NotNil(m)
	assert.Equal("1", m.Device.Imsi.Value)
	assert.Equal("2", m.Device.Imei.Value)
	assert.Nil(m.Device.Network.CellId)
}

// Test *all* the marshalled data stored this far. It should work for all
// data available.
func TestOldData(t *testing.T) {
	// Needs a running postgreSQL server. Skip if the environment variable
	// $POSTGRES is not set or blank
	connStr := os.Getenv("POSTGRES")
	if connStr == "" {
		return
	}
	assert := require.New(t)
	t.Log("Open DB")
	db, err := sql.Open("postgres", connStr)
	assert.NoError(err)
	defer db.Close()

	res, err := db.Query("SELECT metadata, payload, created FROM magpie_data")
	assert.NoError(err)
	defer res.Close()

	t.Log("Start query")
	metadata := make([]byte, 8192)
	payload := make([]byte, 1024)
	var created int64
	count := 0
	for res.Next() {
		assert.NoError(res.Scan(&metadata, &payload, &created))
		fieldMask := model.FieldMask(0)

		o, err := UnmarshalDataStoreMetadata(metadata, fieldMask, payload, created)
		assert.NoError(err)
		if o.Device == nil {
			t.Logf("Device is nil for %v", string(metadata))
		}
		if o.Payload == nil {
			t.Logf("Payload is nil for %v", string(metadata))
		}
		if o.Transport == "" {
			t.Logf("Transport is empty for %v", string(metadata))
		}
		count++
		if (count % 10000) == 0 {
			t.Logf("%d rows processed", count)
		}
	}
	t.Logf("Done processing. %d processed", count)
}
