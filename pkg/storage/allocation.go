package storage

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

	"github.com/eesrc/horde/pkg/model"
)

// APNStore is the store used by APNs
type APNStore interface {
	// CreateAPN creates a new APN
	CreateAPN(model.APN) error

	// RemoveAPN removes an APN. The APN must not contain any NASes or allocations.
	RemoveAPN(apnID int) error

	// Create NAS creates a new NAS. The APN must exist and the ID must be unique.
	CreateNAS(model.NAS) error

	// RemoveNAS removes a defined NAS. It can't containy any allocations.
	RemoveNAS(apnID, nasID int) error

	// ListAPN lists the defined APNs
	ListAPN() ([]model.APN, error)

	// ListNAS lists NASes on a particular APN.
	ListNAS(apnID int) ([]model.NAS, error)

	// ListAllocations lists the allocations done through a particular NAS.
	ListAllocations(apnID, nasID, maxRows int) ([]model.Allocation, error)

	// CreateAllocation creates a new allocation
	CreateAllocation(model.Allocation) error

	// RemoveAllocation removes a (single) allocation.
	RemoveAllocation(apnID int, nasID int, imsi int64) error

	// RetrieveAllAllocations retrieves all allocations for a particular IMSI.
	// This is for diagnostic purposes.
	RetrieveAllAllocations(imsi int64) ([]model.Allocation, error)

	// RetrieveAllocation retrieves a single allocation
	RetrieveAllocation(imsi int64, apnid int, nasid int) (model.Allocation, error)

	// LookupIMSIFromIP retrievs an IMSI based on the allocated IP address.
	LookupIMSIFromIP(ip net.IP, ranges model.NASRanges) (int64, error)

	// RetrieveNAS retrieves a single NAS range from the store
	RetrieveNAS(apnID int, nasid int) (model.NAS, error)
}
