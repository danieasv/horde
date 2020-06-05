package storetest

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
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

func testAllocations(store storage.APNStore, t *testing.T) {
	assert := require.New(t)
	// Create 10 allocations for two APNs
	const (
		apn1      = 1
		apn2      = 2
		apn3      = 3
		numAllocs = 10
	)

	apnA := model.NewAPN(1)
	apnA.Name = "mda.1"
	assert.NoError(store.CreateAPN(apnA))
	assert.NoError(store.CreateNAS(model.NAS{
		ApnID: apnA.ID,
		ID:    apnA.ID,
		CIDR:  "10.0.0.1/8",
	}))

	apnB := model.NewAPN(2)
	apnB.Name = "mda.2"
	assert.NoError(store.CreateAPN(apnB))
	assert.NoError(store.CreateNAS(model.NAS{
		ApnID: apnB.ID,
		ID:    apnB.ID,
		CIDR:  "10.0.0.1/8",
	}))

	apnC := model.NewAPN(3)
	apnC.Name = "mda.3"
	assert.NoError(store.CreateAPN(apnC))
	assert.NoError(store.CreateNAS(model.NAS{
		ApnID: apnC.ID,
		ID:    apnC.ID,
		CIDR:  "10.0.0.1/8",
	}))

	assert.Error(storage.ErrAlreadyExists, store.CreateAPN(apnB))

	for i := 0; i < numAllocs; i++ {
		alloc1 := model.Allocation{
			IMSI:    int64(i + 1000),
			IP:      net.ParseIP(fmt.Sprintf("10.1.1.%d", i)),
			Created: time.Now(),
			ApnID:   apn1,
			NasID:   apn1,
		}

		assert.NoError(store.CreateAllocation(alloc1))

		alloc2 := model.Allocation{
			IMSI:    int64(i + 2000),
			IP:      net.ParseIP(fmt.Sprintf("10.1.1.%d", i)),
			Created: time.Now(),
			ApnID:   apn2,
			NasID:   apn2,
		}
		assert.NoError(store.CreateAllocation(alloc2))
	}

	allocs1, err := store.ListAllocations(apn1, apn1, numAllocs+1)
	assert.NoError(err)
	assert.Len(allocs1, numAllocs)

	allocs2, err := store.ListAllocations(apn2, apn2, numAllocs+1)
	assert.NoError(err)
	assert.Len(allocs2, numAllocs)

	allocs3, err := store.ListAllocations(apn3, apn3, numAllocs)
	assert.NoError(err)
	assert.Len(allocs3, 0)

	// Allocating for a duplicate IMSI in another APN will not fail
	assert.NoError(store.CreateAllocation(model.Allocation{
		IMSI:  1001,
		ApnID: apn3,
		NasID: apn3,
	}))

	assert.Error(storage.ErrNotFound, store.RemoveAllocation(apn2, apn2, 1001))

	assert.NoError(store.RemoveAllocation(apn3, apn3, 1001))

	a1, err := store.RetrieveAllocation(1001, apn1, apn1)
	assert.NoError(err)

	for _, v := range allocs1 {
		if v.IMSI == a1.IMSI {
			assert.True(reflect.DeepEqual(v, a1), "Returned allocation doesn't match stored allocation")
		}
	}

	_, err = store.RetrieveAllocation(2002, apn2, apn2)
	assert.NoError(err)

	range1 := model.NewNASRanges(apn1, "apn1", apn1, "nas1", "10.0.0.0/8")
	// Find allocations by IP
	_, err = store.LookupIMSIFromIP(net.ParseIP("10.1.1.1"), range1)
	assert.NoError(err)

	range2 := model.NewNASRanges(apn2, "apn2", apn2, "nas2", "10.0.0.0/8")
	_, err = store.LookupIMSIFromIP(net.ParseIP("10.1.1.1"), range2)
	assert.NoError(err)

	range3 := model.NewNASRanges(apn3, "apn3", apn3, "nas3", "10.0.0.0/8")
	_, err = store.LookupIMSIFromIP(net.ParseIP("10.1.1.1"), range3)
	assert.Error(storage.ErrNotFound, err)

	_, err = store.RetrieveAllocation(9901, apn3, apn3)
	assert.Error(storage.ErrNotFound, err)

	for i := 0; i < 10; i++ {
		assert.NoError(store.RemoveAllocation(apn1, apn1, int64(1000+i)))
		assert.NoError(store.RemoveAllocation(apn2, apn2, int64(2000+i)))
	}

	assert.NoError(store.RemoveNAS(apn3, apn3), "Remove NAS 3")
	assert.NoError(store.RemoveNAS(apn2, apn2), "Remove NAS 2")
	assert.NoError(store.RemoveNAS(apn1, apn1), "Remove NAS 1")

	assert.NoError(store.RemoveAPN(apn3), "Remove APN 3")
	assert.NoError(store.RemoveAPN(apn2), "Remove APN 2")
	assert.NoError(store.RemoveAPN(apn1), "Remove APN 1")
}

