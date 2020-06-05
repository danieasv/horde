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
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

type wtAllocator struct {
	apnConfig  *storage.APNConfigCache
	store      storage.APNStore
	allocators map[int]RangeAllocator
}

// NewWriteThroughAllocator creates a write-through allocator that writes
// the allocations to a backend store while using the memory allocator
// for the allocations.
func NewWriteThroughAllocator(apnConfig *storage.APNConfigCache, store storage.APNStore) (DeviceAddressAllocator, error) {
	ret := &wtAllocator{
		apnConfig:  apnConfig,
		store:      store,
		allocators: make(map[int]RangeAllocator),
	}
	for _, apn := range apnConfig.APN {
		for _, nas := range apn.Ranges {
			var err error
			a, err := ret.createAllocator(nas)
			if err != nil {
				return nil, err
			}
			ret.allocators[nas.ID] = a
			metrics.DefaultRADIUSCounters.UpdateIPAllocation(nas.Identifier, a.Allocated(), a.Available())
		}
	}
	return ret, nil
}

func (wt *wtAllocator) createAllocator(nas model.NAS) (RangeAllocator, error) {
	nasAllocations, err := wt.store.ListAllocations(nas.ApnID, nas.ID, 999999999)
	if err != nil {
		return nil, err
	}
	alloc, err := NewMemoryIPAllocator(nas.CIDR, nasAllocations)
	if err != nil {
		return nil, err
	}
	return alloc, nil
}

func (wt *wtAllocator) AllocateIP(imsi int64, nasID int) (net.IP, bool, error) {
	nas, ok := wt.apnConfig.FindByID(nasID)
	if !ok {
		logging.Warning("Unknown NAS ID: %d", nasID)
		return nil, false, nil
	}
	allocation, err := wt.store.RetrieveAllocation(imsi, nas.ApnID, nas.ID)
	if err != nil && err != storage.ErrNotFound {
		// Unable to retrieve existing allocation, report error
		logging.Warning("Unable to check for existing allocation: %v. Assuming new allocation.", err)
	}
	if err == nil {
		logging.Info("Device with IMSI %v reuses IP %s", imsi, allocation.IP)
		return allocation.IP, false, nil
	}
	allocator, ok := wt.allocators[nasID]
	if !ok {
		// Create a new allocator for this NAS
		allocator, err = wt.createAllocator(nas)
		if err != nil {
			logging.Error("Error creating allocator for APN ID %d and NAS ID %d: %v", nas.ApnID, nas.ID, err)
			return nil, false, errors.New("unknown NAS range")
		}
		wt.allocators[nasID] = allocator
	}
	ip, err := allocator.AllocateIP()
	if err != nil {
		logging.Warning("Unable to allocate IP for device %v: %v", imsi, err)
		return nil, false, err
	}
	allocation = model.Allocation{
		IMSI:    imsi,
		IP:      ip,
		ApnID:   nas.ApnID,
		NasID:   nas.ID,
		Created: time.Now(),
	}
	if err := wt.store.CreateAllocation(allocation); err != nil {
		logging.Warning("Unable to store assigned IP on device")
		allocator.ReleaseIP(ip)
		return ip, false, err
	}
	logging.Info("Device with IMSI %v has IP %s", imsi, ip)
	return ip, true, nil
}

func (wt *wtAllocator) Available(nasID int) int {
	allocator, ok := wt.allocators[nasID]
	if !ok {
		return 0
	}
	return allocator.Available()
}

func (wt *wtAllocator) Allocated(nasID int) int {
	allocator, ok := wt.allocators[nasID]
	if !ok {
		return 0
	}
	return allocator.Allocated()
}

func (wt *wtAllocator) ReleaseIP(imsi int64, nasID int) error {
	nas, ok := wt.apnConfig.FindByID(nasID)
	if !ok {
		return errors.New("unknown NAS ID")
	}
	allocator, ok := wt.allocators[nas.ID]
	if !ok {
		return errors.New("unknown NAS ID")
	}

	alloc, err := wt.store.RetrieveAllocation(imsi, nas.ApnID, nas.ID)
	if err != nil {
		return err
	}
	if err := wt.store.RemoveAllocation(nas.ApnID, nas.ID, imsi); err != nil {
		return err
	}

	return allocator.ReleaseIP(alloc.IP)
}
