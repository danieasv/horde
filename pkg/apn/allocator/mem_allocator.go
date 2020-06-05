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
	"errors"
	"net"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/utils"
)

// memoryIPAllocator allocates addresses in memory for a single IP range.
type memoryIPAllocator struct {
	subnet     *net.IPNet
	free       []uint32
	allocated  map[uint32]struct{}
	currentIP  uint32
	blockStart int
}

func (m *memoryIPAllocator) AllocateIP() (net.IP, error) {
	if len(m.allocated) == 0 {
		newIP := m.currentIP
		m.allocated[newIP] = struct{}{}
		return utils.NtoaIPv4(newIP), nil
	}
	// reuse one in the free table before allocating a new one
	if len(m.free) > 0 {
		ip := m.free[0]
		m.free = m.free[1:]
		m.allocated[ip] = struct{}{}
		logging.Debug("Recycling IP: %v (%d addresses left in free pool)", utils.NtoaIPv4(ip), len(m.free))
		return utils.NtoaIPv4(ip), nil
	}

	m.currentIP++
	_, ok := m.allocated[m.currentIP]
	for ok {
		m.currentIP++
		_, ok = m.allocated[m.currentIP]
	}
	// Range is full
	if !m.subnet.Contains(utils.NtoaIPv4(m.currentIP)) {
		return nil, errors.New("no more addresses available")
	}
	newIP := m.currentIP
	m.allocated[newIP] = struct{}{}
	return utils.NtoaIPv4(newIP), nil
}

func (m *memoryIPAllocator) Available() int {
	ones, _ := m.subnet.Mask.Size()
	size := (1 << uint32(32-ones)) - m.blockStart
	return int(size) - len(m.allocated)
}

func (m *memoryIPAllocator) Allocated() int {
	return len(m.allocated)
}

func (m *memoryIPAllocator) ReleaseIP(ip net.IP) error {
	oldIP := utils.AtonIPv4(ip)
	_, ok := m.allocated[oldIP]
	if !ok {
		return errors.New("address not allocated")
	}
	delete(m.allocated, oldIP)
	m.free = append(m.free, oldIP)
	return nil
}

// Rebuild allocations increments the currentIP for the allocations
// while updating the allocations and the free list of IP addresses.
func (m *memoryIPAllocator) rebuildAllocations(current []model.Allocation) error {
	if len(current) == 0 {
		return nil
	}
	for _, v := range current {
		m.allocated[utils.AtonIPv4(v.IP)] = struct{}{}
	}
	return nil
}

// NewMemoryIPAllocator returns a memory-based memory allocators. Note that the
// IP addresses in the allocations are are IPv4 addresses so make sure to call
// To4() if you use the net.ParseIP function. (yes this has caused me headaches)
func NewMemoryIPAllocator(cidr string, currentAllocations []model.Allocation) (RangeAllocator, error) {
	ret := memoryIPAllocator{}
	var err error
	var first net.IP
	first, ret.subnet, err = net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	ret.currentIP = utils.AtonIPv4(first)
	// The CIDR range might not start at .0 or .1 but can be .10. The ParseCIDR
	// function returns the first IP in the range while the subnet's first IP
	// is the .0 address.
	ret.blockStart = int(ret.currentIP - utils.AtonIPv4(ret.subnet.IP))
	ret.allocated = make(map[uint32]struct{})
	if err := ret.rebuildAllocations(currentAllocations); err != nil {
		return nil, err
	}
	return &ret, nil
}
