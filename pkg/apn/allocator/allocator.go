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
import "net"

// RangeAllocator is a type capable of allocating IP addresses in a range.
type RangeAllocator interface {
	// AllocateIP allocates a single IP address. An error is returned if the
	// IP addresse can't be allocated
	AllocateIP() (net.IP, error)
	// ReleaseIP releases a previously allocated IP address.
	ReleaseIP(net.IP) error

	// Available returns the number of available IP addresses
	Available() int

	// Allocated returns the number of allocated IP addresses.
	Allocated() int
}

// DeviceAddressAllocator is a type that can allocate IP addresses for devices.
type DeviceAddressAllocator interface {
	// AllocateIP allocates an IP address in a single range of IP addresses. The
	// call is idempotent; if the same IMSI tries to allocate an address a
	// 2nd time it will return the same IP address.
	AllocateIP(imsi int64, nasID int) (net.IP, bool, error)

	// Allocated returns the number of currently allocated addresses in the pool
	Allocated(nasID int) int

	// Available returns the number of available addresses in the pool
	Available(nasID int) int

	// ReleaseIP releases the IP address and moves it into the free pool.
	ReleaseIP(imsi int64, nasID int) error
}
