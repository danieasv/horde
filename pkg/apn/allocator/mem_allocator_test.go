package allocator

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
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
)

func TestMemoryAllocator(t *testing.T) {
	cidr := "10.1.0.1/24"
	assert := require.New(t)
	a, err := NewMemoryIPAllocator(cidr, make([]model.Allocation, 0))
	assert.NoError(err)
	testRangeAllocator(t, a, cidr)
}

func TestAllocation(t *testing.T) {
	const cidr = "10.0.0.1/24"

	assert := require.New(t)
	list := make([]model.Allocation, 0)
	a, err := NewMemoryIPAllocator(cidr, list)
	if err != nil {
		t.Fatal(err)
	}

	expectedAvailable := a.Available()
	// Allocate 10 addresses, create a new allocator with the list
	for i := 1; i <= 10; i++ {
		ip, err := a.AllocateIP()
		assert.NoError(err)
		list = append(list, model.Allocation{IP: ip, IMSI: int64(i), Created: time.Now()})
	}
	expectedAvailable -= 10
	assert.Equalf(expectedAvailable, a.Available(), "Expected %d available addresses but got %d (was %d)", expectedAvailable, a.Available(), expectedAvailable+3)

	// Release the first two from the list and decrement it
	assert.NoError(a.ReleaseIP(list[0].IP))

	assert.NoError(a.ReleaseIP(list[1].IP))

	expectedAvailable += 2
	assert.Equal(expectedAvailable, a.Available())
	list = list[2:]

	// Create a new list based on the old one
	newA, err := NewMemoryIPAllocator(cidr, list)
	assert.NoError(err)

	assert.Equal(expectedAvailable, newA.Available(), "Expected new list to have same number of available IPs")

	_, err = newA.AllocateIP()
	assert.NoError(err, "should be allowed to allocate IP 1")

	_, err = newA.AllocateIP()
	assert.NoError(err, "Unable to allocate IP 2")

	expectedAvailable -= 2
	assert.Equal(expectedAvailable, newA.Available())

	// Release all from the new list. No errors expected
	for len(list) > 0 {
		assert.NoErrorf(newA.ReleaseIP(list[0].IP), "Unable to release IP")
		list = list[1:]
	}
}

func BenchmarkMemoryAllocator(b *testing.B) {
	a, err := NewMemoryIPAllocator("10.1.0.1/8", make([]model.Allocation, 0))
	if err != nil {
		b.Fatal(err)
	}

	allocated := make([]net.IP, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i < 1 || rand.Int()%2 == 0 {
			ip, err := a.AllocateIP()
			if err != nil {
				allocated = append(allocated, ip)
			}
		} else {
			ip := net.ParseIP("127.0.0.1")
			if len(allocated) > 0 {
				ip = allocated[0]
				allocated = allocated[1:]
			}
			a.ReleaseIP(ip)
		}
	}
}

func TestRestartAllocator(t *testing.T) {
	assert := require.New(t)

	var st storage.APNStore = sqlstore.NewMemoryAPNStore()

	apn := model.APN{ID: 1, Name: "Allocator test APN"}
	nas := model.NAS{ApnID: 1, ID: 1, CIDR: "10.1.1.1/24"}
	assert.NoError(st.CreateAPN(apn))
	assert.NoError(st.CreateNAS(nas))

	allocations := []model.Allocation{
		model.Allocation{IP: net.ParseIP("10.1.1.1"), IMSI: 1, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.9"), IMSI: 2, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.11"), IMSI: 3, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.12"), IMSI: 4, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.15"), IMSI: 5, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.16"), IMSI: 6, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.17"), IMSI: 7, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.118"), IMSI: 8, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.119"), IMSI: 0, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.110"), IMSI: 10, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.125"), IMSI: 11, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.150"), IMSI: 12, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.175"), IMSI: 13, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
		model.Allocation{IP: net.ParseIP("10.1.1.100"), IMSI: 14, Created: time.Now(), ApnID: apn.ID, NasID: nas.ID},
	}
	for _, v := range allocations {
		assert.NoError(st.CreateAllocation(v), "Unable to allocate %+v", v)
	}

	allocs, err := st.ListAllocations(1, 1, 9999)
	assert.NoError(err)

	allocator, err := NewMemoryIPAllocator("10.1.1.1/24", allocs)
	assert.NoError(err)

	// Next allocated IP should be 10.1.1.101
	ip, err := allocator.AllocateIP()
	assert.NoError(err)

	allocations = append(allocations, model.Allocation{IP: ip, IMSI: 15, Created: time.Now(), ApnID: 0})

	for i := 0; i < 254-15; i++ {
		ip, err := allocator.AllocateIP()
		assert.NoError(err)
		for _, v := range allocations {
			assert.NotEqualf(ip.String(), v.IP.String(), "Double allocation! %v already exists in list of allocated IPs", v.IP.String())
		}
		allocations = append(allocations, model.Allocation{IP: ip, IMSI: int64(i + 15), Created: time.Now(), ApnID: 0})
	}
}

// Ensure available calculations are correct.
func TestAvailableCalculation(t *testing.T) {
	assert := require.New(t)

	a, err := NewMemoryIPAllocator("10.0.0.1/24", nil)
	assert.NoError(err)
	assert.Equal(255, a.Available())

	a, err = NewMemoryIPAllocator("10.0.0.0/24", nil)
	assert.NoError(err)
	assert.Equal(256, a.Available())

	a, err = NewMemoryIPAllocator("10.0.0.10/24", nil)
	assert.NoError(err)
	assert.Equal(246, a.Available())
}
