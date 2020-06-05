package apn

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
	"context"
	"net"
	"testing"

	"github.com/eesrc/horde/pkg/apn/allocator"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/eesrc/horde/pkg/storage/storetest"
	"github.com/stretchr/testify/require"
)

func TestRADIUSgRPC(t *testing.T) {
	// This is pretty much the test of the access handler code
	assert := require.New(t)

	allocStore := sqlstore.NewMemoryAPNStore()

	apn1 := model.APN{ID: 1, Name: "mda1.ee"}
	nas11 := model.NAS{ID: 1, Identifier: "1NAS1", CIDR: "127.1.1.0/24", ApnID: 1}
	nas12 := model.NAS{ID: 2, Identifier: "1NAS2", CIDR: "127.1.2.0/24", ApnID: 1}
	assert.NoError(allocStore.CreateAPN(apn1))
	assert.NoError(allocStore.CreateNAS(nas11))
	assert.NoError(allocStore.CreateNAS(nas12))

	apn2 := model.APN{ID: 2, Name: "mda2.ee"}
	nas21 := model.NAS{ID: 3, Identifier: "2NAS1", CIDR: "127.2.1.0/24", ApnID: 1}
	nas22 := model.NAS{ID: 4, Identifier: "2NAS2", CIDR: "127.2.2.0/24", ApnID: 1}

	assert.NoError(allocStore.CreateAPN(apn2))
	assert.NoError(allocStore.CreateNAS(nas21))
	assert.NoError(allocStore.CreateNAS(nas22))

	apnConfig, err := storage.NewAPNCache(allocStore)
	assert.NoError(err)

	datastore := sqlstore.NewMemoryStore()
	e := storetest.NewTestEnvironment(t, datastore)

	allocator, err := allocator.NewWriteThroughAllocator(apnConfig, allocStore)
	assert.NoError(err)

	service, err := NewRxtxRADIUSServer(apnConfig, datastore, allocator)
	assert.NoError(err)

	testRequest := func(imsi int64, nasIdentifier string, result bool, cidr string) {
		req := &rxtx.AccessRequest{
			Imsi:          imsi,
			NasIdentifier: nasIdentifier,
		}
		ctx := context.Background()
		res, err := service.Access(ctx, req)
		assert.NoError(err)
		assert.NotNil(res)
		assert.Equal(result, res.Accepted)
		if res.Accepted {
			_, network, _ := net.ParseCIDR(cidr)
			assert.True(network.Contains(res.IpAddress))
		}
	}

	assert.NoError(datastore.CreateDevice(e.U1.ID, model.Device{ID: 1, IMSI: 1001, IMEI: 1001, CollectionID: e.C1.ID, Tags: model.NewTags()}))

	assert.NoError(datastore.CreateDevice(e.U2.ID, model.Device{ID: 3, IMSI: 2001, IMEI: 2001, CollectionID: e.C2.ID, Tags: model.NewTags()}))

	// Uknown NAS
	testRequest(1001, "unknown", false, "")
	// Known device, known NAS
	testRequest(1001, nas11.Identifier, true, nas11.CIDR)
	// Reallocation should also work
	testRequest(1001, nas11.Identifier, true, nas11.CIDR)

	// Allocating via the other NASes should also work
	testRequest(1001, nas11.Identifier, true, nas11.CIDR)
	testRequest(1001, nas12.Identifier, true, nas12.CIDR)
	testRequest(1001, nas21.Identifier, true, nas21.CIDR)

	// ..and a different device should work
	testRequest(2001, nas21.Identifier, true, nas21.CIDR)

	// Unknowns devices are rejected
	testRequest(3001, nas11.Identifier, false, "")
	testRequest(3001, nas12.Identifier, false, "")

	// The range for a single
	// Exhausting a range should give an error. Each range contains 256 addresses
	// but it should not affect the other ranges
	for i := 0; i < 256; i++ {
		assert.NoError(datastore.CreateDevice(e.U1.ID,
			model.Device{
				ID:           model.DeviceKey(3000 + i),
				IMSI:         int64(3000 + i),
				IMEI:         int64(3000 + i),
				CollectionID: e.C1.ID,
				Tags:         model.NewTags(),
			}))
		testRequest(int64(3000+i), nas22.Identifier, true, nas22.CIDR)
	}

	testRequest(1001, nas22.Identifier, false, "")
	testRequest(1001, nas11.Identifier, true, nas11.CIDR)
	testRequest(1001, nas12.Identifier, true, nas12.CIDR)
	testRequest(1001, nas21.Identifier, true, nas21.CIDR)

}
