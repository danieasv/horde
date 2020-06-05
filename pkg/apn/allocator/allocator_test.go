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
	"net"
	"testing"

	"github.com/eesrc/horde/pkg/utils"
	"github.com/stretchr/testify/require"
)

func testAllocator(t *testing.T, nasID int, a DeviceAddressAllocator) {
	assert := require.New(t)

	max := a.Available(nasID)
	cur := a.Available(nasID)
	t.Logf("max available: %d current allocated: %d", max, cur)
	for i := 0; i < max; i++ {
		_, allocated, err := a.AllocateIP(int64(i), nasID)
		assert.NoError(err)
		assert.True(allocated)
		cur--
		assert.Equal(cur, a.Available(nasID), "Expected available addresses to decrease")
	}
	assert.Equal(0, a.Available(nasID), "Expecting 0 available addresses")
	// re-alocating 1 should return the same IP
	_, allocated, err := a.AllocateIP(1, nasID)
	assert.NoError(err)
	assert.False(allocated, "No new address should be allocated")

	// Expect next to fail since there's only 254 addresses free
	var ip net.IP
	ip, allocated, err = a.AllocateIP(int64(max+1), nasID)
	assert.Error(err, "Expected allocation to fail but it didn't for device #%v (available=%d, ip = %s)", max+1, a.Available(nasID), ip.String())
	assert.False(allocated, "Expected no allocation")
	assert.Equal(0, a.Available(nasID))
	// Release one of the addresses, attempt a new allocation
	assert.NoError(a.ReleaseIP(1, nasID), "Should be able to release IMSI 1")

	_, allocated, err = a.AllocateIP(1, nasID)
	assert.NoError(err)
	assert.True(allocated)

	// Release all addresses
	for i := 0; i < max; i++ {
		assert.NoError(a.ReleaseIP(int64(i), nasID))
	}

	assert.Equal(max, a.Available(nasID))
	assert.Equal(0, a.Allocated(nasID))

	imsi := int64(9293)
	_, allocated, err = a.AllocateIP(imsi, nasID)
	assert.NoError(err)
	assert.True(allocated)
	_, allocated, err = a.AllocateIP(imsi, nasID)
	assert.NoError(err)
	assert.False(allocated)

	assert.NoError(a.ReleaseIP(imsi, nasID))
	assert.Error(a.ReleaseIP(imsi, nasID))
}

func testRangeAllocator(t *testing.T, b RangeAllocator, cidr string) {
	assert := require.New(t)
	assert.NotNil(b, "Range must be set")

	_, subnet, err := net.ParseCIDR(cidr)
	assert.NoError(err)

	sum := b.Available() + b.Allocated()

	ips := make([]net.IP, 0)
	prev := b.Available()
	max := b.Available()
	for i := 0; i < max; i++ {
		ip, err := b.AllocateIP()
		assert.NoError(err, "AllocateIP should not return an error")
		assert.NotNil(ip, "Allocated IP should be non-nil")
		prev--
		assert.Equal(prev, b.Available(), "Pool should be 1 less")
		assert.True(subnet.Contains(ip), "IP %s, should be in subnet", ip.String())
		ips = append(ips, ip)
	}
	assert.Equal(0, b.Available(), "0 IP addresses should be available")
	assert.Equal(sum, b.Available()+b.Allocated(), "Sum of allocated + available should be constant")

	assert.Error(b.ReleaseIP(utils.NtoaIPv4(0)), "0.0.0.0 can't be released")

	for _, v := range ips {
		assert.NoError(b.ReleaseIP(v), "IP addresses should be possible to release")
	}

	assert.NotEqual(0, b.Available(), "Addresses should be available")
	assert.Equal(b.Available()+b.Allocated(), sum, "Sum of allocated + available should be constant")
}
