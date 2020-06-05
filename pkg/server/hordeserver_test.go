package server

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
	"os"
	"testing"
	"time"

	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/eesrc/horde/pkg/utils"
	"github.com/stretchr/testify/require"

	"github.com/eesrc/horde/pkg/model"
)

// This is a local test that can be used to measure coverage while running
// the release tests locally. We're abusing the -short flag here so if it is
// set the test will be .. not short :)
// The default behaviour of the test is to send an interrupt signal as soon as
// the service is launched.
//
// Run the test with
//
//    go test -run TestLocalServer -short -cover  --coverprofile=local_test.cover -coverpkg=github.com/eesrc/horde/...
//
// To get the current test coverage run go tool cover:
//
//    go tool cover -func=local_test.cover
//
func TestLocalServer(t *testing.T) {
	args := []string{
		"--log-type=plain",
		"--log-level=debug",
	}
	if !testing.Short() {
		go func() {
			utils.GetSignalChannel() <- os.Interrupt
		}()
	}
	LaunchHorde(args)
}

func TestMetadataConversion(t *testing.T) {
	assert := require.New(t)
	msg := model.DataMessage{
		Device: model.Device{
			ID:           1,
			IMSI:         2,
			IMEI:         3,
			CollectionID: 4,
			Network: model.DeviceNetworkMetadata{
				AllocatedIP: "127.0.0.1",
				AllocatedAt: time.Now(),
				CellID:      5,
				ApnID:       6,
				NasID:       7,
			},
			Firmware: model.DeviceFirmwareMetadata{
				CurrentFirmwareID: 8,
				TargetFirmwareID:  9,
				FirmwareVersion:   "10",
				SerialNumber:      "11",
				ModelNumber:       "12",
				Manufacturer:      "13",
				State:             model.Downloading,
				StateMessage:      "14",
			},
		},
		Received:  time.Now(),
		Payload:   nil,
		Transport: model.UDPPullTransport,
		UDP: model.UDPMetaData{
			LocalPort:  15,
			RemotePort: 16,
		},
	}
	buf := makeMetadata(msg)
	assert.True(len(buf) > 0)
	res, err := apitoolbox.UnmarshalDataStoreMetadata(buf, 0, []byte("hello there"), time.Now().UnixNano())
	assert.NoError(err)
	assert.NotNil(res)
	assert.NotNil(res.UdpMetaData)
	assert.NotNil(res.UdpMetaData.RemotePort)
	assert.Equal(int32(16), res.UdpMetaData.RemotePort.Value)
}
