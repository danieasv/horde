package deviceio

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
	"strconv"
	"strings"

	"github.com/ExploratoryEngineering/logging"
)

// UDPParameters holds the command line parameters for the rxtxudp service
type UDPParameters struct {
	Ports         string `param:"desc=Comma-separated list of ports to listen to;default=31415"`
	ListenAddress string `param:"desc=Listen address for UDP;default=127.0.0.1"`
	APNID         int    `param:"desc=APN ID for listener;default=0"`
	NASID         string `param:"desc=NAS ID list for listener;default=0"`
	AuditLog      bool   `param:"desc=Audit log of traffic to and from devices;default=false"`
}

// NASList returns an array of NAS identifiers
func (u *UDPParameters) NASList() []int {
	return splitList(u.NASID)
}

// PortList decodes the port parameter into separate port numbers.
func (u *UDPParameters) PortList() ([]int, error) {
	var ret []int
	for _, v := range strings.Split(u.Ports, ",") {
		val, err := strconv.ParseInt(v, 10, 17)
		if err != nil {
			return nil, err
		}
		ret = append(ret, int(val))
	}
	return ret, nil
}

// CoAPParameters holds the parameters for the CoAP transceivers, ie the
type CoAPParameters struct {
	Endpoint string `param:"desc=CoAP server endpoint;default=127.0.0.1:5683"`
	Protocol string `param:"desc=CoAP server protocol;default=udp"`
	APNID    int    `param:"desc=APN ID for the CoAP server"`
	NASID    string `param:"desc=NAS ID list for the CoAP server;default=0"`
	AuditLog bool   `param:"desc=Audit log for data in/out of the service;default=false"`
}

// NASList returns an array of NAS identifiers
func (c *CoAPParameters) NASList() []int {
	return splitList(c.NASID)
}

func splitList(list string) []int {
	var ret []int
	for _, v := range strings.Split(list, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(v), 10, 32)
		if err != nil {
			logging.Error("Invalid NAS list entry: %s", v)
			continue
		}
		ret = append(ret, int(id))
	}
	return ret
}