// TestAPNStore tests an APN store. The store is expected to be empty by the test.
// Test data will be removed by the test.
func TestAPNStore(store storage.APNStore, t *testing.T) {
	testAllocations(store, t)

	store.RemoveNAS(0, 0)
	store.RemoveAPN(0)

	assert := require.New(t)

	apn1 := model.APN{
		ID:   100,
		Name: "Item 1",
	}
	apn2 := model.APN{
		ID:   200,
		Name: "Item 2",
	}
	assert.NoError(store.CreateAPN(apn1), "Did not expect error when creating APN 1")
	defer store.RemoveAPN(apn1.ID)

	assert.NoError(store.CreateAPN(apn2), "Did not expect error when creating APN 2")
	defer store.RemoveAPN(apn2.ID)

	assert.Equal(storage.ErrAlreadyExists, store.CreateAPN(apn1), "Expected ErrAlreadyExists when APN is created a 2nd time")

	nas1a := model.NAS{
		ID:         1,
		Identifier: "a",
		CIDR:       "127.0.0.1/24",
		ApnID:      apn1.ID,
	}

	nas1b := model.NAS{
		ID:         2,
		Identifier: "b",
		CIDR:       "127.0.1.1/24",
		ApnID:      apn1.ID,
	}

	nas2a := model.NAS{
		ID:         1,
		Identifier: "a",
		CIDR:       "127.0.0.1/24",
		ApnID:      apn2.ID,
	}

	nas2b := model.NAS{
		ID:         2,
		Identifier: "b",
		CIDR:       "127.0.1.1/24",
		ApnID:      apn2.ID,
	}

	assert.NoError(store.CreateNAS(nas1a), "Did not expect error when creating NAS 1a")
	defer store.RemoveNAS(nas1a.ApnID, nas1a.ID)
	assert.NoError(store.CreateNAS(nas1b), "Did not expect error when creating NAS 1b")
	defer store.RemoveNAS(nas1b.ApnID, nas1b.ID)
	assert.NoError(store.CreateNAS(nas2a), "Did not expect error when creating NAS 2a")
	defer store.RemoveNAS(nas2a.ApnID, nas2a.ID)
	assert.NoError(store.CreateNAS(nas2b), "Did not expect error when creating NAS 2b")
	defer store.RemoveNAS(nas2b.ApnID, nas2b.ID)

	assert.Equal(storage.ErrAlreadyExists, store.CreateNAS(nas1a), "Expected error when creating a duplicate NAS")

	list, err := store.ListAPN()
	assert.NoError(err, "Did not expect error when listing APN")
	assert.Len(list, 2, "List should contain two APNs")
	assert.Contains(list, apn1, "APN 1 is in list")
	assert.Contains(list, apn2, "APN 2 is in list")

	nasList, err := store.ListNAS(apn2.ID)
	assert.NoError(err, "Did not expect error when listing NASes")
	assert.Len(nasList, 2, "List should contain two NASes")
	assert.Contains(nasList, nas2a, "NAS a is in list")
	assert.Contains(nasList, nas2b, "NAS b is in list")

	nas, err := store.RetrieveNAS(nas1a.ApnID, nas1a.ID)
	assert.NoError(err)
	assert.Equal(nas, nas1a)
	nas, err = store.RetrieveNAS(nas2a.ApnID, nas2a.ID)
	assert.NoError(err)
	assert.Equal(nas, nas2a)

	_, err = store.RetrieveNAS(nas1a.ApnID, 10000)
	assert.Error(storage.ErrNotFound, err)

	// Create 10 allocations for each NAS
	for i := 1; i <= 10; i++ {
		allocation := model.Allocation{
			IP:      net.ParseIP(fmt.Sprintf("127.0.0.%d", i)),
			IMSI:    int64(i),
			IMEI:    int64(i),
			ApnID:   nas2a.ApnID,
			NasID:   nas2a.ID,
			Created: time.Now(),
		}
		assert.NoError(store.CreateAllocation(allocation), "Did not expect error when creating allocation")
	}

	assert.Equal(storage.ErrAlreadyExists, store.CreateAllocation(model.Allocation{
		IP:      net.ParseIP("127.0.0.99"),
		IMSI:    int64(5),
		IMEI:    int64(0),
		ApnID:   nas2a.ApnID,
		NasID:   nas2a.ID,
		Created: time.Now(),
	}), "Expected Err exists on duplicate IMSI")

	allocs, err := store.ListAllocations(nas1a.ApnID, nas1a.ID, 99)
	assert.NoError(err, "Expected no error for NAS 1 a")
	assert.Len(allocs, 0)

	allocs, err = store.ListAllocations(nas2a.ApnID, nas2a.ID, 99)
	assert.NoError(err, "Expected no error when retrieving allocation list")
	assert.Len(allocs, 10, "Expected 10 elements")

	// Create new APN, NAS and allocate a 2nd IP for IMSI 4
	assert.NoError(store.CreateAPN(model.APN{ID: 2, Name: "2nd APN"}))
	assert.NoError(store.CreateNAS(model.NAS{ID: 3, ApnID: 2, CIDR: "10.128.0.1/16"}))
	assert.NoError(store.CreateAllocation(model.Allocation{IP: net.ParseIP("10.128.0.1"), IMSI: 4, ApnID: 2, NasID: 3}))

	allocs, err = store.RetrieveAllAllocations(int64(4))
	assert.NoError(err, "Did not expect error when retrieving allocations for IMSI")
	assert.Len(allocs, 2)

	_, err = store.RetrieveAllocation(4, nas2a.ApnID, nas2a.ID)
	assert.NoError(err)

	_, err = store.RetrieveAllocation(4, 2, 3)
	assert.NoError(err)

	ranges := model.NewNASRanges(2, "apn2", 3, "nas2", "10.0.0.0/8")
	imsi, err := store.LookupIMSIFromIP(net.ParseIP("10.128.0.1"), ranges)
	assert.NoError(err)
	assert.Equal(int64(4), imsi)

	_, err = store.LookupIMSIFromIP(net.ParseIP("10.128.0.99"), ranges)
	assert.Error(storage.ErrNotFound, err)

	assert.Error(storage.ErrNotFound, store.RemoveAllocation(2, 3, 40000))

	assert.NoError(store.RemoveAllocation(2, 3, 4))

	assert.NoError(store.RemoveNAS(2, 3))
	assert.NoError(store.RemoveAPN(2))
	// RetrieveAllAllocations
	// RetrieveAllocation
	// LookupIMSIFromIP
	// RetrieveNAS

	for i := 1; i <= 10; i++ {
		assert.NoError(store.RemoveAllocation(nas2a.ApnID, nas2a.ID, int64(i)), "Did not expect error when removing allocation")
	}

	assert.NoError(store.RemoveNAS(nas1a.ApnID, nas1a.ID), "Remove NAS 1 a OK")
	assert.NoError(store.RemoveNAS(nas1b.ApnID, nas1b.ID), "Remove NAS 1 b OK")
	assert.NoError(store.RemoveAPN(apn1.ID), "Remove APN 1 OK")

	assert.NoError(store.RemoveNAS(nas2a.ApnID, nas2a.ID), "Remove NAS 1 a OK")
	assert.NoError(store.RemoveNAS(nas2b.ApnID, nas2b.ID), "Remove NAS 1 b OK")
	assert.NoError(store.RemoveAPN(apn2.ID), "Remove APN 2 OK")

	assert.Equal(storage.ErrNotFound, store.RemoveNAS(nas1a.ApnID, nas1a.ID), "Expected Not found error when remving NAS a 2nd time")
	assert.Equal(storage.ErrNotFound, store.RemoveAPN(apn1.ID), "Expected not found when removing APN a 2nd time")
}
