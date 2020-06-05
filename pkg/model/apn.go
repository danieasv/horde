package model

import (
	"net"

	"github.com/ExploratoryEngineering/logging"
)

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
//
// APNs represent external systems interfacing with Horde on the device side.
// Our current setup assumes a regular APN setup with some kind of transport
// between the PGW/NAS to Horde. The NAS issues RADIUS requests to Horde and
// expects a certain IP range in response. An APN consists of one or more
// NASes. Horde might be configured with multiple APNs, each with one or more
// NASes. Each NAS has an allocated range of IP addresses.
//

//
// APN is the APN configuration.
type APN struct {
	ID   int
	Name string
}

// NewAPN creates a new APN instance
func NewAPN(id int) APN {
	return APN{ID: id}
}

// NAS represents a NAS with a range.
type NAS struct {
	ID         int
	Identifier string
	CIDR       string
	ApnID      int
	net        *net.IPNet
}

// NASRanges is a helper struct to hold both APN ID and a collection of NAS IDs.
// This represents a *single* APN and not all available APNs
// TODO(stalehd): Remove this when ingres listeners are completed.
type NASRanges struct {
	APN    APN
	Ranges []NAS
}

// GetNasID returns the NAS ID for the NAS identifier. It will return -1 if
// the identifier isn't found
func (n *NASRanges) GetNasID(identifier string) int {
	for _, v := range n.Ranges {
		if v.Identifier == identifier {
			return v.ID
		}
	}
	return -1
}

// Find locates the matching NAS
func (n *NASRanges) Find(nasID int) (NAS, bool) {
	for _, v := range n.Ranges {
		if v.ID == nasID {
			return v, true
		}
	}
	return NAS{}, false
}

// ByIP locates the NAS by the IP address
func (n *NASRanges) ByIP(ip net.IP) (NAS, bool) {
	for _, v := range n.Ranges {
		if v.net == nil {
			var err error
			_, v.net, err = net.ParseCIDR(v.CIDR)
			if err != nil {
				logging.Warning("Error parsing CIDR %s: %v", v.CIDR, err)
				return v, false
			}
		}
		if v.net != nil && v.net.Contains(ip) {
			return v, true
		}
	}
	return NAS{}, false
}

// NewNASRanges creates a new NASRanges instance
func NewNASRanges(apnID int, name string, nasID int, identifier string, cidr string) NASRanges {
	return NASRanges{
		APN:    APN{ID: apnID, Name: name},
		Ranges: []NAS{NAS{ID: nasID, Identifier: identifier, CIDR: cidr}},
	}
}
